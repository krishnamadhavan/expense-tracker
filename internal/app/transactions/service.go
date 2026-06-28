package transactions

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/krishnamadhavan/expense-tracker/internal/app"
	"github.com/krishnamadhavan/expense-tracker/internal/domain"
	"github.com/krishnamadhavan/expense-tracker/internal/ports"
)

// Service implements transaction use cases with household isolation.
type Service struct {
	Txns        ports.TransactionRepository
	Accounts    ports.AccountRepository
	Categories  ports.CategoryRepository
	Streams     ports.IncomeStreamRepository
	Categorizer ports.Categorizer
}

// CreateInput is the application-level create command.
type CreateInput struct {
	HouseholdID       domain.HouseholdID
	AccountID         domain.AccountID
	TransferAccountID *domain.AccountID
	CategoryID        *domain.CategoryID
	IncomeStreamID    *domain.IncomeStreamID
	Direction         domain.Direction
	Amount            domain.Money
	Currency          string
	TxnDate           time.Time
	PayeeRaw          string
	Memo              string
}

// Create validates FKs within household, optional categorizer no-op, then inserts.
func (s *Service) Create(ctx context.Context, in CreateInput) (domain.Transaction, error) {
	if err := app.RequireHousehold(in.HouseholdID); err != nil {
		return domain.Transaction{}, err
	}
	cat := s.Categorizer
	if cat == nil {
		cat = ports.NoopCategorizer{}
	}

	if _, err := s.Accounts.GetByID(ctx, in.HouseholdID, in.AccountID); err != nil {
		return domain.Transaction{}, fmt.Errorf("account_id: %w", mapCross(err))
	}
	if in.TransferAccountID != nil && *in.TransferAccountID != uuid.Nil {
		if _, err := s.Accounts.GetByID(ctx, in.HouseholdID, *in.TransferAccountID); err != nil {
			return domain.Transaction{}, fmt.Errorf("transfer_account_id: %w", mapCross(err))
		}
	}

	var catKind *domain.CategoryKind
	if in.CategoryID != nil && *in.CategoryID != uuid.Nil {
		c, err := s.Categories.GetByID(ctx, in.HouseholdID, *in.CategoryID)
		if err != nil {
			return domain.Transaction{}, fmt.Errorf("category_id: %w", mapCross(err))
		}
		k := c.Kind
		catKind = &k
	}
	if in.IncomeStreamID != nil && *in.IncomeStreamID != uuid.Nil {
		if _, err := s.Streams.GetByID(ctx, in.HouseholdID, *in.IncomeStreamID); err != nil {
			return domain.Transaction{}, fmt.Errorf("income_stream_id: %w", mapCross(err))
		}
	}

	draft := domain.TransactionDraft{
		HouseholdID:       in.HouseholdID,
		AccountID:         in.AccountID,
		TransferAccountID: in.TransferAccountID,
		CategoryID:        in.CategoryID,
		CategoryKind:      catKind,
		IncomeStreamID:    in.IncomeStreamID,
		Direction:         in.Direction,
		Amount:            in.Amount,
		Currency:          in.Currency,
		TxnDate:           in.TxnDate,
		PayeeRaw:          in.PayeeRaw,
		Memo:              in.Memo,
	}
	if err := domain.ValidateTransaction(draft); err != nil {
		return domain.Transaction{}, err
	}

	suggest, err := cat.Suggest(ctx, in.HouseholdID, draft)
	if err != nil {
		return domain.Transaction{}, err
	}
	userSet := in.CategoryID != nil && *in.CategoryID != uuid.Nil
	catID := in.CategoryID
	var conf *float64
	payeeNorm := suggest.PayeeNorm
	locked := false
	if userSet {
		locked = true
		one := 1.0
		conf = &one
	} else if suggest.CategoryID != nil && suggest.Confidence != nil && *suggest.Confidence >= 0.85 {
		catID = suggest.CategoryID
		conf = suggest.Confidence
	} else if suggest.CategoryID != nil {
		// still attach suggestion for audit but may be low confidence — attach only if auto threshold
		// keep category nil for low confidence so review queue drives moderation
		conf = suggest.Confidence
	}

	now := time.Now().UTC()
	txn := domain.Transaction{
		ID:                 domain.NewID(),
		HouseholdID:        in.HouseholdID,
		AccountID:          in.AccountID,
		TransferAccountID:  in.TransferAccountID,
		CategoryID:         catID,
		IncomeStreamID:     in.IncomeStreamID,
		Direction:          in.Direction,
		Amount:             in.Amount,
		Currency:           in.Currency,
		TxnDate:            in.TxnDate,
		PayeeRaw:           in.PayeeRaw,
		PayeeNorm:          payeeNorm,
		Memo:               in.Memo,
		CategoryLocked:     locked,
		CategoryConfidence: conf,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	// Expose userSet via payee for callers — return txn; attach events in HTTP layer
	return s.Txns.Create(ctx, txn)
}

// UserSetCategory reports whether create input included an explicit category.
func UserSetCategory(in CreateInput) bool {
	return in.CategoryID != nil && *in.CategoryID != uuid.Nil
}

func (s *Service) Get(ctx context.Context, householdID domain.HouseholdID, id domain.TransactionID) (domain.Transaction, error) {
	if err := app.RequireHousehold(householdID); err != nil {
		return domain.Transaction{}, err
	}
	return s.Txns.GetByID(ctx, householdID, id)
}

func (s *Service) List(ctx context.Context, householdID domain.HouseholdID, limit int) ([]domain.Transaction, error) {
	if err := app.RequireHousehold(householdID); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 50
	}
	return s.Txns.List(ctx, householdID, limit)
}

func (s *Service) Void(ctx context.Context, householdID domain.HouseholdID, id domain.TransactionID) (domain.Transaction, error) {
	if err := app.RequireHousehold(householdID); err != nil {
		return domain.Transaction{}, err
	}
	return s.Txns.Void(ctx, householdID, id)
}

func mapCross(err error) error {
	if errors.Is(err, domain.ErrNotFound) {
		return domain.ErrCrossHousehold
	}
	return err
}
