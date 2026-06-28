package ports

import (
	"context"

	"github.com/krishnamadhavan/expense-tracker/internal/domain"
)

// AccountRepository loads payment-channel accounts scoped by household.
type AccountRepository interface {
	GetByID(ctx context.Context, householdID domain.HouseholdID, id domain.AccountID) (domain.Account, error)
	List(ctx context.Context, householdID domain.HouseholdID) ([]domain.Account, error)
}

// CategoryRepository loads categories scoped by household.
type CategoryRepository interface {
	GetByID(ctx context.Context, householdID domain.HouseholdID, id domain.CategoryID) (domain.Category, error)
	List(ctx context.Context, householdID domain.HouseholdID) ([]domain.Category, error)
}

// IncomeStreamRepository loads income streams scoped by household.
type IncomeStreamRepository interface {
	GetByID(ctx context.Context, householdID domain.HouseholdID, id domain.IncomeStreamID) (domain.IncomeStream, error)
	List(ctx context.Context, householdID domain.HouseholdID) ([]domain.IncomeStream, error)
}
