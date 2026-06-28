package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// LoginRateLimiter is a simple per-IP limiter for single-replica deploys.
type LoginRateLimiter struct {
	mu      sync.Mutex
	hits    map[string][]time.Time
	limit   int
	window  time.Duration
}

// NewLoginRateLimiter allows limit requests per window per IP.
func NewLoginRateLimiter(limit int, window time.Duration) *LoginRateLimiter {
	if limit <= 0 {
		limit = 10
	}
	if window <= 0 {
		window = 15 * time.Minute
	}
	return &LoginRateLimiter{hits: make(map[string][]time.Time), limit: limit, window: window}
}

// Allow reports whether the IP may proceed.
func (l *LoginRateLimiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	cut := now.Add(-l.window)
	arr := l.hits[ip]
	n := 0
	for _, t := range arr {
		if t.After(cut) {
			arr[n] = t
			n++
		}
	}
	arr = arr[:n]
	if len(arr) >= l.limit {
		l.hits[ip] = arr
		return false
	}
	arr = append(arr, now)
	l.hits[ip] = arr
	return true
}

// ClientIP extracts a best-effort client IP.
func ClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
