// Command server is the expense-tracker HTTP API entrypoint.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/krishnamadhavan/expense-tracker/internal/adapters/http/middleware"
	v1 "github.com/krishnamadhavan/expense-tracker/internal/adapters/http/v1"
	"github.com/krishnamadhavan/expense-tracker/internal/adapters/postgres"
	"github.com/krishnamadhavan/expense-tracker/internal/adapters/rulesengine"
	"github.com/krishnamadhavan/expense-tracker/internal/app/categorization"
	"github.com/krishnamadhavan/expense-tracker/internal/app/reports"
	"github.com/krishnamadhavan/expense-tracker/internal/app/budgets"
	appimports "github.com/krishnamadhavan/expense-tracker/internal/app/imports"
	"github.com/krishnamadhavan/expense-tracker/internal/app/auth"
	"github.com/krishnamadhavan/expense-tracker/internal/app/catalog"
	"github.com/krishnamadhavan/expense-tracker/internal/app/transactions"
	"github.com/krishnamadhavan/expense-tracker/internal/config"
	"github.com/krishnamadhavan/expense-tracker/internal/webui"
	"io/fs"
	"strings"
	"github.com/krishnamadhavan/expense-tracker/internal/ports"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		slog.Error("ET_DATABASE_URL is required")
		os.Exit(1)
	}
	if !cfg.CookieSecure {
		slog.Warn("ET_COOKIE_SECURE=false; session cookies will be set without Secure flag")
	}

	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	userRepo := &postgres.UserRepo{Pool: pool}
	sessRepo := &postgres.SessionRepo{Pool: pool}
	tokRepo := &postgres.APITokenRepo{Pool: pool}
	authSvc := &auth.Service{
		Users:       userRepo,
		Sessions:    sessRepo,
		Tokens:      tokRepo,
		AbsoluteTTL: cfg.SessionAbsoluteTTL,
		IdleTTL:     cfg.SessionIdleTTL,
	}
	if err := authSvc.BootstrapIfEmpty(ctx, cfg.BootstrapEmail, cfg.BootstrapPassword); err != nil {
		slog.Error("bootstrap", "err", err)
		os.Exit(1)
	}

	accRepo := &postgres.AccountRepo{Pool: pool}
	catRepo := &postgres.CategoryRepo{Pool: pool}
	streamRepo := &postgres.IncomeStreamRepo{Pool: pool}
	txnRepo := &postgres.TransactionRepo{Pool: pool}
	idemRepo := &postgres.IdempotencyRepo{Pool: pool}

	catalogSvc := &catalog.Service{Accounts: accRepo, Categories: catRepo, IncomeStreams: streamRepo}
	catStore := &postgres.CategorizationStore{Pool: pool}
	var categorizer ports.Categorizer = ports.NoopCategorizer{}
	var catSvc *categorization.Service
	if cfg.CategorizationOn {
		eng := &rulesengine.Engine{Store: catStore}
		categorizer = eng
		catSvc = &categorization.Service{Pool: pool, Engine: eng, Cats: catRepo}
	}
	txnSvc := &transactions.Service{
		Txns:        txnRepo,
		Accounts:    accRepo,
		Categories:  catRepo,
		Streams:     streamRepo,
		Categorizer: categorizer,
	}

	authMW := &middleware.Authenticator{Auth: authSvc, AuthDisabled: cfg.AuthDisabled}
	api := &v1.API{
		Cfg:      cfg,
		Auth:     authSvc,
		Catalog:  catalogSvc,
		Txns:     txnSvc,
		Cat: catSvc,
		Reports: &reports.Service{Pool: pool},
		Budgets: &budgets.Service{Pool: pool},
		Imports: &appimports.Service{Pool: pool, Txns: txnSvc},
		Idem:     idemRepo,
		AuthMW:   authMW,
		LoginRL:  middleware.NewLoginRateLimiter(cfg.LoginRateLimit, cfg.LoginRateWindow),
	}

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Referrer-Policy", "no-referrer")
			next.ServeHTTP(w, r)
		})
	})
	if len(cfg.CORSOrigins) > 0 {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   cfg.CORSOrigins,
			AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "Idempotency-Key"},
			AllowCredentials: true,
		}))
	}
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Get("/readyz", func(w http.ResponseWriter, req *http.Request) {
		if err := pool.Ping(req.Context()); err != nil {
			middleware.WriteError(w, http.StatusServiceUnavailable, "not_ready", "database unavailable")
			return
		}
		middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	r.Get("/api/openapi.yaml", serveOpenAPI)

	r.Group(func(r chi.Router) {
		r.Use(authMW.Authenticate)
		api.Routes(r)
	})

	// Embedded SPA (PR13): /assets/* and SPA fallback for non-API GETs
	static, err := fs.Sub(webui.Dist, "dist")
	if err != nil {
		slog.Error("webui embed", "err", err)
		os.Exit(1)
	}
	fileServer := http.FileServer(http.FS(static))
	r.Handle("/assets/*", http.StripPrefix("/", fileServer))
	r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
		if strings.HasPrefix(req.URL.Path, "/api") || req.URL.Path == "/healthz" || req.URL.Path == "/readyz" {
			http.NotFound(w, req)
			return
		}
		if req.URL.Path != "/" && !strings.Contains(req.URL.Path[1:], ".") {
			// SPA client route fallback
			b, err := fs.ReadFile(static, "index.html")
			if err != nil {
				http.NotFound(w, req)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(b)
			return
		}
		fileServer.ServeHTTP(w, req)
	})

	srv := &http.Server{Addr: cfg.HTTPAddr, Handler: r, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		slog.Info("server listening", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	slog.Info("server stopped")
}

func serveOpenAPI(w http.ResponseWriter, r *http.Request) {
	paths := []string{"api/openapi/openapi.yaml", "/app/api/openapi/openapi.yaml"}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			http.ServeFile(w, r, p)
			return
		}
	}
	middleware.WriteError(w, http.StatusNotFound, "not_found", "openapi.yaml not found")
}
