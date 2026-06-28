package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/krishnamadhavan/expense-tracker/internal/domain"
)

// AccountRepo implements ports.AccountRepository.
type AccountRepo struct {
	Pool *pgxpool.Pool
}

func (r *AccountRepo) GetByID(ctx context.Context, householdID domain.HouseholdID, id domain.AccountID) (domain.Account, error) {
	const q = `
		SELECT id, household_id, name, type, currency, is_active
		FROM accounts
		WHERE household_id = $1 AND id = $2`
	var a domain.Account
	var typ string
	err := r.Pool.QueryRow(ctx, q, householdID, id).Scan(
		&a.ID, &a.HouseholdID, &a.Name, &typ, &a.Currency, &a.IsActive,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Account{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Account{}, err
	}
	a.Type = domain.AccountType(typ)
	return a, nil
}

func (r *AccountRepo) List(ctx context.Context, householdID domain.HouseholdID) ([]domain.Account, error) {
	const q = `
		SELECT id, household_id, name, type, currency, is_active
		FROM accounts
		WHERE household_id = $1
		ORDER BY name`
	rows, err := r.Pool.Query(ctx, q, householdID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Account
	for rows.Next() {
		var a domain.Account
		var typ string
		if err := rows.Scan(&a.ID, &a.HouseholdID, &a.Name, &typ, &a.Currency, &a.IsActive); err != nil {
			return nil, err
		}
		a.Type = domain.AccountType(typ)
		out = append(out, a)
	}
	return out, rows.Err()
}

// InsertTestAccount is used by integration tests only.
func (r *AccountRepo) InsertTestAccount(ctx context.Context, householdID uuid.UUID, name string, typ domain.AccountType) (domain.Account, error) {
	const q = `
		INSERT INTO accounts (household_id, name, type, currency, is_active)
		VALUES ($1, $2, $3, 'INR', true)
		RETURNING id, household_id, name, type, currency, is_active`
	var a domain.Account
	var t string
	err := r.Pool.QueryRow(ctx, q, householdID, name, string(typ)).Scan(
		&a.ID, &a.HouseholdID, &a.Name, &t, &a.Currency, &a.IsActive,
	)
	if err != nil {
		return domain.Account{}, fmt.Errorf("insert account: %w", err)
	}
	a.Type = domain.AccountType(t)
	return a, nil
}
