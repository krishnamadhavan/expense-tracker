package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/krishnamadhavan/expense-tracker/internal/ports"
)

func TestRequireCSRF_Session(t *testing.T) {
	h := RequireCSRF(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	p := ports.Principal{CSRFSecret: "secret", ViaBearer: false}
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req = req.WithContext(context.WithValue(req.Context(), principalKey, p))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("want 403 got %d", rr.Code)
	}
	req2 := httptest.NewRequest(http.MethodPost, "/", nil)
	req2.Header.Set(CSRFHeader, "secret")
	req2 = req2.WithContext(context.WithValue(req2.Context(), principalKey, p))
	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusNoContent {
		t.Fatalf("want 204 got %d", rr2.Code)
	}
}

func TestRequireCSRF_BearerExempt(t *testing.T) {
	h := RequireCSRF(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	p := ports.Principal{ViaBearer: true}
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req = req.WithContext(context.WithValue(req.Context(), principalKey, p))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("got %d", rr.Code)
	}
}
