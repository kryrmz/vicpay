// Package transfer moves money between users over the ledger. Every transfer is
// a single balanced posting (debit sender wallet, credit recipient wallet),
// guarded by a balance check and the sender's KYC spending limits. All value
// movement goes through the ledger engine -- no domain mutates a balance
// directly.
package transfer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vicpay/backend/internal/kyc"
	"github.com/vicpay/backend/internal/ledger"
	"github.com/vicpay/backend/internal/pii"
	"github.com/vicpay/backend/pkg/money"
	"github.com/vicpay/backend/pkg/validator"
)

// Domain errors.
var (
	ErrInvalidAmount       = errors.New("transfer: amount must be positive")
	ErrUnsupportedCurrency = errors.New("transfer: unsupported currency")
	ErrInvalidRecipient    = errors.New("transfer: recipient phone is invalid")
	ErrRecipientNotFound   = errors.New("transfer: recipient not found")
	ErrRecipientUnverified = errors.New("transfer: recipient has not verified their phone")
	ErrSelfTransfer        = errors.New("transfer: cannot transfer to yourself")
	ErrInsufficientFunds   = errors.New("transfer: insufficient funds")
	ErrLimitExceeded       = errors.New("transfer: KYC spending limit exceeded")
)

// Result is the outcome of a successful money movement.
type Result struct {
	PostingID       string `json:"postingId"`
	NewBalanceMinor int64  `json:"newBalanceMinor"`
	Currency        string `json:"currency"`
}

// Service performs transfers and demo top-ups.
type Service struct {
	pool   *pgxpool.Pool
	engine *ledger.Engine
	cipher *pii.Cipher
	now    func() time.Time
}

// NewService wires the transfer service.
func NewService(pool *pgxpool.Pool, engine *ledger.Engine, cipher *pii.Cipher) *Service {
	return &Service{pool: pool, engine: engine, cipher: cipher, now: time.Now}
}

// Transfer sends amountMinor from the sender to the user identified by toPhone.
// idempotencyKey makes retries safe; if empty, one is generated (not retry-safe).
func (s *Service) Transfer(ctx context.Context, fromUserID, toPhone string, amountMinor int64, currency, idempotencyKey string) (*Result, error) {
	if amountMinor <= 0 {
		return nil, ErrInvalidAmount
	}
	if !money.IsSupported(money.Currency(currency)) {
		return nil, ErrUnsupportedCurrency
	}
	if !validator.PhoneE164(toPhone) {
		return nil, ErrInvalidRecipient
	}

	recipientID, verified, err := s.userByPhone(ctx, toPhone)
	if err != nil {
		return nil, err
	}
	if recipientID == "" {
		return nil, ErrRecipientNotFound
	}
	if recipientID == fromUserID {
		return nil, ErrSelfTransfer
	}
	if !verified {
		return nil, ErrRecipientUnverified
	}

	balance, err := s.engine.WalletBalanceMinor(ctx, fromUserID, currency)
	if err != nil {
		return nil, err
	}
	if balance < amountMinor {
		return nil, ErrInsufficientFunds
	}

	if err := s.checkLimits(ctx, fromUserID, currency, amountMinor); err != nil {
		return nil, err
	}

	posting, err := s.engine.Post(ctx, ledger.PostingInput{
		IdempotencyKey: keyOrRandom(idempotencyKey),
		Description:    "p2p transfer",
		Metadata:       map[string]any{"type": "p2p", "from": fromUserID, "to": recipientID},
		Entries: []ledger.EntryInput{
			{Account: ledger.Wallet(fromUserID), Direction: ledger.Debit, AmountMinor: amountMinor, Currency: currency},
			{Account: ledger.Wallet(recipientID), Direction: ledger.Credit, AmountMinor: amountMinor, Currency: currency},
		},
	})
	if err != nil {
		return nil, err
	}
	newBalance, _ := s.engine.WalletBalanceMinor(ctx, fromUserID, currency)
	return &Result{PostingID: posting.ID, NewBalanceMinor: newBalance, Currency: currency}, nil
}

// TopUp credits a user's wallet from the external system account. It is a demo
// stand-in for a real on-ramp and must be gated to non-production environments
// by the caller.
func (s *Service) TopUp(ctx context.Context, userID string, amountMinor int64, currency, idempotencyKey string) (*Result, error) {
	if amountMinor <= 0 {
		return nil, ErrInvalidAmount
	}
	if !money.IsSupported(money.Currency(currency)) {
		return nil, ErrUnsupportedCurrency
	}
	posting, err := s.engine.Post(ctx, ledger.PostingInput{
		IdempotencyKey: keyOrRandom(idempotencyKey),
		Description:    "demo top-up",
		Metadata:       map[string]any{"type": "topup", "to": userID},
		Entries: []ledger.EntryInput{
			{Account: ledger.System("SYSTEM:EXTERNAL:" + currency), Direction: ledger.Debit, AmountMinor: amountMinor, Currency: currency},
			{Account: ledger.Wallet(userID), Direction: ledger.Credit, AmountMinor: amountMinor, Currency: currency},
		},
	})
	if err != nil {
		return nil, err
	}
	newBalance, _ := s.engine.WalletBalanceMinor(ctx, userID, currency)
	return &Result{PostingID: posting.ID, NewBalanceMinor: newBalance, Currency: currency}, nil
}

// userByPhone resolves a user id and verification flag from a phone number via
// its blind index.
func (s *Service) userByPhone(ctx context.Context, phone string) (string, bool, error) {
	var id string
	var verified bool
	err := s.pool.QueryRow(ctx,
		`SELECT id, phone_verified FROM users WHERE phone_hmac = $1`,
		s.cipher.BlindIndex(phone)).Scan(&id, &verified)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return id, verified, nil
}

// checkLimits enforces the sender's KYC daily and monthly spending ceilings.
// Spending is the sum of debits from the user's wallets in the currency over the
// period. Per-currency limit tables are a follow-up; the reference thresholds
// are applied per currency for now.
func (s *Service) checkLimits(ctx context.Context, userID, currency string, amountMinor int64) error {
	var level kyc.Level
	if err := s.pool.QueryRow(ctx, `SELECT kyc_level FROM users WHERE id = $1`, userID).Scan(&level); err != nil {
		return err
	}
	limits := kyc.LimitsFor(level)

	now := s.now().UTC()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	daySum, err := s.outgoingSince(ctx, userID, currency, startOfDay)
	if err != nil {
		return err
	}
	if daySum+amountMinor > limits.DailyMinor {
		return fmt.Errorf("%w: daily", ErrLimitExceeded)
	}
	monthSum, err := s.outgoingSince(ctx, userID, currency, startOfMonth)
	if err != nil {
		return err
	}
	if monthSum+amountMinor > limits.MonthlyMinor {
		return fmt.Errorf("%w: monthly", ErrLimitExceeded)
	}
	return nil
}

// outgoingSince sums a user's wallet debits in a currency since a point in time.
func (s *Service) outgoingSince(ctx context.Context, userID, currency string, since time.Time) (int64, error) {
	var sum int64
	err := s.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(e.amount_minor), 0)
		 FROM journal_entries e
		 JOIN ledger_accounts a ON a.id = e.account_id
		 WHERE a.type = 'user_wallet' AND a.user_id = $1
		   AND e.currency = $2 AND e.direction = 'debit' AND e.created_at >= $3`,
		userID, currency, since).Scan(&sum)
	return sum, err
}

func keyOrRandom(key string) string {
	if key != "" {
		return key
	}
	return uuid.NewString()
}
