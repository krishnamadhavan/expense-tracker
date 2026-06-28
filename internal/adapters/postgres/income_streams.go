package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/krishnamadhavan/expense-tracker/internal/domain"
)

// IncomeStreamRepo implements ports.IncomeStreamRepository.
type IncomeStreamRepo struct {
	Pool *pgxpool.Pool
}

func (r *IncomeStreamRepo) GetByID(ctx context.Context, householdID domain.HouseholdID, id domain.IncomeStreamID) (domain.IncomeStream, error) {
	const q = `
		SELECT id, household_id, name, code
		FROM income_streams
		WHERE household_id = $1 AND id = $2`
	var s domain.IncomeStream
	err := r.Pool.QueryRow(ctx, q, householdID, id).Scan(&s.ID, &s.HouseholdID, &s.Name, &s.Code)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.IncomeStream{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.IncomeStream{}, err
	}
	return s, nil
}

func (r *IncomeStreamRepo) List(ctx context.Context, householdID domain.HouseholdID) ([]domain.IncomeStream, error) {
	const q = `
		SELECT id, household_id, name, code
		FROM income_streams
		WHERE household_id = $1
		ORDER BY code`
	rows, err := r.Pool.Query(ctx, q, householdID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.IncomeStream
	for rows.Next() {
		var s domain.IncomeStream
		if err := rows.Scan(&s.ID, &s.HouseholdID, &s.Name, &s.Code); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
