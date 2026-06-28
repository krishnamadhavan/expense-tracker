package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/krishnamadhavan/expense-tracker/internal/app/auth"
	"github.com/krishnamadhavan/expense-tracker/internal/ports"
)

type ctxKey int

const principalKey ctxKey = 1

// SessionCookieName is the HttpOnly session cookie.
const SessionCookieName = "et_session"

// CSRFCookieName delivers the CSRF secret (not HttpOnly so SPA can read it).
const CSRFCookieName = "csrf_token"

// CSRFHeader is compared to session.csrf_secret (session is authority).
const CSRFHeader = "X-CSRF-Token"

// Authenticator resolves session or bearer tokens.
type Authenticator struct {
	Auth         *auth.Service
	AuthDisabled bool
	// DevPrincipal used only when AuthDisabled (tests).
	DevPrincipal ports.Principal
}

// PrincipalFrom returns the authenticated principal or false.
func PrincipalFrom(ctx context.Context) (ports.Principal, bool) {
	p, ok := ctx.Value(principalKey).(ports.Principal)
	return p, ok
}

// Authenticate populates principal from Bearer or session cookie.
func (a *Authenticator) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.AuthDisabled {
			ctx := context.WithValue(r.Context(), principalKey, a.DevPrincipal)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		if h := r.Header.Get("Authorization"); strings.HasPrefix(strings.ToLower(h), "bearer ") {
			raw := strings.TrimSpace(h[7:])
			p, err := a.Auth.ResolveBearer(r.Context(), raw)
			if err == nil {
				ctx := context.WithValue(r.Context(), principalKey, p)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}
		if c, err := r.Cookie(SessionCookieName); err == nil && c.Value != "" {
			p, err := a.Auth.ResolveSession(r.Context(), c.Value)
			if err == nil {
				ctx := context.WithValue(r.Context(), principalKey, p)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// RequireAuth rejects unauthenticated requests.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := PrincipalFrom(r.Context()); !ok {
			WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireWrite ensures bearer tokens have write scope (sessions always write).
func RequireWrite(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, ok := PrincipalFrom(r.Context())
		if !ok {
			WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
			return
		}
		if p.ViaBearer && !p.Scope.IncludesWrite() {
			WriteError(w, http.StatusForbidden, "forbidden", "write scope required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireCSRF enforces X-CSRF-Token == session csrf_secret for cookie sessions on mutating methods.
func RequireCSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}
		p, ok := PrincipalFrom(r.Context())
		if !ok {
			next.ServeHTTP(w, r)
			return
		}
		if p.ViaBearer {
			next.ServeHTTP(w, r)
			return
		}
		got := r.Header.Get(CSRFHeader)
		if got == "" || got != p.CSRFSecret {
			WriteError(w, http.StatusForbidden, "csrf_failed", "invalid or missing CSRF token")
			return
		}
		next.ServeHTTP(w, r)
	})
}
