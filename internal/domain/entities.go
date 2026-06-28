package domain

import (
	"time"

	"github.com/google/uuid"
)

// Household is the tenant scope for all financial data.
type Household struct {
	ID              HouseholdID
	Name            string
	DefaultCurrency string // ISO 4217, default INR
	FYStartMonth    time.Month // 1-12, default April
	Timezone        string     // IANA, default Asia/Kolkata
	CreatedAt       time.Time
}

// User is an authenticated principal belonging to one household.
type User struct {
	ID           UserID
	HouseholdID  HouseholdID
	Email        string
	PasswordHash string
	Role         UserRole
	CreatedAt    time.Time
}

// Account is a payment-channel tag (UPI, debit, credit, …). No balance fields in v1.
type Account struct {
	ID          AccountID
	HouseholdID HouseholdID
	Name        string
	Type        AccountType
	Currency    string
	IsActive    bool
}

// Category classifies transactions; Kind must align with transaction Direction when set.
type Category struct {
	ID          CategoryID
	HouseholdID HouseholdID
	ParentID    *CategoryID
	Name        string
	Kind        CategoryKind
	IsSystem    bool
}

// IncomeStream is a labeled income source (paycheck, business, rental, …).
type IncomeStream struct {
	ID          IncomeStreamID
	HouseholdID HouseholdID
	Name        string
	Code        string
}

// Tag is a free-form label within a household.
type Tag struct {
	ID          TagID
	HouseholdID HouseholdID
	Name        string
}

// Transaction is a single money movement (income, expense, or transfer).
type Transaction struct {
	ID                TransactionID
	HouseholdID       HouseholdID
	AccountID         AccountID
	TransferAccountID *AccountID // required when Direction == transfer (destination)
	CategoryID        *CategoryID
	IncomeStreamID    *IncomeStreamID
	Direction         Direction
	Amount            Money
	Currency          string
	TxnDate           time.Time // date in household timezone (time truncated to date in app layer)
	PayeeRaw          string
	PayeeNorm         string
	Memo              string
	CategoryLocked    bool
	CategoryConfidence *float64 // 0..1 suggestion confidence; nil if none
	VoidedAt          *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// IsVoided reports whether the transaction has been soft-voided.
func (t Transaction) IsVoided() bool {
	return t.VoidedAt != nil
}

// CategoryRule matches payee (or other field) patterns to a category.
type CategoryRule struct {
	ID          CategoryRuleID
	HouseholdID HouseholdID
	CategoryID  CategoryID
	MatchField  string // e.g. "payee_norm"
	MatchType   MatchType
	Pattern     string
	Priority    int
	Confidence  float64
	Origin      RuleOrigin
	HitCount    int
	IsActive    bool
}

// MerchantNorm is a normalized payee key with optional default category.
type MerchantNorm struct {
	ID                uuid.UUID
	HouseholdID       HouseholdID
	NormKey           string
	DisplayName       string
	DefaultCategoryID *CategoryID
}

// Budget is an optional spending/income limit for a month or FY window.
type Budget struct {
	ID           uuid.UUID
	HouseholdID  HouseholdID
	CategoryID   *CategoryID
	PeriodType   PeriodType
	PeriodStart  time.Time // first day of period (date)
	AmountLimit  Money
	Name         string
}
