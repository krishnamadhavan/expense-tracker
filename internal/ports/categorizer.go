package ports

import (
	"context"

	"github.com/krishnamadhavan/expense-tracker/internal/domain"
)

// SuggestResult is a categorization suggestion (PR06 implements real rules).
type SuggestResult struct {
	CategoryID *domain.CategoryID
	Confidence *float64
	PayeeNorm  string
	RuleID     *domain.CategoryRuleID
}

// Categorizer suggests a category for a draft transaction. PR04 uses a no-op implementation.
type Categorizer interface {
	Suggest(ctx context.Context, householdID domain.HouseholdID, draft domain.TransactionDraft) (SuggestResult, error)
}

// NoopCategorizer never suggests a category.
type NoopCategorizer struct{}

func (NoopCategorizer) Suggest(context.Context, domain.HouseholdID, domain.TransactionDraft) (SuggestResult, error) {
	return SuggestResult{}, nil
}
