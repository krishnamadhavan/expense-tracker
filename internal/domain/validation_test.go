package domain

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestValidateTransaction_TransferRequiresDistinctCounterparty(t *testing.T) {
	hh := uuid.New()
	from := uuid.New()
	to := uuid.New()
	amt := MustParseMoney("100.00")
	day := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	t.Run("valid transfer", func(t *testing.T) {
		err := ValidateTransaction(TransactionDraft{
			HouseholdID:       hh,
			AccountID:         from,
			TransferAccountID: &to,
			Direction:         DirectionTransfer,
			Amount:            amt,
			Currency:          "INR",
			TxnDate:           day,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("missing transfer_account_id", func(t *testing.T) {
		err := ValidateTransaction(TransactionDraft{
			HouseholdID: hh,
			AccountID:   from,
			Direction:   DirectionTransfer,
			Amount:      amt,
			Currency:    "INR",
			TxnDate:     day,
		})
		if !errors.Is(err, ErrInvalidTransfer) {
			t.Fatalf("got %v, want ErrInvalidTransfer", err)
		}
	})

	t.Run("same source and destination", func(t *testing.T) {
		same := from
		err := ValidateTransaction(TransactionDraft{
			HouseholdID:       hh,
			AccountID:         from,
			TransferAccountID: &same,
			Direction:         DirectionTransfer,
			Amount:            amt,
			Currency:          "INR",
			TxnDate:           day,
		})
		if !errors.Is(err, ErrInvalidTransfer) {
			t.Fatalf("got %v, want ErrInvalidTransfer", err)
		}
	})

	t.Run("transfer_account_id on expense rejected", func(t *testing.T) {
		err := ValidateTransaction(TransactionDraft{
			HouseholdID:       hh,
			AccountID:         from,
			TransferAccountID: &to,
			Direction:         DirectionExpense,
			Amount:            amt,
			Currency:          "INR",
			TxnDate:           day,
		})
		if !errors.Is(err, ErrInvalidTransfer) {
			t.Fatalf("got %v, want ErrInvalidTransfer", err)
		}
	})

	t.Run("income_stream on transfer rejected", func(t *testing.T) {
		stream := uuid.New()
		err := ValidateTransaction(TransactionDraft{
			HouseholdID:       hh,
			AccountID:         from,
			TransferAccountID: &to,
			IncomeStreamID:    &stream,
			Direction:         DirectionTransfer,
			Amount:            amt,
			Currency:          "INR",
			TxnDate:           day,
		})
		if !errors.Is(err, ErrInvalidArgument) {
			t.Fatalf("got %v, want ErrInvalidArgument", err)
		}
	})
}

func TestValidateCategoryForDirection(t *testing.T) {
	cat := uuid.New()
	expense := CategoryKindExpense
	income := CategoryKindIncome
	transfer := CategoryKindTransfer

	cases := []struct {
		name    string
		dir     Direction
		catID   *CategoryID
		kind    *CategoryKind
		wantErr error
	}{
		{"uncategorized expense ok", DirectionExpense, nil, nil, nil},
		{"uncategorized transfer ok", DirectionTransfer, nil, nil, nil},
		{"expense with expense kind", DirectionExpense, &cat, &expense, nil},
		{"expense with income kind", DirectionExpense, &cat, &income, ErrCategoryKindMismatch},
		{"income with income kind", DirectionIncome, &cat, &income, nil},
		{"income with expense kind", DirectionIncome, &cat, &expense, ErrCategoryKindMismatch},
		{"transfer with transfer kind", DirectionTransfer, &cat, &transfer, nil},
		{"transfer with expense kind", DirectionTransfer, &cat, &expense, ErrCategoryKindMismatch},
		{"category without kind", DirectionExpense, &cat, nil, ErrInvalidArgument},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCategoryForDirection(tc.dir, tc.catID, tc.kind)
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("unexpected err: %v", err)
				}
				return
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestDirection_InPnL(t *testing.T) {
	if !DirectionIncome.InPnL() || !DirectionExpense.InPnL() {
		t.Fatal("income and expense must be in P&L")
	}
	if DirectionTransfer.InPnL() {
		t.Fatal("transfer must not be in P&L")
	}
}

func TestParseMoney(t *testing.T) {
	m, err := ParseMoney("10.50")
	if err != nil || m.Minor != 1050 || m.String() != "10.50" {
		t.Fatalf("got minor=%d str=%q err=%v", m.Minor, m.String(), err)
	}
	if _, err := ParseMoney("-1"); !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("negative: %v", err)
	}
	if _, err := ParseMoney("1.001"); !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("3 dp: %v", err)
	}
}
