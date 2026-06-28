package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds ET_* settings for the API server.
type Config struct {
	HTTPAddr            string
	DatabaseURL         string
	BootstrapPassword   string
	BootstrapEmail      string
	CookieSecure        bool
	CookieSameSite      string
	CORSOrigins         []string
	AuthDisabled        bool // tests only
	CategorizationOn    bool
	SessionAbsoluteTTL  time.Duration
	SessionIdleTTL      time.Duration
	LoginRateLimit      int
	LoginRateWindow     time.Duration
}

// Load reads configuration from the environment.
func Load() Config {
	return Config{
		HTTPAddr:           envOr("ET_HTTP_ADDR", ":8080"),
		DatabaseURL:        firstNonEmpty(os.Getenv("ET_DATABASE_URL"), os.Getenv("DATABASE_URL")),
		BootstrapPassword:  os.Getenv("ET_BOOTSTRAP_PASSWORD"),
		BootstrapEmail:     envOr("ET_BOOTSTRAP_EMAIL", "admin@localhost"),
		CookieSecure:       envBool("ET_COOKIE_SECURE", true),
		CookieSameSite:     envOr("ET_COOKIE_SAMESITE", "Lax"),
		CORSOrigins:        splitCSV(os.Getenv("ET_CORS_ORIGINS")),
		AuthDisabled:       envBool("ET_AUTH_DISABLED", false),
		CategorizationOn:   envBool("ET_CATEGORIZATION_ENABLED", true),
		SessionAbsoluteTTL: 12 * time.Hour,
		SessionIdleTTL:     2 * time.Hour,
		LoginRateLimit:     10,
		LoginRateWindow:    15 * time.Minute,
	}
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func envBool(k string, d bool) bool {
	v := os.Getenv(k)
	if v == "" {
		return d
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return d
	}
	return b
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
