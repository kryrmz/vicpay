package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/vicpay/backend/pkg/response"
)

// RateLimiter is a simple fixed-window, in-memory limiter keyed by client IP. It
// fails CLOSED under no external dependency (there is none), and separate
// buckets can be created per route group so, e.g., login cannot exhaust the
// budget of register. For multi-instance deployments this is swapped for a Redis
// limiter behind the same middleware shape.
type RateLimiter struct {
	mu       sync.Mutex
	counters map[string]*window
	limit    int
	window   time.Duration
	now      func() time.Time
}

type window struct {
	count int
	reset time.Time
}

// NewRateLimiter builds a limiter allowing `limit` requests per `per` window.
func NewRateLimiter(limit int, per time.Duration) *RateLimiter {
	return &RateLimiter{
		counters: map[string]*window{},
		limit:    limit,
		window:   per,
		now:      time.Now,
	}
}

// Middleware enforces the limit, returning 429 when exceeded.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.allow(clientIP(r)) {
			w.Header().Set("Retry-After", "60")
			response.Fail(w, http.StatusTooManyRequests, "rate_limited", "too many requests")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := rl.now()
	win, ok := rl.counters[key]
	if !ok || now.After(win.reset) {
		rl.counters[key] = &window{count: 1, reset: now.Add(rl.window)}
		return true
	}
	if win.count >= rl.limit {
		return false
	}
	win.count++
	return true
}

// clientIP extracts the remote IP, preferring the last hop's RemoteAddr. A
// production deployment behind a trusted proxy should parse a validated
// X-Forwarded-For; we intentionally do not trust that header by default.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
