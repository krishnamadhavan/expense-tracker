package imports

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/krishnamadhavan/expense-tracker/internal/adapters/importers"
	"github.com/krishnamadhavan/expense-tracker/internal/adapters/rulesengine"
	"github.com/krishnamadhavan/expense-tracker/internal/app"
	"github.com/krishnamadhavan/expense-tracker/internal/app/transactions"
	"github.com/krishnamadhavan/expense-tracker/internal/domain"
	"github.com/krishnamadhavan/expense-tracker/internal/ports"
)

// Service parses bank/CC statements and commits transactions.
type Service struct {
	Pool *pgxpool.Pool
	Txns *transactions.Service
	// optional categorizer already on Txns
}

func (s *Service) ListFormats() []string {
	var out []string
	for _, p := range importers.All() {
		out = append(out, p.Name())
	}
	return out
}

func (s *Service) AutoDetect(filename string, payload []byte) string {
	return importers.Detect(filename, payload)
}

func (s *Service) Preview(_ context.Context, format, filename string, payload []byte) (ports.ParseResult, error) {
	if format == "" || format == "auto" {
		format = importers.Detect(filename, payload)
	}
	p := importers.ByName(format)
	if p == nil {
		return ports.ParseResult{}, fmt.Errorf("%w: unknown format %q", domain.ErrInvalidArgument, format)
	}
	return p.Parse(filename, payload)
}

func (s *Service) Commit(ctx context.Context, req ports.ImportCommitRequest) (ports.ImportCommitResult, error) {
	if err := app.RequireHousehold(req.HouseholdID); err != nil {
		return ports.ImportCommitResult{}, err
	}
	if req.AccountID == uuid.Nil {
		return ports.ImportCommitResult{}, domain.ErrMissingAccount
	}
	// verify account in household via txn service path on create

	batchID := uuid.New()
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO import_batches (id, household_id, source, status)
		VALUES ($1, $2, $3, 'committed')`, batchID, req.HouseholdID, req.Source+"/"+req.Filename)
	if err != nil {
		return ports.ImportCommitResult{}, err
	}

	out := ports.ImportCommitResult{BatchID: batchID}
	for _, row := range req.Rows {
		ext := row.ExternalRef
		if ext == "" {
			ext = row.TxnDate.Format("2006-01-02") + "|" + string(row.Direction) + "|" + row.Amount.String() + "|" + row.PayeeRaw
		}
		if req.SkipDuplicates {
			var exists bool
			_ = s.Pool.QueryRow(ctx, `
				SELECT EXISTS(SELECT 1 FROM transactions WHERE account_id=$1 AND external_ref=$2)`,
				req.AccountID, ext).Scan(&exists)
			if exists {
				out.SkippedDup++
				continue
			}
		}

		in := transactions.CreateInput{
			HouseholdID: req.HouseholdID,
			AccountID:   req.AccountID,
			Direction:   row.Direction,
			Amount:      row.Amount,
			Currency:    "INR",
			TxnDate:     row.TxnDate,
			PayeeRaw:    row.PayeeRaw,
			Memo:        row.Memo,
		}
		if row.Direction == domain.DirectionIncome {
			if req.DefaultIncomeStreamID == nil || *req.DefaultIncomeStreamID == uuid.Nil {
				out.Failed++
				out.Errors = append(out.Errors, "income row missing default income_stream_id: "+row.PayeeRaw)
				continue
			}
			in.IncomeStreamID = req.DefaultIncomeStreamID
		}

		txn, err := s.Txns.Create(ctx, in)
		if err != nil {
			out.Failed++
			if len(out.Errors) < 50 {
				out.Errors = append(out.Errors, err.Error()+": "+row.PayeeRaw)
			}
			continue
		}
		// set external_ref + import_batch + payee_norm if empty
		norm := txn.PayeeNorm
		if norm == "" {
			norm = rulesengine.NormalizePayee(row.PayeeRaw, nil)
		}
		_, err = s.Pool.Exec(ctx, `
			UPDATE transactions SET external_ref=$2, import_batch_id=$3, source='import', payee_norm=COALESCE(NULLIF(payee_norm,''), $4), updated_at=now()
			WHERE id=$1`, txn.ID, ext, batchID, norm)
		if err != nil {
			// unique violation => dup
			out.SkippedDup++
			_, _ = s.Pool.Exec(ctx, `UPDATE transactions SET voided_at=now() WHERE id=$1`, txn.ID)
			continue
		}
		out.Created++
	}
	_ = time.Now()
	return out, nil
}
