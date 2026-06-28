package reports

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/krishnamadhavan/expense-tracker/internal/app"
	"github.com/krishnamadhavan/expense-tracker/internal/domain"
)

// Service builds P&L and breakdown series (transfers excluded from P&L).
type Service struct {
	Pool *pgxpool.Pool
}

// Summary is income vs expense for a half-open window.
type Summary struct {
	Income  string `json:"income"`
	Expense string `json:"expense"`
	Net     string `json:"net"`
	Start   string `json:"start"`
	End     string `json:"end_exclusive"`
	Label   string `json:"label,omitempty"`
}

// BreakdownRow is a labeled total.
type BreakdownRow struct {
	ID     string `json:"id,omitempty"`
	Name   string `json:"name"`
	Amount string `json:"amount"`
}

// Point is a timeseries sample.
type Point struct {
	Date    string `json:"date"`
	Income  string `json:"income"`
	Expense string `json:"expense"`
}

func moneyStr(minor int64) string {
	m, _ := domain.MoneyFromMinor(minor)
	return m.String()
}

func scanSum(pool *pgxpool.Pool, ctx context.Context, q string, args ...any) (int64, error) {
	var f float64
	err := pool.QueryRow(ctx, q, args...).Scan(&f)
	return int64(f*100 + 0.5), err
}

// RangeSummary computes P&L for [start, end).
func (s *Service) RangeSummary(ctx context.Context, hh domain.HouseholdID, start, end time.Time, label string) (Summary, error) {
	if err := app.RequireHousehold(hh); err != nil {
		return Summary{}, err
	}
	inc, err := scanSum(s.Pool, ctx, `
		SELECT COALESCE(SUM(amount),0) FROM transactions
		WHERE household_id=$1 AND voided_at IS NULL AND direction='income'
		  AND txn_date >= $2::date AND txn_date < $3::date`, hh, start, end)
	if err != nil {
		return Summary{}, err
	}
	exp, err := scanSum(s.Pool, ctx, `
		SELECT COALESCE(SUM(amount),0) FROM transactions
		WHERE household_id=$1 AND voided_at IS NULL AND direction='expense'
		  AND txn_date >= $2::date AND txn_date < $3::date`, hh, start, end)
	if err != nil {
		return Summary{}, err
	}
	return Summary{
		Income: moneyStr(inc), Expense: moneyStr(exp), Net: moneyStr(inc - exp),
		Start: start.Format("2006-01-02"), End: end.Format("2006-01-02"), Label: label,
	}, nil
}

// ByCategory expense or income breakdown.
func (s *Service) ByCategory(ctx context.Context, hh domain.HouseholdID, start, end time.Time, direction domain.Direction) ([]BreakdownRow, error) {
	if err := app.RequireHousehold(hh); err != nil {
		return nil, err
	}
	rows, err := s.Pool.Query(ctx, `
		SELECT COALESCE(c.id::text,''), COALESCE(c.name,'(uncategorized)'), COALESCE(SUM(t.amount),0)
		FROM transactions t
		LEFT JOIN categories c ON c.id = t.category_id
		WHERE t.household_id=$1 AND t.voided_at IS NULL AND t.direction=$2
		  AND t.txn_date >= $3::date AND t.txn_date < $4::date
		GROUP BY c.id, c.name
		ORDER BY SUM(t.amount) DESC`, hh, string(direction), start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BreakdownRow
	for rows.Next() {
		var id, name string
		var amt float64
		if err := rows.Scan(&id, &name, &amt); err != nil {
			return nil, err
		}
		out = append(out, BreakdownRow{ID: id, Name: name, Amount: moneyStr(int64(amt*100 + 0.5))})
	}
	return out, rows.Err()
}

// ByAccount channel activity (expenses + transfer outflows on account).
func (s *Service) ByAccount(ctx context.Context, hh domain.HouseholdID, start, end time.Time) ([]BreakdownRow, error) {
	if err := app.RequireHousehold(hh); err != nil {
		return nil, err
	}
	rows, err := s.Pool.Query(ctx, `
		SELECT a.id::text, a.name, COALESCE(SUM(
		  CASE WHEN t.direction='expense' OR (t.direction='transfer' AND t.account_id=a.id) THEN t.amount ELSE 0 END
		),0)
		FROM accounts a
		LEFT JOIN transactions t ON t.household_id=a.household_id AND t.voided_at IS NULL
		  AND t.txn_date >= $2::date AND t.txn_date < $3::date
		  AND (t.account_id=a.id OR t.transfer_account_id=a.id)
		WHERE a.household_id=$1
		GROUP BY a.id, a.name
		ORDER BY a.name`, hh, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BreakdownRow
	for rows.Next() {
		var id, name string
		var amt float64
		if err := rows.Scan(&id, &name, &amt); err != nil {
			return nil, err
		}
		out = append(out, BreakdownRow{ID: id, Name: name, Amount: moneyStr(int64(amt*100 + 0.5))})
	}
	return out, rows.Err()
}

// ByIncomeStream for income direction.
func (s *Service) ByIncomeStream(ctx context.Context, hh domain.HouseholdID, start, end time.Time) ([]BreakdownRow, error) {
	if err := app.RequireHousehold(hh); err != nil {
		return nil, err
	}
	rows, err := s.Pool.Query(ctx, `
		SELECT s.id::text, s.name, COALESCE(SUM(t.amount),0)
		FROM income_streams s
		LEFT JOIN transactions t ON t.income_stream_id=s.id AND t.voided_at IS NULL
		  AND t.direction='income' AND t.txn_date >= $2::date AND t.txn_date < $3::date
		WHERE s.household_id=$1
		GROUP BY s.id, s.name
		ORDER BY s.name`, hh, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BreakdownRow
	for rows.Next() {
		var id, name string
		var amt float64
		if err := rows.Scan(&id, &name, &amt); err != nil {
			return nil, err
		}
		out = append(out, BreakdownRow{ID: id, Name: name, Amount: moneyStr(int64(amt*100 + 0.5))})
	}
	return out, rows.Err()
}

// DailyTimeseries income/expense by day.
func (s *Service) DailyTimeseries(ctx context.Context, hh domain.HouseholdID, start, end time.Time) ([]Point, error) {
	if err := app.RequireHousehold(hh); err != nil {
		return nil, err
	}
	rows, err := s.Pool.Query(ctx, `
		SELECT txn_date::text,
		  COALESCE(SUM(amount) FILTER (WHERE direction='income'),0),
		  COALESCE(SUM(amount) FILTER (WHERE direction='expense'),0)
		FROM transactions
		WHERE household_id=$1 AND voided_at IS NULL
		  AND txn_date >= $2::date AND txn_date < $3::date
		  AND direction IN ('income','expense')
		GROUP BY txn_date
		ORDER BY txn_date`, hh, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Point
	for rows.Next() {
		var d string
		var inc, exp float64
		if err := rows.Scan(&d, &inc, &exp); err != nil {
			return nil, err
		}
		out = append(out, Point{
			Date: d, Income: moneyStr(int64(inc*100 + 0.5)), Expense: moneyStr(int64(exp*100 + 0.5)),
		})
	}
	return out, rows.Err()
}

// ExportCSV returns transaction rows as CSV bytes for the range.
func (s *Service) ExportCSV(ctx context.Context, hh domain.HouseholdID, start, end time.Time) ([]byte, error) {
	if err := app.RequireHousehold(hh); err != nil {
		return nil, err
	}
	rows, err := s.Pool.Query(ctx, `
		SELECT txn_date, direction, amount, currency, payee_raw, memo, voided_at IS NOT NULL
		FROM transactions WHERE household_id=$1
		  AND txn_date >= $2::date AND txn_date < $3::date
		ORDER BY txn_date, id`, hh, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	buf := []byte("txn_date,direction,amount,currency,payee_raw,memo,voided\n")
	for rows.Next() {
		var d time.Time
		var dir, cur, payee, memo string
		var amt float64
		var voided bool
		var payeeN, memoN *string
		if err := rows.Scan(&d, &dir, &amt, &cur, &payeeN, &memoN, &voided); err != nil {
			return nil, err
		}
		if payeeN != nil {
			payee = *payeeN
		}
		if memoN != nil {
			memo = *memoN
		}
		line := d.Format("2006-01-02") + "," + dir + "," + moneyStr(int64(amt*100+0.5)) + "," + cur + "," + csvEscape(payee) + "," + csvEscape(memo) + ","
		if voided {
			line += "true\n"
		} else {
			line += "false\n"
		}
		buf = append(buf, line...)
	}
	return buf, rows.Err()
}

func csvEscape(s string) string {
	if s == "" {
		return ""
	}
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}
