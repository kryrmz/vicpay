// Package ledger is VicPay's double-entry accounting engine. Every movement of
// value is a balanced posting of two or more entries; the journal is append-only
// (enforced by database triggers) and user-wallet balances are a cache kept in
// step with the journal inside the same transaction.
//
// Concurrency follows KiramoPay's proven design: READ COMMITTED plus
// SELECT ... FOR UPDATE on the affected user-wallet rows, taken in a
// deterministic id order to avoid deadlocks, with bounded retries on
// serialization/deadlock errors. This lets "hot" accounts queue on a row lock
// rather than abort under contention.
package ledger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Direction is the side of an entry.
type Direction string

const (
	// Debit decreases a user wallet's balance / increases a system account draw.
	Debit Direction = "debit"
	// Credit increases a user wallet's balance.
	Credit Direction = "credit"
)

// Sentinel errors.
var (
	ErrTooFewEntries   = errors.New("ledger: a posting needs at least two entries")
	ErrUnbalanced      = errors.New("ledger: posting does not balance per currency")
	ErrAccountNotFound = errors.New("ledger: account not found")
	ErrBadAmount       = errors.New("ledger: entry amount must be positive")
)

// maxRetries bounds serialization/deadlock retries.
const maxRetries = 8

// AccountRef points at either a named system account or a user's wallet in a
// given currency. Exactly one form must be used per entry.
type AccountRef struct {
	SystemCode string // e.g. "SYSTEM:EXTERNAL:USD"; empty for a user wallet
	UserID     string // set for a user wallet
}

// System builds a reference to a named system account.
func System(code string) AccountRef { return AccountRef{SystemCode: code} }

// Wallet builds a reference to a user's wallet.
func Wallet(userID string) AccountRef { return AccountRef{UserID: userID} }

func (r AccountRef) isSystem() bool { return r.SystemCode != "" }

// EntryInput is one leg of a posting.
type EntryInput struct {
	Account     AccountRef
	Direction   Direction
	AmountMinor int64
	Currency    string
}

// PostingInput describes a balanced financial event to record.
type PostingInput struct {
	IdempotencyKey string
	Description    string
	Metadata       map[string]any
	Entries        []EntryInput
}

// Posting is a recorded posting.
type Posting struct {
	ID             string
	IdempotencyKey string
	Replayed       bool // true when returned from an idempotent replay
}

// Engine records postings against a Postgres pool.
type Engine struct {
	pool *pgxpool.Pool
}

// New builds an Engine over the given pool.
func New(pool *pgxpool.Pool) *Engine { return &Engine{pool: pool} }

// Validate checks a posting in memory before any database work: at least two
// entries, all positive amounts, and debits == credits for every currency.
func Validate(in PostingInput) error {
	if len(in.Entries) < 2 {
		return ErrTooFewEntries
	}
	net := map[string]int64{}
	for _, e := range in.Entries {
		if e.AmountMinor <= 0 {
			return ErrBadAmount
		}
		switch e.Direction {
		case Debit:
			net[e.Currency] -= e.AmountMinor
		case Credit:
			net[e.Currency] += e.AmountMinor
		default:
			return fmt.Errorf("ledger: invalid direction %q", e.Direction)
		}
	}
	for _, v := range net {
		if v != 0 {
			return ErrUnbalanced
		}
	}
	return nil
}

// Post records a posting atomically and idempotently. Re-posting the same
// IdempotencyKey returns the original posting with Replayed=true.
func (e *Engine) Post(ctx context.Context, in PostingInput) (*Posting, error) {
	if err := Validate(in); err != nil {
		return nil, err
	}

	accountIDs, walletLockIDs, err := e.resolveAccounts(ctx, in)
	if err != nil {
		return nil, err
	}
	// Deterministic lock order prevents deadlocks between concurrent postings.
	sort.Strings(walletLockIDs)

	// Compute per-wallet cache deltas from the input.
	deltas := map[string]int64{}
	for i, entry := range in.Entries {
		if entry.Account.isSystem() {
			continue
		}
		id := accountIDs[i]
		if entry.Direction == Credit {
			deltas[id] += entry.AmountMinor
		} else {
			deltas[id] -= entry.AmountMinor
		}
	}

	metaJSON, err := json.Marshal(orEmpty(in.Metadata))
	if err != nil {
		return nil, fmt.Errorf("ledger: marshal metadata: %w", err)
	}

	var last error
	for attempt := 0; attempt < maxRetries; attempt++ {
		posting, err := e.postOnce(ctx, in, accountIDs, walletLockIDs, deltas, metaJSON)
		if err == nil {
			return posting, nil
		}
		if replay, ok := e.idempotentReplay(ctx, err, in.IdempotencyKey); ok {
			return replay, nil
		}
		if !isRetryable(err) {
			return nil, err
		}
		last = err
		backoff(ctx, attempt)
	}
	return nil, fmt.Errorf("ledger: exhausted retries: %w", last)
}

func (e *Engine) postOnce(
	ctx context.Context,
	in PostingInput,
	accountIDs []string,
	walletLockIDs []string,
	deltas map[string]int64,
	metaJSON []byte,
) (*Posting, error) {
	tx, err := e.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Lock affected user wallets in deterministic order.
	if len(walletLockIDs) > 0 {
		if _, err := tx.Exec(ctx,
			`SELECT id FROM ledger_accounts WHERE id = ANY($1) ORDER BY id FOR UPDATE`,
			walletLockIDs); err != nil {
			return nil, err
		}
	}

	var postingID string
	if err := tx.QueryRow(ctx,
		`INSERT INTO journal_postings (idempotency_key, description, metadata)
		 VALUES ($1, $2, $3) RETURNING id`,
		in.IdempotencyKey, in.Description, metaJSON,
	).Scan(&postingID); err != nil {
		return nil, err
	}

	for i, entry := range in.Entries {
		if _, err := tx.Exec(ctx,
			`INSERT INTO journal_entries (posting_id, account_id, direction, amount_minor, currency)
			 VALUES ($1, $2, $3, $4, $5)`,
			postingID, accountIDs[i], entry.Direction, entry.AmountMinor, entry.Currency,
		); err != nil {
			return nil, err
		}
	}

	for accountID, delta := range deltas {
		if _, err := tx.Exec(ctx,
			`UPDATE ledger_accounts SET cached_balance_minor = cached_balance_minor + $2 WHERE id = $1`,
			accountID, delta); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &Posting{ID: postingID, IdempotencyKey: in.IdempotencyKey}, nil
}

// resolveAccounts maps each entry to a concrete account id, auto-provisioning a
// user wallet for (user, currency) when needed. It returns the per-entry account
// ids and the distinct set of user-wallet ids that must be locked.
func (e *Engine) resolveAccounts(ctx context.Context, in PostingInput) ([]string, []string, error) {
	ids := make([]string, len(in.Entries))
	lockSet := map[string]struct{}{}
	for i, entry := range in.Entries {
		var id string
		var err error
		if entry.Account.isSystem() {
			id, err = e.systemAccountID(ctx, entry.Account.SystemCode)
		} else {
			id, err = e.userWalletID(ctx, entry.Account.UserID, entry.Currency)
			if err == nil {
				lockSet[id] = struct{}{}
			}
		}
		if err != nil {
			return nil, nil, err
		}
		ids[i] = id
	}
	locks := make([]string, 0, len(lockSet))
	for id := range lockSet {
		locks = append(locks, id)
	}
	return ids, locks, nil
}

func (e *Engine) systemAccountID(ctx context.Context, code string) (string, error) {
	var id string
	err := e.pool.QueryRow(ctx, `SELECT id FROM ledger_accounts WHERE code = $1`, code).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("%w: system account %q", ErrAccountNotFound, code)
	}
	return id, err
}

// userWalletID returns the wallet id for (user, currency), creating it if absent.
func (e *Engine) userWalletID(ctx context.Context, userID, currency string) (string, error) {
	var id string
	err := e.pool.QueryRow(ctx,
		`SELECT id FROM ledger_accounts WHERE type = 'user_wallet' AND user_id = $1 AND currency = $2`,
		userID, currency).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	err = e.pool.QueryRow(ctx,
		`INSERT INTO ledger_accounts (type, user_id, currency, cached_balance_minor)
		 VALUES ('user_wallet', $1, $2, 0) RETURNING id`,
		userID, currency).Scan(&id)
	if err == nil {
		return id, nil
	}
	// Lost the provisioning race: another posting created it first.
	if isUniqueViolation(err) {
		if e2 := e.pool.QueryRow(ctx,
			`SELECT id FROM ledger_accounts WHERE type = 'user_wallet' AND user_id = $1 AND currency = $2`,
			userID, currency).Scan(&id); e2 == nil {
			return id, nil
		}
	}
	return "", err
}

// idempotentReplay recognizes a unique-key collision on the idempotency key and
// returns the already-recorded posting.
func (e *Engine) idempotentReplay(ctx context.Context, err error, key string) (*Posting, bool) {
	if !isUniqueViolation(err) {
		return nil, false
	}
	var id string
	if e2 := e.pool.QueryRow(ctx,
		`SELECT id FROM journal_postings WHERE idempotency_key = $1`, key).Scan(&id); e2 != nil {
		return nil, false
	}
	return &Posting{ID: id, IdempotencyKey: key, Replayed: true}, true
}

// WalletBalanceMinor returns a user's cached wallet balance in minor units.
func (e *Engine) WalletBalanceMinor(ctx context.Context, userID, currency string) (int64, error) {
	var bal int64
	err := e.pool.QueryRow(ctx,
		`SELECT cached_balance_minor FROM ledger_accounts
		 WHERE type = 'user_wallet' AND user_id = $1 AND currency = $2`,
		userID, currency).Scan(&bal)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return bal, err
}

// DriftCount returns how many user wallets disagree with the journal. It must be
// zero; a nonzero value signals a ledger integrity bug.
func (e *Engine) DriftCount(ctx context.Context) (int, error) {
	var n int
	err := e.pool.QueryRow(ctx, `SELECT count(*) FROM wallet_journal_drift`).Scan(&n)
	return n, err
}

func orEmpty(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	return m
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func isRetryable(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == "40001" || pgErr.Code == "40P01" // serialization_failure, deadlock_detected
}

func backoff(ctx context.Context, attempt int) {
	base := time.Duration(attempt*attempt+1) * 3 * time.Millisecond
	jitter := time.Duration(rand.Int64N(int64(base) + 1))
	select {
	case <-ctx.Done():
	case <-time.After(base + jitter):
	}
}
