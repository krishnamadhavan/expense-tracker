package domain

import "testing"

func TestAccountTypeValid(t *testing.T) {
	if !AccountTypeUPI.Valid() || AccountType("wallet").Valid() {
		t.Fatal("account type validity")
	}
}

func TestMatchTypeNoRegex(t *testing.T) {
	if MatchType("regex").Valid() {
		t.Fatal("regex must not be valid in v1")
	}
	if err := ValidateMatchType(MatchTypeExact); err != nil {
		t.Fatal(err)
	}
	if err := ValidateMatchType(MatchType("regex")); err == nil {
		t.Fatal("expected error")
	}
}

func TestTokenScope(t *testing.T) {
	if !TokenScopeWrite.IncludesRead() || !TokenScopeWrite.IncludesWrite() {
		t.Fatal("write implies read")
	}
	if TokenScopeRead.IncludesWrite() {
		t.Fatal("read must not include write")
	}
}
