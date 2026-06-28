package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/krishnamadhavan/expense-tracker/internal/domain"
)

// TransactionRepo implements ports.TransactionRepository.
type TransactionRepo struct {
	Pool *pgxpool.Pool
}

func (r *TransactionRepo) Create(ctx context.Context, txn domain.Transaction) (domain.Transaction, error) {
	const q = `
		INSERT INTO transactions (
			id, household_id, account_id, transfer_account_id, category_id, income_stream_id,
			direction, amount, currency, txn_date, payee_raw, payee_norm, memo,
			category_confidence, category_locked, created_at, updated_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17
		)
		RETURNING id, household_id, account_id, transfer_account_id, category_id, income_stream_id,
			direction, amount, currency, txn_date, payee_raw, payee_norm, memo,
			category_confidence, category_locked, voided_at, created_at, updated_at`
	amountMajor := float64(txn.Amount.Minor) / 100.0
	row := r.Pool.QueryRow(ctx, q,
		txn.ID, txn.HouseholdID, txn.AccountID, txn.TransferAccountID, txn.CategoryID, txn.IncomeStreamID,
		string(txn.Direction), amountMajor, txn.Currency, txn.TxnDate,
		nullStr(txn.PayeeRaw), nullStr(txn.PayeeNorm), nullStr(txn.Memo),
		txn.CategoryConfidence, txn.CategoryLocked, txn.CreatedAt, txn.UpdatedAt,
	)
	return scanTxn(row)
}

func (r *TransactionRepo) GetByID(ctx context.Context, householdID domain.HouseholdID, id domain.TransactionID) (domain.Transaction, error) {
	const q = `
		SELECT id, household_id, account_id, transfer_account_id, category_id, income_stream_id,
			direction, amount, currency, txn_date, payee_raw, payee_norm, memo,
			category_confidence, category_locked, voided_at, created_at, updated_at
		FROM transactions
		WHERE household_id = $1 AND id = $2`
	return scanTxn(r.Pool.QueryRow(ctx, q, householdID, id))
}

func (r *TransactionRepo) List(ctx context.Context, householdID domain.HouseholdID, limit int) ([]domain.Transaction, error) {
	const q = `
		SELECT id, household_id, account_id, transfer_account_id, category_id, income_stream_id,
			direction, amount, currency, txn_date, payee_raw, payee_norm, memo,
			category_confidence, category_locked, voided_at, created_at, updated_at
		FROM transactions
		WHERE household_id = $1
		ORDER BY txn_date DESC, id DESC
		LIMIT $2`
	rows, err := r.Pool.Query(ctx, q, householdID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Transaction
	for rows.Next() {
		t, err := scanTxnRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *TransactionRepo) Void(ctx context.Context, householdID domain.HouseholdID, id domain.TransactionID) (domain.Transaction, error) {
	const q = `
		UPDATE transactions
		SET voided_at = now(), updated_at = now()
		WHERE household_id = $1 AND id = $2 AND voided_at IS NULL
		RETURNING id, household_id, account_id, transfer_account_id, category_id, income_stream_id,
			direction, amount, currency, txn_date, payee_raw, payee_norm, memo,
			category_confidence, category_locked, voided_at, created_at, updated_at`
	t, err := scanTxn(r.Pool.QueryRow(ctx, q, householdID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		// already voided or missing — try get
		existing, gerr := r.GetByID(ctx, householdID, id)
		if gerr != nil {
			return domain.Transaction{}, gerr
		}
		if existing.IsVoided() {
			return domain.Transaction{}, domain.ErrVoidedTransaction
		}
		return domain.Transaction{}, domain.ErrNotFound
	}
	return t, err
}

type scannable interface {
	Scan(dest ...any) error
}

func scanTxn(row scannable) (domain.Transaction, error) {
	var t domain.Transaction
	var dir string
	var amount float64
	var payeeRaw, payeeNorm, memo *string
	var voided *time.Time
	err := row.Scan(
		&t.ID, &t.HouseholdID, &t.AccountID, &t.TransferAccountID, &t.CategoryID, &t.IncomeStreamID,
		&dir, &amount, &t.Currency, &t.TxnDate, &payeeRaw, &payeeNorm, &memo,
		&t.CategoryConfidence, &t.CategoryLocked, &voided, &t.CreatedAt, &t.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Transaction{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Transaction{}, err
	}
	t.Direction = domain.Direction(dir)
	// convert amount to minor units (2 dp)
	minor := int64(amount*100 + 0.5)
	m, err := domain.MoneyFromMinor(minor)
	if err != nil {
		return domain.Transaction{}, err
	}
	t.Amount = m
	if payeeRaw != nil {
		t.PayeeRaw = *payeeRaw
	}
	if payeeNorm != nil {
		t.PayeeNorm = *payeeNorm
	}
	if memo != nil {
		t.Memo = *memo
	}
	t.VoidedAt = voided
	return t, nil
}

func scanTxnRows(rows pgx.Rows) (domain.Transaction, error) {
	return scanTxn(rows)
}

func nullStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
