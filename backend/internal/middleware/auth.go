package middleware

import (
	"context"
	"net/http"
	"strings"

	pkgjwt "github.com/vicpay/backend/pkg/jwt"
	"github.com/vicpay/backend/pkg/response"
)

// Authenticator validates a Bearer access token and injects the user id. It
// fails closed: a missing or invalid token yields 401.
func Authenticator(jwtMgr *pkgjwt.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearer(r)
			if !ok {
				response.Fail(w, http.StatusUnauthorized, "unauthorized", "missing bearer token")
				return
			}
			claims, err := jwtMgr.Parse(token, pkgjwt.Access)
			if err != nil {
				response.Fail(w, http.StatusUnauthorized, "unauthorized", "invalid or expired token")
				return
			}
			ctx := context.WithValue(r.Context(), ctxUserID, claims.Subject)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func bearer(r *http.Request) (string, bool) {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(h) <= len(prefix) || !strings.EqualFold(h[:len(prefix)], prefix) {
		return "", false
	}
	return strings.TrimSpace(h[len(prefix):]), true
}
