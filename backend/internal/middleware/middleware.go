// Package middleware holds cross-cutting HTTP middleware: panic recovery,
// request ids, bearer-token authentication, CSRF origin checks, and rate
// limiting. Security middleware here is meant to be mounted from day one -- the
// predecessor project shipped lockout and CSRF code that was never wired to the
// router.
package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/vicpay/backend/pkg/response"
)

type ctxKey int

const (
	ctxUserID ctxKey = iota
	ctxRequestID
)

// UserID returns the authenticated user id from the context, if any.
func UserID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(ctxUserID).(string)
	return id, ok && id != ""
}

// RequestID returns the per-request id from the context, if any.
func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(ctxRequestID).(string)
	return id
}

// Recoverer converts a panic into a 500 without leaking internals.
func Recoverer(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("panic recovered", "err", rec, "path", r.URL.Path, "request_id", RequestID(r.Context()))
					response.Fail(w, http.StatusInternalServerError, "internal", "internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// RequestIDMiddleware assigns a request id, honoring an inbound X-Request-Id.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" {
			id = uuid.NewString()
		}
		w.Header().Set("X-Request-Id", id)
		ctx := context.WithValue(r.Context(), ctxRequestID, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
