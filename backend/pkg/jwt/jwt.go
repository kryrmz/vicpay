// Package jwt issues and verifies short-lived access tokens and long-lived
// refresh tokens. Access and refresh tokens are NOT interchangeable: they carry
// a distinct "typ" claim and are rejected if presented in the wrong role.
// Refresh tokens carry a family id and parent jti so the server can detect
// token reuse and revoke the whole family.
package jwt

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenType distinguishes access from refresh tokens.
type TokenType string

const (
	// Access tokens authorize API calls; short lived.
	Access TokenType = "access"
	// Refresh tokens mint new access tokens; long lived, rotated on use.
	Refresh TokenType = "refresh"
)

// ErrWrongType is returned when a token's typ claim does not match expectations.
var ErrWrongType = errors.New("jwt: wrong token type")

// Claims is the VicPay JWT payload.
type Claims struct {
	Type     TokenType `json:"typ"`
	FamilyID string    `json:"fid,omitempty"`
	ParentID string    `json:"pjti,omitempty"`
	jwt.RegisteredClaims
}

// Manager signs and parses tokens with a single HMAC secret.
type Manager struct {
	secret    []byte
	issuer    string
	accessTTL time.Duration
	now       func() time.Time
}

// NewManager builds a Manager. accessTTL bounds the access token lifetime.
func NewManager(secret, issuer string, accessTTL time.Duration) *Manager {
	return &Manager{secret: []byte(secret), issuer: issuer, accessTTL: accessTTL, now: time.Now}
}

// IssueAccess signs an access token for the subject with a fresh jti.
func (m *Manager) IssueAccess(subject, jti string) (string, error) {
	now := m.now()
	return m.sign(Claims{
		Type: Access,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			ID:        jti,
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTTL)),
		},
	})
}

// IssueRefresh signs a refresh token, threading the family and parent jti used
// for rotation-reuse detection.
func (m *Manager) IssueRefresh(subject, jti, familyID, parentID string, ttl time.Duration) (string, error) {
	now := m.now()
	return m.sign(Claims{
		Type:     Refresh,
		FamilyID: familyID,
		ParentID: parentID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			ID:        jti,
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	})
}

// Parse validates a token's signature, expiry and issuer and asserts its type.
func (m *Manager) Parse(token string, want TokenType) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("jwt: unexpected signing method %v", t.Header["alg"])
		}
		return m.secret, nil
	}, jwt.WithIssuer(m.issuer), jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return nil, err
	}
	if claims.Type != want {
		return nil, ErrWrongType
	}
	return claims, nil
}

func (m *Manager) sign(c Claims) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(m.secret)
}

// Fingerprint returns a stable sha256 hex of a token, for storing a reference
// in the database without persisting the raw token.
func Fingerprint(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
