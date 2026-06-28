package auth

import "testing"

func TestHashCheckPassword(t *testing.T) {
	h, err := HashPassword("s3cret")
	if err != nil {
		t.Fatal(err)
	}
	if !CheckPassword(h, "s3cret") || CheckPassword(h, "nope") {
		t.Fatal("password check")
	}
}
