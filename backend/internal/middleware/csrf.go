package middleware

import (
	"net/http"
	"net/url"

	"github.com/vicpay/backend/pkg/response"
)

// CSRF guards cookie-authenticated, state-changing requests by checking the
// Origin (falling back to Referer) against an allowlist. Combined with a
// SameSite=Strict refresh cookie this is defense in depth. Safe methods pass
// through untouched. This middleware is mounted from day one on the endpoints
// that trust the refresh cookie.
func CSRF(allowedOrigins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isSafeMethod(r.Method) {
				next.ServeHTTP(w, r)
				return
			}
			origin := requestOrigin(r)
			if origin == "" {
				response.Fail(w, http.StatusForbidden, "csrf", "missing Origin/Referer on state-changing request")
				return
			}
			if _, ok := allowed[origin]; !ok {
				response.Fail(w, http.StatusForbidden, "csrf", "origin not allowed")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func isSafeMethod(m string) bool {
	switch m {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

// requestOrigin returns the scheme://host of the Origin header, or the Referer's
// origin if Origin is absent.
func requestOrigin(r *http.Request) string {
	if o := r.Header.Get("Origin"); o != "" {
		return o
	}
	if ref := r.Header.Get("Referer"); ref != "" {
		if u, err := url.Parse(ref); err == nil && u.Scheme != "" && u.Host != "" {
			return u.Scheme + "://" + u.Host
		}
	}
	return ""
}
