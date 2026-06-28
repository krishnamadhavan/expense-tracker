package categorization

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/krishnamadhavan/expense-tracker/internal/adapters/rulesengine"
	"github.com/krishnamadhavan/expense-tracker/internal/app"
	"github.com/krishnamadhavan/expense-tracker/internal/domain"
	"github.com/krishnamadhavan/expense-tracker/internal/ports"
)

// Service owns suggest helpers and Moderate learning path.
type Service struct {
	Pool   *pgxpool.Pool
	Engine *rulesengine.Engine
	Cats   ports.CategoryRepository
}

// AttachSuggestion applies engine result to a created transaction (events + optional review queue).
func (s *Service) AttachSuggestion(ctx context.Context, householdID domain.HouseholdID, txn domain.Transaction, draft domain.TransactionDraft, userSetCategory bool) error {
	if s.Engine == nil {
		return nil
	}
	_, detailed, err := s.Engine.SuggestDetailed(ctx, householdID, draft)
	if err != nil {
		return err
	}
	norm := rulesengine.NormalizePayee(draft.PayeeRaw, s.Engine.PSPSuffixes)
	var suggested *domain.CategoryID
	var conf *float64
	var ruleID *domain.CategoryRuleID
	if detailed.CategoryID != uuid.Nil {
		cid := detailed.CategoryID
		suggested = &cid
		c := detailed.Confidence
		conf = &c
		ruleID = detailed.RuleID
	}
	outcome := "pending"
	if userSetCategory {
		outcome = "accepted" // user provided category on create
	}
	st := &pgCat{Pool: s.Pool}
	if err := st.InsertCategorizationEvent(ctx, txn.ID, suggested, ruleID, conf, outcome); err != nil {
		return err
	}
	// Review queue when not auto and user did not set category
	auto := conf != nil && *conf >= rulesengine.ThresholdAuto && !detailed.Conflict
	if userSetCategory || auto {
		return nil
	}
	reason := "low_confidence"
	if detailed.Conflict {
		reason = "conflict"
	} else if norm == "" {
		reason = "null_payee"
	} else if detailed.FromMerchantNorm {
		reason = "low_confidence"
	} else if suggested == nil || *suggested == uuid.Nil {
		// check new merchant: no rule, no merchant row — FromMerchantNorm false and conf 0
		if norm != "" && (conf == nil || *conf == 0) {
			reason = "new_merchant"
		}
	}
	return st.UpsertReviewQueue(ctx, txn.ID, reason)
}

// ModerateInput is the moderation command.
type ModerateInput struct {
	HouseholdID  domain.HouseholdID
	UserID       domain.UserID
	TransactionID domain.TransactionID
	ToCategoryID domain.CategoryID
}

// ModerateResult summarizes learning outcome.
type ModerateResult struct {
	Transaction domain.Transaction
	Learning    string // boost | inserted | deferred | deactivated_insert | user_rule_protected | skipped_null_payee
}

// Moderate applies category and learning algorithm in one DB transaction.
func (s *Service) Moderate(ctx context.Context, in ModerateInput) (ModerateResult, error) {
	if err := app.RequireHousehold(in.HouseholdID); err != nil {
		return ModerateResult{}, err
	}
	// validate category in household
	cat, err := s.Cats.GetByID(ctx, in.HouseholdID, in.ToCategoryID)
	if err != nil {
		return ModerateResult{}, err
	}

	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return ModerateResult{}, err
	}
	defer tx.Rollback(ctx)

	var txn domain.Transaction
	var dir string
	var amount float64
	var payeeNorm *string
	var fromCat *uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT id, household_id, account_id, transfer_account_id, category_id, income_stream_id,
		       direction, amount, currency, txn_date, payee_raw, payee_norm, memo,
		       category_confidence, category_locked, voided_at, created_at, updated_at
		FROM transactions WHERE household_id = $1 AND id = $2 FOR UPDATE`,
		in.HouseholdID, in.TransactionID,
	).Scan(
		&txn.ID, &txn.HouseholdID, &txn.AccountID, &txn.TransferAccountID, &fromCat, &txn.IncomeStreamID,
		&dir, &amount, &txn.Currency, &txn.TxnDate, &txn.PayeeRaw, &payeeNorm, &txn.Memo,
		&txn.CategoryConfidence, &txn.CategoryLocked, &txn.VoidedAt, &txn.CreatedAt, &txn.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ModerateResult{}, domain.ErrNotFound
	}
	if err != nil {
		return ModerateResult{}, err
	}
	txn.Direction = domain.Direction(dir)
	minor := int64(amount*100 + 0.5)
	txn.Amount, _ = domain.MoneyFromMinor(minor)
	if payeeNorm != nil {
		txn.PayeeNorm = *payeeNorm
	}
	if fromCat != nil {
		id := domain.CategoryID(*fromCat)
		txn.CategoryID = &id
	}

	// category kind vs direction
	kind := cat.Kind
	if err := domain.ValidateCategoryForDirection(txn.Direction, &in.ToCategoryID, &kind); err != nil {
		return ModerateResult{}, err
	}

	conf := 1.0
	_, err = tx.Exec(ctx, `
		UPDATE transactions SET category_id = $3, category_locked = true, category_confidence = $4, updated_at = now()
		WHERE household_id = $1 AND id = $2`,
		in.HouseholdID, in.TransactionID, in.ToCategoryID, conf)
	if err != nil {
		return ModerateResult{}, err
	}

	var fromPtr *uuid.UUID
	if txn.CategoryID != nil {
		u := uuid.UUID(*txn.CategoryID)
		fromPtr = &u
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO moderation_events (transaction_id, user_id, from_category_id, to_category_id)
		VALUES ($1, $2, $3, $4)`, in.TransactionID, in.UserID, fromPtr, in.ToCategoryID)
	if err != nil {
		return ModerateResult{}, err
	}

	// categorization_events outcome
	var sug *uuid.UUID
	_ = tx.QueryRow(ctx, `
		SELECT suggested_category_id FROM categorization_events
		WHERE transaction_id = $1 ORDER BY created_at DESC LIMIT 1`, in.TransactionID).Scan(&sug)
	outcome := "accepted"
	if sug != nil && *sug != uuid.UUID(in.ToCategoryID) {
		outcome = "overridden"
	}
	tag, err := tx.Exec(ctx, `
		UPDATE categorization_events SET outcome = $2
		WHERE id = (
		  SELECT id FROM categorization_events WHERE transaction_id = $1 ORDER BY created_at DESC LIMIT 1
		)`, in.TransactionID, outcome)
	if err != nil {
		return ModerateResult{}, err
	}
	if tag.RowsAffected() == 0 {
		_, _ = tx.Exec(ctx, `
			INSERT INTO categorization_events (transaction_id, suggested_category_id, confidence, outcome)
			VALUES ($1, $2, 1.0, 'accepted')`, in.TransactionID, in.ToCategoryID)
	}

	_, _ = tx.Exec(ctx, `
		UPDATE review_queue SET status = 'resolved', resolved_at = now()
		WHERE transaction_id = $1 AND status = 'open'`, in.TransactionID)

	learning := "skipped_null_payee"
	norm := txn.PayeeNorm
	if norm == "" {
		norm = rulesengine.NormalizePayee(txn.PayeeRaw, nil)
	}
	if norm != "" {
		_, _ = tx.Exec(ctx, `UPDATE transactions SET payee_norm = $2 WHERE id = $1`, in.TransactionID, norm)
		learning, err = learnRules(ctx, tx, in.HouseholdID, norm, in.ToCategoryID)
		if err != nil {
			return ModerateResult{}, err
		}
		// merchant_norms upsert always when norm present
		_, err = tx.Exec(ctx, `
			INSERT INTO merchant_norms (household_id, norm_key, display_name, default_category_id, updated_at)
			VALUES ($1, $2, $2, $3, now())
			ON CONFLICT (household_id, norm_key) DO UPDATE
			SET default_category_id = EXCLUDED.default_category_id, updated_at = now()`,
			in.HouseholdID, norm, in.ToCategoryID)
		if err != nil {
			return ModerateResult{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return ModerateResult{}, err
	}

	txn.CategoryID = &in.ToCategoryID
	txn.CategoryLocked = true
	txn.CategoryConfidence = &conf
	txn.PayeeNorm = norm
	return ModerateResult{Transaction: txn, Learning: learning}, nil
}

func learnRules(ctx context.Context, tx pgx.Tx, hh domain.HouseholdID, norm string, to domain.CategoryID) (string, error) {
	// lock active exact rule
	var ruleID *uuid.UUID
	var ruleCat uuid.UUID
	var origin string
	var conf float64
	var hits int
	err := tx.QueryRow(ctx, `
		SELECT id, category_id, origin, confidence, hit_count FROM category_rules
		WHERE household_id = $1 AND match_field = 'payee_norm' AND match_type = 'exact'
		  AND pattern = $2 AND is_active = true
		FOR UPDATE`, hh, norm).Scan(&ruleID, &ruleCat, &origin, &conf, &hits)
	if errors.Is(err, pgx.ErrNoRows) {
		_, err = tx.Exec(ctx, `
			INSERT INTO category_rules (household_id, category_id, match_field, match_type, pattern, priority, confidence, origin, hit_count, is_active)
			VALUES ($1, $2, 'payee_norm', 'exact', $3, 50, 0.75, 'learned', 1, true)`,
			hh, to, norm)
		return "inserted", err
	}
	if err != nil {
		return "", err
	}

	if ruleCat == uuid.UUID(to) {
		_, err = tx.Exec(ctx, `
			UPDATE category_rules SET hit_count = hit_count + 1,
			  confidence = LEAST(0.99, confidence + 0.01), updated_at = now()
			WHERE id = $1`, *ruleID)
		return "boost", err
	}

	// conflict count last 90 days for this payee_norm where to_category != ruleCat
	// include current moderation: count prior mismatches + 1
	var prior int
	_ = tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM moderation_events me
		JOIN transactions t ON t.id = me.transaction_id
		WHERE t.household_id = $1 AND t.payee_norm = $2
		  AND me.to_category_id <> $3
		  AND me.created_at >= now() - interval '90 days'`,
		hh, norm, ruleCat).Scan(&prior)
	// current event already inserted with to != ruleCat, so prior includes it if payee_norm was set on txn.
	// Ensure txn payee_norm updated — set on txn before learn? We should set payee_norm on transaction if empty
	_, _ = tx.Exec(ctx, `UPDATE transactions SET payee_norm = $2 WHERE id = (
		SELECT id FROM transactions WHERE household_id = $1 AND payee_norm = $2 LIMIT 0)`, hh, norm)
	// recount including moderations linked to txns with this norm — current moderation's txn may have empty payee_norm
	// Update payee_norm on moderated txn first in Moderate before learn — do in Moderate before learnRules call

	conflictCount := prior // after insert, if payee_norm on txn matches, prior includes current
	// Force at least 1 for this mismatch event
	if conflictCount < 1 {
		conflictCount = 1
	}

	if conflictCount == 1 {
		return "deferred", nil
	}

	if origin == "user" {
		return "user_rule_protected", nil
	}

	// deactivate + insert learned
	_, err = tx.Exec(ctx, `UPDATE category_rules SET is_active = false, updated_at = now() WHERE id = $1`, *ruleID)
	if err != nil {
		return "", err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO category_rules (household_id, category_id, match_field, match_type, pattern, priority, confidence, origin, hit_count, is_active)
		VALUES ($1, $2, 'payee_norm', 'exact', $3, 50, 0.75, 'learned', 1, true)`,
		hh, to, norm)
	return "deactivated_insert", err
}

// pgCat thin helpers
type pgCat struct{ Pool *pgxpool.Pool }

func (p *pgCat) InsertCategorizationEvent(ctx context.Context, txnID domain.TransactionID, suggested *domain.CategoryID, ruleID *domain.CategoryRuleID, confidence *float64, outcome string) error {
	_, err := p.Pool.Exec(ctx, `
		INSERT INTO categorization_events (transaction_id, suggested_category_id, rule_id, confidence, outcome)
		VALUES ($1, $2, $3, $4, $5)`, txnID, suggested, ruleID, confidence, outcome)
	return err
}

func (p *pgCat) UpsertReviewQueue(ctx context.Context, txnID domain.TransactionID, reason string) error {
	_, err := p.Pool.Exec(ctx, `
		INSERT INTO review_queue (transaction_id, reason, status) VALUES ($1, $2, 'open')
		ON CONFLICT (transaction_id) DO NOTHING`, txnID, reason)
	return err
}

// QualityReport is a simple accuracy snapshot.
type QualityReport struct {
	TotalEvents   int     `json:"total_events"`
	Accepted      int     `json:"accepted"`
	Overridden    int     `json:"overridden"`
	Pending       int     `json:"pending"`
	AcceptRate    float64 `json:"accept_rate"`
	OpenReviews   int     `json:"open_reviews"`
}

// Quality returns household categorization quality metrics.
func (s *Service) Quality(ctx context.Context, householdID domain.HouseholdID) (QualityReport, error) {
	if err := app.RequireHousehold(householdID); err != nil {
		return QualityReport{}, err
	}
	var q QualityReport
	err := s.Pool.QueryRow(ctx, `
		SELECT
		  COUNT(*)::int,
		  COUNT(*) FILTER (WHERE outcome = 'accepted')::int,
		  COUNT(*) FILTER (WHERE outcome = 'overridden')::int,
		  COUNT(*) FILTER (WHERE outcome = 'pending')::int
		FROM categorization_events ce
		JOIN transactions t ON t.id = ce.transaction_id
		WHERE t.household_id = $1`, householdID).Scan(&q.TotalEvents, &q.Accepted, &q.Overridden, &q.Pending)
	if err != nil {
		return q, err
	}
	decided := q.Accepted + q.Overridden
	if decided > 0 {
		q.AcceptRate = float64(q.Accepted) / float64(decided)
	}
	_ = s.Pool.QueryRow(ctx, `
		SELECT COUNT(*)::int FROM review_queue rq
		JOIN transactions t ON t.id = rq.transaction_id
		WHERE t.household_id = $1 AND rq.status = 'open'`, householdID).Scan(&q.OpenReviews)
	return q, nil
}

// ListOpenReviews returns open review queue items for household.
func (s *Service) ListOpenReviews(ctx context.Context, householdID domain.HouseholdID) ([]map[string]any, error) {
	if err := app.RequireHousehold(householdID); err != nil {
		return nil, err
	}
	rows, err := s.Pool.Query(ctx, `
		SELECT rq.id, rq.transaction_id, rq.reason, rq.created_at, t.payee_raw, t.amount, t.direction
		FROM review_queue rq
		JOIN transactions t ON t.id = rq.transaction_id
		WHERE t.household_id = $1 AND rq.status = 'open'
		ORDER BY rq.created_at`, householdID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id, txnID uuid.UUID
		var reason, payee, dir string
		var created time.Time
		var amount float64
		if err := rows.Scan(&id, &txnID, &reason, &created, &payee, &amount, &dir); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"id": id, "transaction_id": txnID, "reason": reason, "created_at": created,
			"payee_raw": payee, "amount": fmt.Sprintf("%.2f", amount), "direction": dir,
		})
	}
	return out, rows.Err()
}
