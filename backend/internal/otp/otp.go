// Package otp implements real, server-side one-time-passcode challenges for
// phone verification. This deliberately replaces KiramoPay's cosmetic client
// setTimeout: a code is generated on the server, hashed at rest, single-use,
// expiring, rate-limited, and delivery is abstracted behind a Sender so a real
// SMS provider (Twilio/Sinch/...) drops in for production while dev uses a log
// sender. The plaintext code is NEVER returned in an API response.
package otp

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"math/big"
	"time"
)

// Purpose scopes a challenge so a code minted for one flow cannot satisfy another.
type Purpose string

const (
	// PurposePhoneVerify verifies phone ownership during onboarding.
	PurposePhoneVerify Purpose = "phone_verify"
	// PurposeAccountRecovery re-verifies a phone during account recovery.
	PurposeAccountRecovery Purpose = "account_recovery"
)

// Common errors.
var (
	ErrNotFound    = errors.New("otp: no active challenge")
	ErrExpired     = errors.New("otp: challenge expired")
	ErrTooMany     = errors.New("otp: too many attempts")
	ErrCodeInvalid = errors.New("otp: code invalid")
)

// Config tunes the challenge lifecycle.
type Config struct {
	CodeLength  int
	TTL         time.Duration
	MaxAttempts int
}

// DefaultConfig is a 6-digit code valid for 5 minutes with 5 attempts.
var DefaultConfig = Config{CodeLength: 6, TTL: 5 * time.Minute, MaxAttempts: 5}

// Challenge is the persisted server-side record. The code is stored only as a
// sha256 hash; the plaintext exists just long enough to hand to the Sender.
type Challenge struct {
	ID          string
	Recipient   string // E.164 phone
	Purpose     Purpose
	CodeHash    string
	ExpiresAt   time.Time
	Attempts    int
	MaxAttempts int
	ConsumedAt  *time.Time
}

// Sender delivers a code out-of-band (SMS). Implementations must not log the code.
type Sender interface {
	Send(ctx context.Context, recipient, code string) error
}

// Store persists challenges. A Postgres implementation lives in the auth layer;
// tests use an in-memory fake.
type Store interface {
	Save(ctx context.Context, c *Challenge) error
	Active(ctx context.Context, recipient string, purpose Purpose) (*Challenge, error)
	IncrementAttempts(ctx context.Context, id string) error
	Consume(ctx context.Context, id string, at time.Time) error
}

// Service issues and verifies challenges.
type Service struct {
	cfg    Config
	store  Store
	sender Sender
	now    func() time.Time
}

// NewService wires a Service. If cfg is the zero value, DefaultConfig is used.
func NewService(store Store, sender Sender, cfg Config) *Service {
	if cfg.CodeLength == 0 {
		cfg = DefaultConfig
	}
	return &Service{cfg: cfg, store: store, sender: sender, now: time.Now}
}

// Issue creates a challenge, persists its hash, and dispatches the code. It
// returns the challenge id only; the plaintext never leaves this method except
// through the Sender.
func (s *Service) Issue(ctx context.Context, id, recipient string, purpose Purpose) (string, error) {
	code, err := numericCode(s.cfg.CodeLength)
	if err != nil {
		return "", err
	}
	now := s.now()
	c := &Challenge{
		ID:          id,
		Recipient:   recipient,
		Purpose:     purpose,
		CodeHash:    hashCode(code),
		ExpiresAt:   now.Add(s.cfg.TTL),
		MaxAttempts: s.cfg.MaxAttempts,
	}
	if err := s.store.Save(ctx, c); err != nil {
		return "", err
	}
	if err := s.sender.Send(ctx, recipient, code); err != nil {
		return "", err
	}
	return id, nil
}

// Verify checks a submitted code against the active challenge for the recipient
// and purpose. On success the challenge is consumed (single-use). Wrong codes
// count against MaxAttempts.
func (s *Service) Verify(ctx context.Context, recipient string, purpose Purpose, code string) error {
	c, err := s.store.Active(ctx, recipient, purpose)
	if err != nil {
		return err
	}
	if c == nil || c.ConsumedAt != nil {
		return ErrNotFound
	}
	if s.now().After(c.ExpiresAt) {
		return ErrExpired
	}
	if c.Attempts >= c.MaxAttempts {
		return ErrTooMany
	}
	if subtle.ConstantTimeCompare([]byte(c.CodeHash), []byte(hashCode(code))) != 1 {
		_ = s.store.IncrementAttempts(ctx, c.ID)
		return ErrCodeInvalid
	}
	return s.store.Consume(ctx, c.ID, s.now())
}

func hashCode(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}

// numericCode returns a zero-padded random decimal string of the given length,
// drawn from a cryptographically secure source.
func numericCode(length int) (string, error) {
	const digits = "0123456789"
	buf := make([]byte, length)
	for i := range buf {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		buf[i] = digits[n.Int64()]
	}
	return string(buf), nil
}
