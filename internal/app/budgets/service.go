package budgets

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/krishnamadhavan/expense-tracker/internal/app"
	"github.com/krishnamadhavan/expense-tracker/internal/domain"
)

// Service CRUD for budgets.
type Service struct {
	Pool *pgxpool.Pool
}

// BudgetDTO is API-facing.
type BudgetDTO struct {
	ID          uuid.UUID `json:"id"`
	CategoryID  uuid.UUID `json:"category_id"`
	PeriodType  string    `json:"period_type"`
	PeriodStart string    `json:"period_start"`
	AmountLimit string    `json:"amount_limit"`
	Currency    string    `json:"currency"`
}

func (s *Service) List(ctx context.Context, hh domain.HouseholdID) ([]BudgetDTO, error) {
	if err := app.RequireHousehold(hh); err != nil {
		return nil, err
	}
	rows, err := s.Pool.Query(ctx, `
		SELECT id, category_id, period_type, period_start, amount_limit, currency
		FROM budgets WHERE household_id=$1 ORDER BY period_start DESC`, hh)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BudgetDTO
	for rows.Next() {
		var b BudgetDTO
		var start time.Time
		var amt float64
		if err := rows.Scan(&b.ID, &b.CategoryID, &b.PeriodType, &start, &amt, &b.Currency); err != nil {
			return nil, err
		}
		b.PeriodStart = start.Format("2006-01-02")
		m, _ := domain.MoneyFromMinor(int64(amt*100 + 0.5))
		b.AmountLimit = m.String()
		out = append(out, b)
	}
	return out, rows.Err()
}

func (s *Service) Create(ctx context.Context, hh domain.HouseholdID, categoryID uuid.UUID, periodType, periodStart, amount, currency string) (BudgetDTO, error) {
	if err := app.RequireHousehold(hh); err != nil {
		return BudgetDTO{}, err
	}
	if periodType != "month" && periodType != "fy" {
		return BudgetDTO{}, domain.ErrInvalidArgument
	}
	mon, err := domain.ParseMoney(amount)
	if err != nil {
		return BudgetDTO{}, err
	}
	if currency == "" {
		currency = "INR"
	}
	start, err := time.Parse("2006-01-02", periodStart)
	if err != nil {
		return BudgetDTO{}, domain.ErrInvalidArgument
	}
	var b BudgetDTO
	var st time.Time
	var amt float64
	err = s.Pool.QueryRow(ctx, `
		INSERT INTO budgets (household_id, category_id, period_type, period_start, amount_limit, currency)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id, category_id, period_type, period_start, amount_limit, currency`,
		hh, categoryID, periodType, start, float64(mon.Minor)/100.0, currency,
	).Scan(&b.ID, &b.CategoryID, &b.PeriodType, &st, &amt, &b.Currency)
	if err != nil {
		return BudgetDTO{}, err
	}
	b.PeriodStart = st.Format("2006-01-02")
	m, _ := domain.MoneyFromMinor(int64(amt*100 + 0.5))
	b.AmountLimit = m.String()
	return b, nil
}
