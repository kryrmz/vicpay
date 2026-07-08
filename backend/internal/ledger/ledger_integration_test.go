package ledger_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/vicpay/backend/internal/ledger"
	"github.com/vicpay/backend/internal/testutil"
)

// fund credits a user's wallet from the external system account.
func fund(t *testing.T, e *ledger.Engine, userID, currency string, amount int64, key string) {
	t.Helper()
	_, err := e.Post(context.Background(), ledger.PostingInput{
		IdempotencyKey: key,
		Description:    "funding",
		Entries: []ledger.EntryInput{
			{Account: ledger.System("SYSTEM:EXTERNAL:" + currency), Direction: ledger.Debit, AmountMinor: amount, Currency: currency},
			{Account: ledger.Wallet(userID), Direction: ledger.Credit, AmountMinor: amount, Currency: currency},
		},
	})
	if err != nil {
		t.Fatalf("fund: %v", err)
	}
}

func TestPostBasicTransfer(t *testing.T) {
	pool := testutil.SetupDB(t)
	e := ledger.New(pool)
	ctx := context.Background()

	alice := testutil.CreateUser(t, pool, "alice")
	bob := testutil.CreateUser(t, pool, "bob")

	fund(t, e, alice, "USD", 10000, "fund-alice")

	_, err := e.Post(ctx, ledger.PostingInput{
		IdempotencyKey: "transfer-1",
		Description:    "alice pays bob",
		Metadata:       map[string]any{"kind": "p2p"},
		Entries: []ledger.EntryInput{
			{Account: ledger.Wallet(alice), Direction: ledger.Debit, AmountMinor: 2500, Currency: "USD"},
			{Account: ledger.Wallet(bob), Direction: ledger.Credit, AmountMinor: 2500, Currency: "USD"},
		},
	})
	if err != nil {
		t.Fatalf("transfer: %v", err)
	}

	if bal, _ := e.WalletBalanceMinor(ctx, alice, "USD"); bal != 7500 {
		t.Fatalf("alice balance = %d, want 7500", bal)
	}
	if bal, _ := e.WalletBalanceMinor(ctx, bob, "USD"); bal != 2500 {
		t.Fatalf("bob balance = %d, want 2500", bal)
	}
	assertNoDrift(t, e)
}

func TestPostIdempotencyKey(t *testing.T) {
	pool := testutil.SetupDB(t)
	e := ledger.New(pool)
	ctx := context.Background()
	alice := testutil.CreateUser(t, pool, "alice")

	fund(t, e, alice, "USD", 5000, "fund-once")
	// Replaying the same funding key must not double-credit.
	fund(t, e, alice, "USD", 5000, "fund-once")

	p, err := e.Post(ctx, ledger.PostingInput{
		IdempotencyKey: "fund-once",
		Description:    "funding",
		Entries: []ledger.EntryInput{
			{Account: ledger.System("SYSTEM:EXTERNAL:USD"), Direction: ledger.Debit, AmountMinor: 5000, Currency: "USD"},
			{Account: ledger.Wallet(alice), Direction: ledger.Credit, AmountMinor: 5000, Currency: "USD"},
		},
	})
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if !p.Replayed {
		t.Fatal("expected Replayed=true on idempotent repost")
	}
	if bal, _ := e.WalletBalanceMinor(ctx, alice, "USD"); bal != 5000 {
		t.Fatalf("balance = %d, want 5000 (no double credit)", bal)
	}
}

// TestImmutableEntries verifies the append-only triggers for real -- KiramoPay's
// equivalent test was skipped because its test schema lacked the trigger.
func TestImmutableEntries(t *testing.T) {
	pool := testutil.SetupDB(t)
	e := ledger.New(pool)
	ctx := context.Background()
	alice := testutil.CreateUser(t, pool, "alice")
	fund(t, e, alice, "USD", 1000, "fund-alice")

	if _, err := pool.Exec(ctx, `UPDATE journal_entries SET amount_minor = 1`); err == nil {
		t.Fatal("UPDATE on journal_entries must be rejected by the immutability trigger")
	}
	if _, err := pool.Exec(ctx, `DELETE FROM journal_entries`); err == nil {
		t.Fatal("DELETE on journal_entries must be rejected")
	}
	if _, err := pool.Exec(ctx, `UPDATE journal_postings SET description = 'x'`); err == nil {
		t.Fatal("UPDATE on journal_postings must be rejected")
	}
	if _, err := pool.Exec(ctx, `DELETE FROM journal_postings`); err == nil {
		t.Fatal("DELETE on journal_postings must be rejected")
	}
}

// TestUnbalancedRejectedByDB proves the deferred balance trigger fires at COMMIT
// even when the application layer is bypassed with raw SQL.
func TestUnbalancedRejectedByDB(t *testing.T) {
	pool := testutil.SetupDB(t)
	ctx := context.Background()
	alice := testutil.CreateUser(t, pool, "alice")

	var walletID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO ledger_accounts (type, user_id, currency, cached_balance_minor)
		 VALUES ('user_wallet', $1, 'USD', 0) RETURNING id`, alice).Scan(&walletID); err != nil {
		t.Fatalf("create wallet: %v", err)
	}
	var extID string
	_ = pool.QueryRow(ctx, `SELECT id FROM ledger_accounts WHERE code = 'SYSTEM:EXTERNAL:USD'`).Scan(&extID)

	tx, _ := pool.Begin(ctx)
	var postingID string
	_ = tx.QueryRow(ctx,
		`INSERT INTO journal_postings (idempotency_key, description) VALUES ('bad', 'x') RETURNING id`).Scan(&postingID)
	// Debit 1000 but credit only 999 -> unbalanced.
	_, _ = tx.Exec(ctx, `INSERT INTO journal_entries (posting_id, account_id, direction, amount_minor, currency)
		VALUES ($1, $2, 'debit', 1000, 'USD')`, postingID, extID)
	_, _ = tx.Exec(ctx, `INSERT INTO journal_entries (posting_id, account_id, direction, amount_minor, currency)
		VALUES ($1, $2, 'credit', 999, 'USD')`, postingID, walletID)
	if err := tx.Commit(ctx); err == nil {
		t.Fatal("COMMIT of an unbalanced posting must be rejected by the deferred balance trigger")
	}
}

// TestConcurrentTransfers runs many concurrent transfers between two wallets and
// asserts exact net movement with zero cache/journal drift.
func TestConcurrentTransfers(t *testing.T) {
	pool := testutil.SetupDB(t)
	e := ledger.New(pool)
	ctx := context.Background()
	alice := testutil.CreateUser(t, pool, "alice")
	bob := testutil.CreateUser(t, pool, "bob")
	fund(t, e, alice, "USD", 100000, "fund-alice")

	const n = 100
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := e.Post(ctx, ledger.PostingInput{
				IdempotencyKey: fmt.Sprintf("concurrent-%d", i),
				Description:    "concurrent transfer",
				Entries: []ledger.EntryInput{
					{Account: ledger.Wallet(alice), Direction: ledger.Debit, AmountMinor: 100, Currency: "USD"},
					{Account: ledger.Wallet(bob), Direction: ledger.Credit, AmountMinor: 100, Currency: "USD"},
				},
			})
			errs <- err
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent post: %v", err)
		}
	}

	if bal, _ := e.WalletBalanceMinor(ctx, alice, "USD"); bal != 100000-n*100 {
		t.Fatalf("alice balance = %d, want %d", bal, 100000-n*100)
	}
	if bal, _ := e.WalletBalanceMinor(ctx, bob, "USD"); bal != n*100 {
		t.Fatalf("bob balance = %d, want %d", bal, n*100)
	}
	assertNoDrift(t, e)
}

func assertNoDrift(t *testing.T, e *ledger.Engine) {
	t.Helper()
	drift, err := e.DriftCount(context.Background())
	if err != nil {
		t.Fatalf("drift count: %v", err)
	}
	if drift != 0 {
		t.Fatalf("wallet_journal_drift has %d rows, want 0 (cache disagrees with journal)", drift)
	}
}
