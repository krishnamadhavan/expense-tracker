package db

import (
	"testing"

	"github.com/google/uuid"
)

func TestDefaultSeedHouseholdIDParses(t *testing.T) {
	id, err := uuid.Parse(DefaultSeedHouseholdID)
	if err != nil {
		t.Fatal(err)
	}
	if id.String() != DefaultSeedHouseholdID {
		t.Fatalf("got %s", id)
	}
}
