// Package authstore holds Postgres-backed implementations of the auth-adjacent
// stores (OTP challenges) and a development OTP sender.
package authstore

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vicpay/backend/internal/otp"
)

// OTPStore persists OTP challenges in the otp_challenges table.
type OTPStore struct{ pool *pgxpool.Pool }

// NewOTPStore builds an OTPStore.
func NewOTPStore(pool *pgxpool.Pool) *OTPStore { return &OTPStore{pool: pool} }

// Save inserts a new challenge.
func (s *OTPStore) Save(ctx context.Context, c *otp.Challenge) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO otp_challenges (id, recipient, purpose, code_hash, attempts, max_attempts, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		c.ID, c.Recipient, string(c.Purpose), c.CodeHash, c.Attempts, c.MaxAttempts, c.ExpiresAt)
	return err
}

// Active returns the most recent unconsumed challenge for (recipient, purpose).
func (s *OTPStore) Active(ctx context.Context, recipient string, purpose otp.Purpose) (*otp.Challenge, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT id, recipient, purpose, code_hash, attempts, max_attempts, expires_at, consumed_at
		 FROM otp_challenges
		 WHERE recipient = $1 AND purpose = $2 AND consumed_at IS NULL
		 ORDER BY created_at DESC LIMIT 1`, recipient, string(purpose))
	var c otp.Challenge
	var purposeStr string
	if err := row.Scan(&c.ID, &c.Recipient, &purposeStr, &c.CodeHash, &c.Attempts,
		&c.MaxAttempts, &c.ExpiresAt, &c.ConsumedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	c.Purpose = otp.Purpose(purposeStr)
	return &c, nil
}

// IncrementAttempts bumps the failed-attempt counter.
func (s *OTPStore) IncrementAttempts(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `UPDATE otp_challenges SET attempts = attempts + 1 WHERE id = $1`, id)
	return err
}

// Consume marks a challenge used (single-use).
func (s *OTPStore) Consume(ctx context.Context, id string, at time.Time) error {
	_, err := s.pool.Exec(ctx, `UPDATE otp_challenges SET consumed_at = $2 WHERE id = $1`, id, at)
	return err
}

// LogSender is a development OTP sender. When echo is true it logs the code so a
// developer can complete the flow without a real SMS provider; production wires
// a real provider and must set echo to false. The code is never returned to the
// client in any case.
type LogSender struct {
	echo   bool
	logger *slog.Logger
}

// NewLogSender builds a LogSender.
func NewLogSender(echo bool, logger *slog.Logger) *LogSender {
	return &LogSender{echo: echo, logger: logger}
}

// Send "delivers" the code. In echo mode it logs it; otherwise it only records
// that a code was dispatched.
func (s *LogSender) Send(_ context.Context, recipient, code string) error {
	if s.echo {
		s.logger.Warn("otp dev echo (never enable in production)", "recipient", recipient, "code", code)
		return nil
	}
	s.logger.Info("otp dispatched", "recipient", recipient)
	return nil
}
