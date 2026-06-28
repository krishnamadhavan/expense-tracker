//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/krishnamadhavan/expense-tracker/internal/adapters/postgres"
	"github.com/krishnamadhavan/expense-tracker/internal/app/transactions"
	"github.com/krishnamadhavan/expense-tracker/internal/db"
	"github.com/krishnamadhavan/expense-tracker/internal/domain"
	"github.com/krishnamadhavan/expense-tracker/internal/ports"
)

func testPool(t *testing.T) *postgres.AccountRepo {
	t.Helper()
	url := os.Getenv("ET_DATABASE_URL")
	if url == "" {
		t.Skip("ET_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, url)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return &postgres.AccountRepo{Pool: pool}
}

func TestIntegration_CrossHouseholdRejected(t *testing.T) {
	url := os.Getenv("ET_DATABASE_URL")
	if url == "" {
		t.Skip("ET_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	seedHH, err := uuid.Parse(db.DefaultSeedHouseholdID)
	if err != nil {
		t.Fatal(err)
	}

	// Second household with its own account
	var otherHH uuid.UUID
	err = pool.QueryRow(ctx, `
		INSERT INTO households (name) VALUES ('Other HH') RETURNING id`).Scan(&otherHH)
	if err != nil {
		t.Fatal(err)
	}
	accRepo := &postgres.AccountRepo{Pool: pool}
	otherAcc, err := accRepo.InsertTestAccount(ctx, otherHH, "Other UPI", domain.AccountTypeUPI)
	if err != nil {
		t.Fatal(err)
	}

	// Seed household account
	seedAccs, err := accRepo.List(ctx, seedHH)
	if err != nil || len(seedAccs) == 0 {
		t.Fatalf("seed accounts: %v len=%d", err, len(seedAccs))
	}
	seedAcc := seedAccs[0]

	catRepo := &postgres.CategoryRepo{Pool: pool}
	streamRepo := &postgres.IncomeStreamRepo{Pool: pool}
	txnRepo := &postgres.TransactionRepo{Pool: pool}
	svc := &transactions.Service{
		Txns:        txnRepo,
		Accounts:    accRepo,
		Categories:  catRepo,
		Streams:     streamRepo,
		Categorizer: ports.NoopCategorizer{},
	}

	// Cross-household account_id
	_, err = svc.Create(ctx, transactions.CreateInput{
		HouseholdID: seedHH,
		AccountID:   otherAcc.ID,
		Direction:   domain.DirectionExpense,
		Amount:      domain.MustParseMoney("10.00"),
		Currency:    "INR",
		TxnDate:     time.Now(),
	})
	if !errors.Is(err, domain.ErrCrossHousehold) {
		t.Fatalf("account cross: got %v", err)
	}

	// Cross-household transfer_account_id
	_, err = svc.Create(ctx, transactions.CreateInput{
		HouseholdID:       seedHH,
		AccountID:         seedAcc.ID,
		TransferAccountID: &otherAcc.ID,
		Direction:         domain.DirectionTransfer,
		Amount:            domain.MustParseMoney("10.00"),
		Currency:          "INR",
		TxnDate:           time.Now(),
	})
	if !errors.Is(err, domain.ErrCrossHousehold) {
		t.Fatalf("transfer cross: got %v", err)
	}

	// Happy path expense on seed account
	txn, err := svc.Create(ctx, transactions.CreateInput{
		HouseholdID: seedHH,
		AccountID:   seedAcc.ID,
		Direction:   domain.DirectionExpense,
		Amount:      domain.MustParseMoney("42.50"),
		Currency:    "INR",
		TxnDate:     time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		PayeeRaw:    "Test Merchant",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := svc.Get(ctx, seedHH, txn.ID)
	if err != nil || got.Amount.Minor != 4250 {
		t.Fatalf("get: %+v err=%v", got, err)
	}
	voided, err := svc.Void(ctx, seedHH, txn.ID)
	if err != nil || !voided.IsVoided() {
		t.Fatalf("void: %+v err=%v", voided, err)
	}
}
