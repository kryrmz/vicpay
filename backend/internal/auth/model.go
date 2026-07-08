package auth

import (
	"errors"
	"time"

	"github.com/vicpay/backend/internal/kyc"
)

// Domain errors surfaced to handlers, which map them to HTTP responses.
var (
	ErrInvalidPhone       = errors.New("auth: phone must be E.164")
	ErrWeakPassword       = errors.New("auth: password too weak")
	ErrPhoneTaken         = errors.New("auth: phone already registered")
	ErrInvalidCredentials = errors.New("auth: invalid credentials")
	ErrPhoneNotVerified   = errors.New("auth: phone not verified")
	ErrUserNotFound       = errors.New("auth: user not found")
	ErrInvalidToken       = errors.New("auth: invalid or expired token")
	ErrTokenReuse         = errors.New("auth: refresh token reuse detected")
	ErrSessionExpired     = errors.New("auth: session expired")
)

// Profile is the non-sensitive view of a user returned to clients. The phone is
// masked; raw PII never leaves the server.
type Profile struct {
	ID          string    `json:"id"`
	PhoneMasked string    `json:"phoneMasked"`
	KYCLevel    kyc.Level `json:"kycLevel"`
	Verified    bool      `json:"phoneVerified"`
}

// Session is the result of a successful authentication. The access token is
// returned in the response body (held only in memory by the client); the refresh
// token is delivered via an HttpOnly cookie, not here.
type Session struct {
	Profile      Profile `json:"user"`
	AccessToken  string  `json:"accessToken"`
	refreshToken string  // set out-of-band into the cookie; never serialized
}

// RefreshToken returns the raw refresh token for cookie placement.
func (s Session) RefreshToken() string { return s.refreshToken }

// userRow is the internal representation of a users row.
type userRow struct {
	id            string
	passwordHash  string
	phoneCipher   []byte
	phoneVerified bool
	kycLevel      kyc.Level
	status        string
}

// refreshRow mirrors a refresh_tokens row.
type refreshRow struct {
	jti          string
	userID       string
	familyID     string
	parentJTI    *string
	tokenHash    string
	issuedAt     time.Time
	familyOrigin time.Time
	expiresAt    time.Time
	usedAt       *time.Time
	revokedAt    *time.Time
}
