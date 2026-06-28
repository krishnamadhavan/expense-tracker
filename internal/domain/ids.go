package domain

import "github.com/google/uuid"

// ID type aliases keep call sites readable; all are UUIDs at rest.
type (
	HouseholdID    = uuid.UUID
	UserID         = uuid.UUID
	AccountID      = uuid.UUID
	CategoryID     = uuid.UUID
	IncomeStreamID = uuid.UUID
	TransactionID  = uuid.UUID
	TagID          = uuid.UUID
	CategoryRuleID = uuid.UUID
	SessionID      = uuid.UUID
	APITokenID     = uuid.UUID
)

// NewID returns a random UUID suitable for new entities.
func NewID() uuid.UUID {
	return uuid.New()
}

// ParseID parses a UUID string or returns an error.
func ParseID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}
