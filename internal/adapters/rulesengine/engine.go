package rulesengine

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/krishnamadhavan/expense-tracker/internal/domain"
	"github.com/krishnamadhavan/expense-tracker/internal/ports"
)

// RuleStore loads active rules and merchant norms for a household.
type RuleStore interface {
	ListActiveRules(ctx context.Context, householdID domain.HouseholdID) ([]domain.CategoryRule, error)
	GetMerchantNorm(ctx context.Context, householdID domain.HouseholdID, normKey string) (defaultCategoryID *domain.CategoryID, ok bool, err error)
}

// Engine implements ports.Categorizer using rules + ConfidenceFn.
type Engine struct {
	Store       RuleStore
	PSPSuffixes []string
}

// Suggest implements ports.Categorizer.
func (e *Engine) Suggest(ctx context.Context, householdID domain.HouseholdID, draft domain.TransactionDraft) (ports.SuggestResult, error) {
	norm := NormalizePayee(draft.PayeeRaw, e.PSPSuffixes)
	rules, err := e.Store.ListActiveRules(ctx, householdID)
	if err != nil {
		return ports.SuggestResult{PayeeNorm: norm}, err
	}
	var matches []RuleMatch
	for _, rule := range rules {
		if matchRule(rule, norm, draft) {
			matches = append(matches, RuleMatch{
				RuleID:     rule.ID,
				CategoryID: rule.CategoryID,
				Priority:   rule.Priority,
				Confidence: rule.Confidence,
				HitCount:   rule.HitCount,
				Origin:     rule.Origin,
			})
		}
	}
	var merchantDef *domain.CategoryID
	if norm != "" {
		if def, ok, err := e.Store.GetMerchantNorm(ctx, householdID, norm); err != nil {
			return ports.SuggestResult{PayeeNorm: norm}, err
		} else if ok {
			merchantDef = def
		}
	}
	res := ConfidenceFn(matches, merchantDef)
	out := ports.SuggestResult{PayeeNorm: norm, RuleID: res.RuleID}
	if res.CategoryID != uuid.Nil {
		cid := res.CategoryID
		out.CategoryID = &cid
		c := res.Confidence
		out.Confidence = &c
	}
	// Stash conflict via confidence path handled in app layer using re-run — expose via negative confidence sentinel not ideal.
	// App uses Engine.SuggestDetailed for queue reasons.
	_ = res.Conflict
	return out, nil
}

// SuggestDetailed returns full confidence result for create path review queue.
func (e *Engine) SuggestDetailed(ctx context.Context, householdID domain.HouseholdID, draft domain.TransactionDraft) (ports.SuggestResult, ConfidenceResult, error) {
	norm := NormalizePayee(draft.PayeeRaw, e.PSPSuffixes)
	rules, err := e.Store.ListActiveRules(ctx, householdID)
	if err != nil {
		return ports.SuggestResult{PayeeNorm: norm}, ConfidenceResult{}, err
	}
	var matches []RuleMatch
	hadRuleMatch := false
	for _, rule := range rules {
		if matchRule(rule, norm, draft) {
			hadRuleMatch = true
			matches = append(matches, RuleMatch{
				RuleID:     rule.ID,
				CategoryID: rule.CategoryID,
				Priority:   rule.Priority,
				Confidence: rule.Confidence,
				HitCount:   rule.HitCount,
				Origin:     rule.Origin,
			})
		}
	}
	var merchantDef *domain.CategoryID
	merchantOK := false
	if norm != "" {
		if def, ok, err := e.Store.GetMerchantNorm(ctx, householdID, norm); err != nil {
			return ports.SuggestResult{PayeeNorm: norm}, ConfidenceResult{}, err
		} else if ok {
			merchantDef = def
			merchantOK = true
		}
	}
	res := ConfidenceFn(matches, merchantDef)
	out := ports.SuggestResult{PayeeNorm: norm, RuleID: res.RuleID}
	if res.CategoryID != uuid.Nil {
		cid := res.CategoryID
		out.CategoryID = &cid
		c := res.Confidence
		out.Confidence = &c
	}
	// annotate new merchant for callers via Conflict false + FromMerchantNorm + !hadRuleMatch
	if !hadRuleMatch && !merchantOK && norm != "" {
		res.Conflict = false // reason new_merchant handled in app
	}
	_ = merchantOK
	return out, res, nil
}

func matchRule(rule domain.CategoryRule, payeeNorm string, draft domain.TransactionDraft) bool {
	var field string
	switch rule.MatchField {
	case "payee_norm":
		field = payeeNorm
	case "payee_raw":
		field = strings.ToLower(strings.TrimSpace(draft.PayeeRaw))
	case "memo":
		field = strings.ToLower(strings.TrimSpace(draft.Memo))
	case "account_id":
		field = draft.AccountID.String()
	default:
		return false
	}
	pat := rule.Pattern
	switch rule.MatchType {
	case domain.MatchTypeExact:
		return field == pat
	case domain.MatchTypePrefix:
		return strings.HasPrefix(field, pat)
	case domain.MatchTypeContains:
		return strings.Contains(field, pat)
	default:
		return false
	}
}

var _ ports.Categorizer = (*Engine)(nil)
