package rulesengine

import (
	"math"
	"sort"

	"github.com/google/uuid"
	"github.com/krishnamadhavan/expense-tracker/internal/domain"
)

// RuleMatch is a rule that matched the draft.
type RuleMatch struct {
	RuleID     domain.CategoryRuleID
	CategoryID domain.CategoryID
	Priority   int
	Confidence float64
	HitCount   int
	Origin     domain.RuleOrigin
}

// ConfidenceResult is the output of ConfidenceFn.
type ConfidenceResult struct {
	CategoryID domain.CategoryID
	Confidence float64
	Conflict   bool
	RuleID     *domain.CategoryRuleID
	// FromMerchantNorm is true when only merchant default was used (no rule matches).
	FromMerchantNorm bool
}

// ThresholdAuto is the minimum confidence to auto-attach without review queue.
const ThresholdAuto = 0.85

// ConfidenceFn implements design pseudocode (tier = priority integer).
func ConfidenceFn(matches []RuleMatch, merchantDefault *domain.CategoryID) ConfidenceResult {
	if len(matches) == 0 {
		if merchantDefault != nil && *merchantDefault != uuid.Nil {
			return ConfidenceResult{
				CategoryID:       *merchantDefault,
				Confidence:       0.55,
				FromMerchantNorm: true,
			}
		}
		return ConfidenceResult{Confidence: 0}
	}

	minPri := matches[0].Priority
	for _, m := range matches[1:] {
		if m.Priority < minPri {
			minPri = m.Priority
		}
	}
	var tier []RuleMatch
	for _, m := range matches {
		if m.Priority == minPri {
			tier = append(tier, m)
		}
	}

	// best per category
	type best struct {
		m RuleMatch
	}
	byCat := map[domain.CategoryID]best{}
	for _, m := range tier {
		cur, ok := byCat[m.CategoryID]
		if !ok || m.Confidence > cur.m.Confidence || (m.Confidence == cur.m.Confidence && m.RuleID.String() < cur.m.RuleID.String()) {
			byCat[m.CategoryID] = best{m: m}
		}
	}

	if len(byCat) > 1 {
		// conflict: pick highest confidence then lowest rule id
		var winner RuleMatch
		first := true
		for _, b := range byCat {
			m := b.m
			if first || m.Confidence > winner.Confidence ||
				(m.Confidence == winner.Confidence && m.RuleID.String() < winner.RuleID.String()) {
				winner = m
				first = false
			}
		}
		conf := math.Min(winner.Confidence, 0.40)
		rid := winner.RuleID
		return ConfidenceResult{
			CategoryID: winner.CategoryID,
			Confidence: conf,
			Conflict:   true,
			RuleID:     &rid,
		}
	}

	var winner RuleMatch
	for _, b := range byCat {
		winner = b.m
	}
	conf := winner.Confidence
	boost := 0.02 * float64(winner.HitCount/5)
	if boost > 0.10 {
		boost = 0.10
	}
	conf = math.Min(0.99, conf+boost)
	rid := winner.RuleID
	return ConfidenceResult{
		CategoryID: winner.CategoryID,
		Confidence: conf,
		RuleID:     &rid,
	}
}

// SortMatches stabilizes test output (not required by ConfidenceFn).
func SortMatches(m []RuleMatch) {
	sort.Slice(m, func(i, j int) bool {
		if m[i].Priority != m[j].Priority {
			return m[i].Priority < m[j].Priority
		}
		return m[i].RuleID.String() < m[j].RuleID.String()
	})
}
