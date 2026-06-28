package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/krishnamadhavan/expense-tracker/internal/domain"
)

// CategorizationStore implements rulesengine.RuleStore and learning persistence.
type CategorizationStore struct {
	Pool *pgxpool.Pool
}

func (s *CategorizationStore) ListActiveRules(ctx context.Context, householdID domain.HouseholdID) ([]domain.CategoryRule, error) {
	const q = `
		SELECT id, household_id, category_id, match_field, match_type, pattern,
		       priority, confidence, origin, hit_count, is_active
		FROM category_rules
		WHERE household_id = $1 AND is_active = true`
	rows, err := s.Pool.Query(ctx, q, householdID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.CategoryRule
	for rows.Next() {
		var r domain.CategoryRule
		var mt, origin string
		if err := rows.Scan(&r.ID, &r.HouseholdID, &r.CategoryID, &r.MatchField, &mt, &r.Pattern,
			&r.Priority, &r.Confidence, &origin, &r.HitCount, &r.IsActive); err != nil {
			return nil, err
		}
		r.MatchType = domain.MatchType(mt)
		r.Origin = domain.RuleOrigin(origin)
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *CategorizationStore) GetMerchantNorm(ctx context.Context, householdID domain.HouseholdID, normKey string) (*domain.CategoryID, bool, error) {
	const q = `SELECT default_category_id FROM merchant_norms WHERE household_id = $1 AND norm_key = $2`
	var cat *uuid.UUID
	err := s.Pool.QueryRow(ctx, q, householdID, normKey).Scan(&cat)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if cat == nil {
		return nil, true, nil
	}
	id := domain.CategoryID(*cat)
	return &id, true, nil
}

// InsertCategorizationEvent records a suggestion audit row.
func (s *CategorizationStore) InsertCategorizationEvent(ctx context.Context, txnID domain.TransactionID, suggested *domain.CategoryID, ruleID *domain.CategoryRuleID, confidence *float64, outcome string) error {
	const q = `
		INSERT INTO categorization_events (transaction_id, suggested_category_id, rule_id, confidence, outcome)
		VALUES ($1, $2, $3, $4, $5)`
	_, err := s.Pool.Exec(ctx, q, txnID, suggested, ruleID, confidence, outcome)
	return err
}

// UpsertReviewQueue opens or ignores existing open row.
func (s *CategorizationStore) UpsertReviewQueue(ctx context.Context, txnID domain.TransactionID, reason string) error {
	const q = `
		INSERT INTO review_queue (transaction_id, reason, status)
		VALUES ($1, $2, 'open')
		ON CONFLICT (transaction_id) DO UPDATE
		SET reason = EXCLUDED.reason, status = 'open', resolved_at = NULL
		WHERE review_queue.status = 'resolved' OR review_queue.status = 'open'`
	// simpler: only insert if not exists
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO review_queue (transaction_id, reason, status)
		VALUES ($1, $2, 'open')
		ON CONFLICT (transaction_id) DO NOTHING`, txnID, reason)
	return err
}

// BeginTx starts a transaction.
func (s *CategorizationStore) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return s.Pool.Begin(ctx)
}
