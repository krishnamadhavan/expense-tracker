package domain

// Direction is the economic orientation of a transaction.
type Direction string

const (
	DirectionIncome   Direction = "income"
	DirectionExpense  Direction = "expense"
	DirectionTransfer Direction = "transfer"
)

// Valid reports whether d is a known direction.
func (d Direction) Valid() bool {
	switch d {
	case DirectionIncome, DirectionExpense, DirectionTransfer:
		return true
	default:
		return false
	}
}

// InPnL reports whether this direction contributes to income/expense P&L totals.
// Transfers are never included in P&L (design KD19 / transfer rules).
func (d Direction) InPnL() bool {
	return d == DirectionIncome || d == DirectionExpense
}

// AccountType is a payment-channel tag (not a balance-sheet account).
type AccountType string

const (
	AccountTypeUPI        AccountType = "upi"
	AccountTypeDebitCard  AccountType = "debit_card"
	AccountTypeCreditCard AccountType = "credit_card"
	AccountTypeCash       AccountType = "cash"
	AccountTypeBank       AccountType = "bank"
	AccountTypeOther      AccountType = "other"
)

// Valid reports whether t is a known account type.
func (t AccountType) Valid() bool {
	switch t {
	case AccountTypeUPI, AccountTypeDebitCard, AccountTypeCreditCard,
		AccountTypeCash, AccountTypeBank, AccountTypeOther:
		return true
	default:
		return false
	}
}

// CategoryKind classifies categories for validation against transaction direction.
type CategoryKind string

const (
	CategoryKindExpense  CategoryKind = "expense"
	CategoryKindIncome   CategoryKind = "income"
	CategoryKindTransfer CategoryKind = "transfer"
)

// Valid reports whether k is a known category kind.
func (k CategoryKind) Valid() bool {
	switch k {
	case CategoryKindExpense, CategoryKindIncome, CategoryKindTransfer:
		return true
	default:
		return false
	}
}

// MatchType is how a category rule matches payee/memo fields.
type MatchType string

const (
	MatchTypeExact    MatchType = "exact"
	MatchTypePrefix   MatchType = "prefix"
	MatchTypeContains MatchType = "contains"
)

// Valid reports whether m is allowed in v1 (no regex).
func (m MatchType) Valid() bool {
	switch m {
	case MatchTypeExact, MatchTypePrefix, MatchTypeContains:
		return true
	default:
		return false
	}
}

// RuleOrigin identifies who authored a category rule.
type RuleOrigin string

const (
	RuleOriginSystem  RuleOrigin = "system"
	RuleOriginUser    RuleOrigin = "user"
	RuleOriginLearned RuleOrigin = "learned"
)

// Valid reports whether o is known.
func (o RuleOrigin) Valid() bool {
	switch o {
	case RuleOriginSystem, RuleOriginUser, RuleOriginLearned:
		return true
	default:
		return false
	}
}

// UserRole is the principal role within a household (v1: admin only in practice).
type UserRole string

const (
	UserRoleAdmin UserRole = "admin"
)

// PeriodType is used by budgets and report windows.
type PeriodType string

const (
	PeriodTypeMonth PeriodType = "month"
	PeriodTypeFY    PeriodType = "fy"
)

// Valid reports whether p is known.
func (p PeriodType) Valid() bool {
	switch p {
	case PeriodTypeMonth, PeriodTypeFY:
		return true
	default:
		return false
	}
}

// TokenScope is an API token capability. "write" implies read in application logic.
type TokenScope string

const (
	TokenScopeRead  TokenScope = "read"
	TokenScopeWrite TokenScope = "write"
)

// Valid reports whether s is known.
func (s TokenScope) Valid() bool {
	switch s {
	case TokenScopeRead, TokenScopeWrite:
		return true
	default:
		return false
	}
}

// IncludesRead is true for read or write scopes.
func (s TokenScope) IncludesRead() bool {
	return s == TokenScopeRead || s == TokenScopeWrite
}

// IncludesWrite is true only for write scope.
func (s TokenScope) IncludesWrite() bool {
	return s == TokenScopeWrite
}
