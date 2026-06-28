package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// TransactionDraft is the validated input for creating or updating a transaction
// before persistence (PR04 will map this to the DB adapter).
type TransactionDraft struct {
	HouseholdID       HouseholdID
	AccountID         AccountID
	TransferAccountID *AccountID
	CategoryID        *CategoryID
	CategoryKind      *CategoryKind // required when CategoryID is set (caller supplies from catalog)
	IncomeStreamID    *IncomeStreamID
	Direction         Direction
	Amount            Money
	Currency          string
	TxnDate           time.Time
	PayeeRaw          string
	Memo              string
}

// ValidateTransaction enforces transfer invariants, amount, direction, and
// category kind vs direction alignment (design § category kind vs direction).
func ValidateTransaction(d TransactionDraft) error {
	if d.HouseholdID == uuid.Nil {
		return fmt.Errorf("%w: household_id", ErrMissingHousehold)
	}
	if d.AccountID == uuid.Nil {
		return fmt.Errorf("%w: account_id", ErrMissingAccount)
	}
	if !d.Direction.Valid() {
		return fmt.Errorf("%w: %q", ErrInvalidDirection, d.Direction)
	}
	if d.Amount.Minor < 0 {
		return fmt.Errorf("%w: must be >= 0", ErrInvalidAmount)
	}
	cur := strings.TrimSpace(d.Currency)
	if cur == "" {
		return fmt.Errorf("%w: currency is required", ErrInvalidArgument)
	}
	if len(cur) != 3 {
		return fmt.Errorf("%w: currency must be ISO 4217 (3 letters)", ErrInvalidArgument)
	}
	if d.TxnDate.IsZero() {
		return fmt.Errorf("%w: txn_date is required", ErrInvalidArgument)
	}

	switch d.Direction {
	case DirectionTransfer:
		if d.TransferAccountID == nil || *d.TransferAccountID == uuid.Nil {
			return fmt.Errorf("%w: transfer_account_id is required for transfers", ErrInvalidTransfer)
		}
		if *d.TransferAccountID == d.AccountID {
			return fmt.Errorf("%w: transfer source and destination must differ", ErrInvalidTransfer)
		}
		// Income stream does not apply to transfers.
		if d.IncomeStreamID != nil && *d.IncomeStreamID != uuid.Nil {
			return fmt.Errorf("%w: income_stream_id not allowed on transfers", ErrInvalidArgument)
		}
	default:
		if d.TransferAccountID != nil && *d.TransferAccountID != uuid.Nil {
			return fmt.Errorf("%w: transfer_account_id only valid when direction=transfer", ErrInvalidTransfer)
		}
	}

	if err := ValidateCategoryForDirection(d.Direction, d.CategoryID, d.CategoryKind); err != nil {
		return err
	}
	return nil
}

// ValidateCategoryForDirection implements the normative matrix:
//
//	expense  -> category kind expense only (if set)
//	income   -> category kind income only (if set)
//	transfer -> null category OK; if set, kind must be transfer
func ValidateCategoryForDirection(dir Direction, categoryID *CategoryID, kind *CategoryKind) error {
	if categoryID == nil || *categoryID == uuid.Nil {
		// Uncategorized is always allowed at domain level (review queue may still apply).
		return nil
	}
	if kind == nil {
		return fmt.Errorf("%w: category kind required when category_id is set", ErrInvalidArgument)
	}
	if !kind.Valid() {
		return fmt.Errorf("%w: unknown category kind %q", ErrInvalidArgument, *kind)
	}
	switch dir {
	case DirectionExpense:
		if *kind != CategoryKindExpense {
			return fmt.Errorf("%w: expense requires expense category", ErrCategoryKindMismatch)
		}
	case DirectionIncome:
		if *kind != CategoryKindIncome {
			return fmt.Errorf("%w: income requires income category", ErrCategoryKindMismatch)
		}
	case DirectionTransfer:
		if *kind != CategoryKindTransfer {
			return fmt.Errorf("%w: transfer requires transfer category (or null)", ErrCategoryKindMismatch)
		}
	default:
		return fmt.Errorf("%w: %q", ErrInvalidDirection, dir)
	}
	return nil
}

// ValidateAccountType ensures the payment channel type is known.
func ValidateAccountType(t AccountType) error {
	if !t.Valid() {
		return fmt.Errorf("%w: account type %q", ErrInvalidArgument, t)
	}
	return nil
}

// ValidateMatchType ensures v1 rule match types only (no regex).
func ValidateMatchType(m MatchType) error {
	if !m.Valid() {
		return fmt.Errorf("%w: match type %q (regex not supported in v1)", ErrInvalidArgument, m)
	}
	return nil
}

// ValidateFYStartMonth ensures month is 1..12.
func ValidateFYStartMonth(m time.Month) error {
	if m < 1 || m > 12 {
		return fmt.Errorf("%w: fy_start_month must be 1-12", ErrInvalidArgument)
	}
	return nil
}
