package app

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/krishnamadhavan/expense-tracker/internal/domain"
)

// RequireHousehold ensures householdID is non-nil (principal tenancy gate for all commands/queries).
func RequireHousehold(householdID domain.HouseholdID) error {
	if householdID == uuid.Nil {
		return fmt.Errorf("%w", domain.ErrMissingHousehold)
	}
	return nil
}
