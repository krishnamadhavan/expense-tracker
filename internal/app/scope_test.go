package app

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/krishnamadhavan/expense-tracker/internal/domain"
)

func TestRequireHousehold(t *testing.T) {
	if err := RequireHousehold(uuid.Nil); !errors.Is(err, domain.ErrMissingHousehold) {
		t.Fatalf("got %v", err)
	}
	if err := RequireHousehold(uuid.New()); err != nil {
		t.Fatal(err)
	}
}
