package ports

import (
	"context"

	"github.com/krishnamadhavan/expense-tracker/internal/domain"
)

// TransactionRepository persists transactions; all methods take householdID for tenancy.
type TransactionRepository interface {
	Create(ctx context.Context, txn domain.Transaction) (domain.Transaction, error)
	GetByID(ctx context.Context, householdID domain.HouseholdID, id domain.TransactionID) (domain.Transaction, error)
	List(ctx context.Context, householdID domain.HouseholdID, limit int) ([]domain.Transaction, error)
	// Void sets voided_at; does not hard-delete.
	Void(ctx context.Context, householdID domain.HouseholdID, id domain.TransactionID) (domain.Transaction, error)
}
