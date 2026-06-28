package rulesengine

import (
	"testing"

	"github.com/google/uuid"
	"github.com/krishnamadhavan/expense-tracker/internal/domain"
)

func TestConfidenceFn_EmptyAndMerchant(t *testing.T) {
	r := ConfidenceFn(nil, nil)
	if r.Confidence != 0 || r.CategoryID != uuid.Nil {
		t.Fatalf("%+v", r)
	}
	cat := uuid.New()
	r2 := ConfidenceFn(nil, &cat)
	if r2.Confidence != 0.55 || r2.CategoryID != cat || !r2.FromMerchantNorm {
		t.Fatalf("%+v", r2)
	}
}

func TestConfidenceFn_LearnedNotAuto(t *testing.T) {
	rid := uuid.New()
	cat := uuid.New()
	m := []RuleMatch{{
		RuleID: rid, CategoryID: cat, Priority: 50, Confidence: 0.75, HitCount: 0, Origin: domain.RuleOriginLearned,
	}}
	r := ConfidenceFn(m, nil)
	if r.Confidence != 0.75 || r.Confidence >= ThresholdAuto {
		t.Fatalf("conf=%v", r.Confidence)
	}
	m[0].HitCount = 10 // +0.04 -> 0.79
	r2 := ConfidenceFn(m, nil)
	if r2.Confidence < 0.78 || r2.Confidence > 0.80 || r2.Confidence >= ThresholdAuto {
		t.Fatalf("conf=%v", r2.Confidence)
	}
}

func TestConfidenceFn_UserAuto(t *testing.T) {
	rid := uuid.New()
	cat := uuid.New()
	m := []RuleMatch{{
		RuleID: rid, CategoryID: cat, Priority: 10, Confidence: 0.95, HitCount: 0, Origin: domain.RuleOriginUser,
	}}
	r := ConfidenceFn(m, nil)
	if r.Confidence < ThresholdAuto || r.Conflict {
		t.Fatalf("%+v", r)
	}
}

func TestConfidenceFn_ConflictSamePriority(t *testing.T) {
	c1, c2 := uuid.New(), uuid.New()
	m := []RuleMatch{
		{RuleID: uuid.New(), CategoryID: c1, Priority: 50, Confidence: 0.9},
		{RuleID: uuid.New(), CategoryID: c2, Priority: 50, Confidence: 0.8},
	}
	r := ConfidenceFn(m, nil)
	if !r.Conflict || r.Confidence > 0.40 {
		t.Fatalf("%+v", r)
	}
}

func TestConfidenceFn_DifferentPriorityNoConflict(t *testing.T) {
	c1, c2 := uuid.New(), uuid.New()
	r1, r2 := uuid.New(), uuid.New()
	// lower priority number wins
	m := []RuleMatch{
		{RuleID: r1, CategoryID: c1, Priority: 10, Confidence: 0.9},
		{RuleID: r2, CategoryID: c2, Priority: 50, Confidence: 0.99},
	}
	r := ConfidenceFn(m, nil)
	if r.Conflict || r.CategoryID != c1 {
		t.Fatalf("%+v", r)
	}
}

func TestNormalizePayee_StripsPSP(t *testing.T) {
	got := NormalizePayee("  Foo@oksbi  ", nil)
	if got != "foo" {
		t.Fatalf("got %q", got)
	}
}
