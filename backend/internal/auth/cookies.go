package auth

import (
	"net/http"
	"time"
)

// Cookie names. The __Host- prefix requires Secure, Path=/ and no Domain, which
// pins the cookie to the exact origin -- the strongest first-party guarantee.
const (
	cookieProd = "__Host-vp_refresh"
	cookieDev  = "vp_refresh"
)

// CookieWriter emits and clears the refresh cookie. In development it drops the
// Secure attribute (and the __Host- prefix) so the flow works over plain HTTP.
type CookieWriter struct {
	secure bool
	maxAge time.Duration
}

// NewCookieWriter builds a CookieWriter. secure should be true outside dev.
func NewCookieWriter(secure bool, maxAge time.Duration) *CookieWriter {
	return &CookieWriter{secure: secure, maxAge: maxAge}
}

func (c *CookieWriter) name() string {
	if c.secure {
		return cookieProd
	}
	return cookieDev
}

// Set writes the refresh token as an HttpOnly, SameSite=Strict cookie.
func (c *CookieWriter) Set(w http.ResponseWriter, token string) {
	// #nosec G124 -- HttpOnly and SameSite=Strict are always set; Secure is
	// toggled by environment and is only false for local HTTP development.
	http.SetCookie(w, &http.Cookie{
		Name:     c.name(),
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   c.secure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(c.maxAge.Seconds()),
	})
}

// Clear expires the refresh cookie.
func (c *CookieWriter) Clear(w http.ResponseWriter) {
	// #nosec G124 -- HttpOnly and SameSite=Strict are always set; Secure is
	// toggled by environment and is only false for local HTTP development.
	http.SetCookie(w, &http.Cookie{
		Name:     c.name(),
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   c.secure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}

// Read extracts the refresh token from the request cookie, if present.
func (c *CookieWriter) Read(r *http.Request) (string, bool) {
	ck, err := r.Cookie(c.name())
	if err != nil || ck.Value == "" {
		return "", false
	}
	return ck.Value, true
}
