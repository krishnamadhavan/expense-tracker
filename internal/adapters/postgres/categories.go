package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/krishnamadhavan/expense-tracker/internal/domain"
)

// CategoryRepo implements ports.CategoryRepository.
type CategoryRepo struct {
	Pool *pgxpool.Pool
}

func (r *CategoryRepo) GetByID(ctx context.Context, householdID domain.HouseholdID, id domain.CategoryID) (domain.Category, error) {
	const q = `
		SELECT id, household_id, parent_id, name, kind, is_system
		FROM categories
		WHERE household_id = $1 AND id = $2`
	var c domain.Category
	var kind string
	var parent *domain.CategoryID
	err := r.Pool.QueryRow(ctx, q, householdID, id).Scan(
		&c.ID, &c.HouseholdID, &parent, &c.Name, &kind, &c.IsSystem,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Category{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Category{}, err
	}
	c.ParentID = parent
	c.Kind = domain.CategoryKind(kind)
	return c, nil
}

func (r *CategoryRepo) List(ctx context.Context, householdID domain.HouseholdID) ([]domain.Category, error) {
	const q = `
		SELECT id, household_id, parent_id, name, kind, is_system
		FROM categories
		WHERE household_id = $1
		ORDER BY kind, name`
	rows, err := r.Pool.Query(ctx, q, householdID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Category
	for rows.Next() {
		var c domain.Category
		var kind string
		var parent *domain.CategoryID
		if err := rows.Scan(&c.ID, &c.HouseholdID, &parent, &c.Name, &kind, &c.IsSystem); err != nil {
			return nil, err
		}
		c.ParentID = parent
		c.Kind = domain.CategoryKind(kind)
		out = append(out, c)
	}
	return out, rows.Err()
}
