package catalog

import (
	"context"
	"fmt"

	"github.com/krishnamadhavan/expense-tracker/internal/app"
	"github.com/krishnamadhavan/expense-tracker/internal/domain"
	"github.com/krishnamadhavan/expense-tracker/internal/ports"
)

// Service exposes household-scoped catalog reads.
type Service struct {
	Accounts      ports.AccountRepository
	Categories    ports.CategoryRepository
	IncomeStreams ports.IncomeStreamRepository
}

func (s *Service) ListAccounts(ctx context.Context, householdID domain.HouseholdID) ([]domain.Account, error) {
	if err := app.RequireHousehold(householdID); err != nil {
		return nil, err
	}
	return s.Accounts.List(ctx, householdID)
}

func (s *Service) GetAccount(ctx context.Context, householdID domain.HouseholdID, id domain.AccountID) (domain.Account, error) {
	if err := app.RequireHousehold(householdID); err != nil {
		return domain.Account{}, err
	}
	return s.Accounts.GetByID(ctx, householdID, id)
}

func (s *Service) ListCategories(ctx context.Context, householdID domain.HouseholdID) ([]domain.Category, error) {
	if err := app.RequireHousehold(householdID); err != nil {
		return nil, err
	}
	return s.Categories.List(ctx, householdID)
}

func (s *Service) GetCategory(ctx context.Context, householdID domain.HouseholdID, id domain.CategoryID) (domain.Category, error) {
	if err := app.RequireHousehold(householdID); err != nil {
		return domain.Category{}, err
	}
	return s.Categories.GetByID(ctx, householdID, id)
}

func (s *Service) ListIncomeStreams(ctx context.Context, householdID domain.HouseholdID) ([]domain.IncomeStream, error) {
	if err := app.RequireHousehold(householdID); err != nil {
		return nil, err
	}
	return s.IncomeStreams.List(ctx, householdID)
}

// EnsureAccountInHousehold returns ErrCrossHousehold-style not found if missing (repo scopes by household).
func (s *Service) EnsureAccountInHousehold(ctx context.Context, householdID domain.HouseholdID, id domain.AccountID) (domain.Account, error) {
	acc, err := s.GetAccount(ctx, householdID, id)
	if err != nil {
		return domain.Account{}, err
	}
	if acc.HouseholdID != householdID {
		return domain.Account{}, fmt.Errorf("%w: account", domain.ErrCrossHousehold)
	}
	return acc, nil
}
