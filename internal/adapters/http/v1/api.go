package v1

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/krishnamadhavan/expense-tracker/internal/adapters/http/middleware"
	"github.com/krishnamadhavan/expense-tracker/internal/adapters/postgres"
	"github.com/krishnamadhavan/expense-tracker/internal/app/auth"
	"github.com/krishnamadhavan/expense-tracker/internal/app/catalog"
	"github.com/krishnamadhavan/expense-tracker/internal/app/transactions"
	"github.com/krishnamadhavan/expense-tracker/internal/config"
	"github.com/krishnamadhavan/expense-tracker/internal/domain"
)

// API wires v1 routes.
type API struct {
	Cfg      config.Config
	Auth     *auth.Service
	Catalog  *catalog.Service
	Txns     *transactions.Service
	Idem     *postgres.IdempotencyRepo
	AuthMW   *middleware.Authenticator
	LoginRL  *middleware.LoginRateLimiter
}

// Routes mounts /api/v1.
func (a *API) Routes(r chi.Router) {
	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/login", a.login)
		r.With(middleware.RequireAuth).Post("/auth/logout", a.logout)
		r.With(middleware.RequireAuth).Get("/auth/me", a.me)
		r.With(middleware.RequireAuth).Get("/auth/csrf", a.csrf)
		r.With(middleware.RequireAuth, middleware.RequireCSRF, middleware.RequireWrite).Post("/auth/tokens", a.createToken)

		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth)
			r.Get("/accounts", a.listAccounts)
			r.Get("/categories", a.listCategories)
			r.Get("/income-streams", a.listStreams)

			r.With(middleware.RequireCSRF, middleware.RequireWrite).Post("/transactions", a.createTxn)
			r.Get("/transactions", a.listTxns)
			r.Get("/transactions/{id}", a.getTxn)
			r.With(middleware.RequireCSRF, middleware.RequireWrite).Post("/transactions/{id}/void", a.voidTxn)
		})
	})
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (a *API) login(w http.ResponseWriter, r *http.Request) {
	ip := middleware.ClientIP(r)
	if a.LoginRL != nil && !a.LoginRL.Allow(ip) {
		middleware.WriteError(w, http.StatusTooManyRequests, "rate_limited", "too many login attempts")
		return
	}
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "bad_request", "invalid JSON")
		return
	}
	raw, csrf, exp, err := a.Auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if auth.IsInvalidCredentials(err) || errors.Is(err, domain.ErrNotFound) {
			middleware.WriteError(w, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
			return
		}
		middleware.WriteError(w, http.StatusInternalServerError, "internal", "login failed")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    raw,
		Path:     "/",
		HttpOnly: true,
		Secure:   a.Cfg.CookieSecure,
		SameSite: sameSite(a.Cfg.CookieSameSite),
		Expires:  exp,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     middleware.CSRFCookieName,
		Value:    csrf,
		Path:     "/",
		HttpOnly: false,
		Secure:   a.Cfg.CookieSecure,
		SameSite: sameSite(a.Cfg.CookieSameSite),
		Expires:  exp,
	})
	middleware.WriteJSON(w, http.StatusOK, map[string]any{
		"expires_at": exp.UTC().Format(time.RFC3339),
		"csrf_token": csrf,
	})
}

func (a *API) logout(w http.ResponseWriter, r *http.Request) {
	p, _ := middleware.PrincipalFrom(r.Context())
	if p.SessionID != uuid.Nil {
		_ = a.Auth.Logout(r.Context(), p.SessionID)
	}
	clearCookie(w, middleware.SessionCookieName, a.Cfg)
	clearCookie(w, middleware.CSRFCookieName, a.Cfg)
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) me(w http.ResponseWriter, r *http.Request) {
	p, _ := middleware.PrincipalFrom(r.Context())
	middleware.WriteJSON(w, http.StatusOK, map[string]any{
		"user_id":      p.UserID,
		"household_id": p.HouseholdID,
		"email":        p.Email,
		"via_bearer":   p.ViaBearer,
	})
}

func (a *API) csrf(w http.ResponseWriter, r *http.Request) {
	p, _ := middleware.PrincipalFrom(r.Context())
	if p.ViaBearer || p.CSRFSecret == "" {
		middleware.WriteError(w, http.StatusBadRequest, "bad_request", "CSRF only for session auth")
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]string{"csrf_token": p.CSRFSecret})
}

type tokenReq struct {
	Name   string `json:"name"`
	Scopes string `json:"scopes"`
}

func (a *API) createToken(w http.ResponseWriter, r *http.Request) {
	p, _ := middleware.PrincipalFrom(r.Context())
	var req tokenReq
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Name == "" {
		req.Name = "default"
	}
	scope := domain.TokenScopeWrite
	if req.Scopes == "read" {
		scope = domain.TokenScopeRead
	}
	raw, rec, err := a.Auth.CreateAPIToken(r.Context(), p.UserID, req.Name, scope, 0)
	if err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	middleware.WriteJSON(w, http.StatusCreated, map[string]any{
		"id":         rec.ID,
		"name":       rec.Name,
		"prefix":     rec.Prefix,
		"scopes":     rec.Scopes,
		"expires_at": rec.ExpiresAt.UTC().Format(time.RFC3339),
		"token":      raw,
	})
}

func (a *API) listAccounts(w http.ResponseWriter, r *http.Request) {
	p, _ := middleware.PrincipalFrom(r.Context())
	list, err := a.Catalog.ListAccounts(r.Context(), p.HouseholdID)
	if err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]any{"items": list})
}

func (a *API) listCategories(w http.ResponseWriter, r *http.Request) {
	p, _ := middleware.PrincipalFrom(r.Context())
	list, err := a.Catalog.ListCategories(r.Context(), p.HouseholdID)
	if err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]any{"items": list})
}

func (a *API) listStreams(w http.ResponseWriter, r *http.Request) {
	p, _ := middleware.PrincipalFrom(r.Context())
	list, err := a.Catalog.ListIncomeStreams(r.Context(), p.HouseholdID)
	if err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]any{"items": list})
}

type createTxnReq struct {
	AccountID         string  `json:"account_id"`
	TransferAccountID *string `json:"transfer_account_id"`
	CategoryID        *string `json:"category_id"`
	IncomeStreamID    *string `json:"income_stream_id"`
	Direction         string  `json:"direction"`
	Amount            string  `json:"amount"`
	Currency          string  `json:"currency"`
	TxnDate           string  `json:"txn_date"`
	PayeeRaw          string  `json:"payee_raw"`
	Memo              string  `json:"memo"`
}

func (a *API) createTxn(w http.ResponseWriter, r *http.Request) {
	p, _ := middleware.PrincipalFrom(r.Context())
	body, err := io.ReadAll(r.Body)
	if err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "bad_request", "read body")
		return
	}
	reqHash := requestHash(r.Method, r.URL.Path, body)
	idemKey := r.Header.Get("Idempotency-Key")
	if idemKey != "" && a.Idem != nil {
		if stored, err := a.Idem.Get(r.Context(), p.UserID, idemKey); err == nil {
			if stored.RequestHash != reqHash {
				middleware.WriteError(w, http.StatusConflict, "idempotency_conflict", "Idempotency-Key reuse with different body")
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(stored.ResponseCode)
			_, _ = w.Write(stored.ResponseBody)
			return
		}
	}

	var req createTxnReq
	if err := json.Unmarshal(body, &req); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "bad_request", "invalid JSON")
		return
	}
	accID, err := uuid.Parse(req.AccountID)
	if err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "bad_request", "invalid account_id")
		return
	}
	amt, err := domain.ParseMoney(req.Amount)
	if err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "bad_request", "invalid amount")
		return
	}
	currency := req.Currency
	if currency == "" {
		currency = "INR"
	}
	day := time.Now().UTC()
	if req.TxnDate != "" {
		day, err = time.Parse("2006-01-02", req.TxnDate)
		if err != nil {
			middleware.WriteError(w, http.StatusBadRequest, "bad_request", "txn_date must be YYYY-MM-DD")
			return
		}
	}
	in := transactions.CreateInput{
		HouseholdID: p.HouseholdID,
		AccountID:   accID,
		Direction:   domain.Direction(req.Direction),
		Amount:      amt,
		Currency:    currency,
		TxnDate:     day,
		PayeeRaw:    req.PayeeRaw,
		Memo:        req.Memo,
	}
	if req.TransferAccountID != nil && *req.TransferAccountID != "" {
		id, err := uuid.Parse(*req.TransferAccountID)
		if err != nil {
			middleware.WriteError(w, http.StatusBadRequest, "bad_request", "invalid transfer_account_id")
			return
		}
		in.TransferAccountID = &id
	}
	if req.CategoryID != nil && *req.CategoryID != "" {
		id, err := uuid.Parse(*req.CategoryID)
		if err != nil {
			middleware.WriteError(w, http.StatusBadRequest, "bad_request", "invalid category_id")
			return
		}
		in.CategoryID = &id
	}
	if req.IncomeStreamID != nil && *req.IncomeStreamID != "" {
		id, err := uuid.Parse(*req.IncomeStreamID)
		if err != nil {
			middleware.WriteError(w, http.StatusBadRequest, "bad_request", "invalid income_stream_id")
			return
		}
		in.IncomeStreamID = &id
	}

	txn, err := a.Txns.Create(r.Context(), in)
	if err != nil {
		writeDomainErr(w, err)
		return
	}
	resp := map[string]any{"transaction": txnToJSON(txn)}
	payload, _ := json.Marshal(resp)
	if idemKey != "" && a.Idem != nil {
		_ = a.Idem.Put(r.Context(), p.UserID, idemKey, reqHash, http.StatusCreated, payload)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(payload)
}

func (a *API) listTxns(w http.ResponseWriter, r *http.Request) {
	p, _ := middleware.PrincipalFrom(r.Context())
	list, err := a.Txns.List(r.Context(), p.HouseholdID, 50)
	if err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	items := make([]any, 0, len(list))
	for _, t := range list {
		items = append(items, txnToJSON(t))
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *API) getTxn(w http.ResponseWriter, r *http.Request) {
	p, _ := middleware.PrincipalFrom(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "bad_request", "invalid id")
		return
	}
	txn, err := a.Txns.Get(r.Context(), p.HouseholdID, id)
	if err != nil {
		writeDomainErr(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]any{"transaction": txnToJSON(txn)})
}

func (a *API) voidTxn(w http.ResponseWriter, r *http.Request) {
	p, _ := middleware.PrincipalFrom(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "bad_request", "invalid id")
		return
	}
	txn, err := a.Txns.Void(r.Context(), p.HouseholdID, id)
	if err != nil {
		writeDomainErr(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]any{"transaction": txnToJSON(txn)})
}

func txnToJSON(t domain.Transaction) map[string]any {
	m := map[string]any{
		"id":           t.ID,
		"household_id": t.HouseholdID,
		"account_id":   t.AccountID,
		"direction":    t.Direction,
		"amount":       t.Amount.String(),
		"currency":     t.Currency,
		"txn_date":     t.TxnDate.Format("2006-01-02"),
		"payee_raw":    t.PayeeRaw,
		"payee_norm":   t.PayeeNorm,
		"memo":         t.Memo,
		"voided":       t.IsVoided(),
	}
	if t.TransferAccountID != nil {
		m["transfer_account_id"] = t.TransferAccountID
	}
	if t.CategoryID != nil {
		m["category_id"] = t.CategoryID
	}
	if t.IncomeStreamID != nil {
		m["income_stream_id"] = t.IncomeStreamID
	}
	return m
}

func writeDomainErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		middleware.WriteError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, domain.ErrCrossHousehold):
		middleware.WriteError(w, http.StatusBadRequest, "cross_household", err.Error())
	case errors.Is(err, domain.ErrInvalidArgument), errors.Is(err, domain.ErrInvalidAmount),
		errors.Is(err, domain.ErrInvalidDirection), errors.Is(err, domain.ErrInvalidTransfer),
		errors.Is(err, domain.ErrCategoryKindMismatch), errors.Is(err, domain.ErrMissingAccount),
		errors.Is(err, domain.ErrMissingHousehold):
		middleware.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
	case errors.Is(err, domain.ErrVoidedTransaction):
		middleware.WriteError(w, http.StatusConflict, "already_voided", err.Error())
	default:
		middleware.WriteError(w, http.StatusInternalServerError, "internal", err.Error())
	}
}

func requestHash(method, path string, body []byte) string {
	h := sha256.New()
	_, _ = h.Write([]byte(method))
	_, _ = h.Write([]byte("\n"))
	_, _ = h.Write([]byte(path))
	_, _ = h.Write([]byte("\n"))
	_, _ = h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}

func sameSite(s string) http.SameSite {
	switch strings.ToLower(s) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

func clearCookie(w http.ResponseWriter, name string, cfg config.Config) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: name == middleware.SessionCookieName,
		Secure:   cfg.CookieSecure,
		SameSite: sameSite(cfg.CookieSameSite),
	})
}

// Ensure ports import used for Principal in docs — silence if unused.
