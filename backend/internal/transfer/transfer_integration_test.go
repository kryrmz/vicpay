package transfer_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vicpay/backend/internal/ledger"
	"github.com/vicpay/backend/internal/pii"
	"github.com/vicpay/backend/internal/testutil"
	"github.com/vicpay/backend/internal/transfer"
)

func newService(t *testing.T) (*transfer.Service, *pgxpool.Pool, *pii.Cipher) {
	t.Helper()
	pool := testutil.SetupDB(t)
	cipher, err := pii.NewCipher([]byte("a-32-byte-master-key-for-testing!"))
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	svc := transfer.NewService(pool, ledger.New(pool), cipher)
	return svc, pool, cipher
}

// createUser inserts a user whose phone_hmac matches the cipher's blind index so
// the transfer service can resolve them by phone.
func createUser(t *testing.T, pool *pgxpool.Pool, cipher *pii.Cipher, phone string, verified bool) string {
	t.Helper()
	blob, err := cipher.Encrypt(phone)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO users (phone_hmac, phone_cipher, password_hash, phone_verified)
		 VALUES ($1, $2, 'x', $3) RETURNING id`,
		cipher.BlindIndex(phone), blob, verified).Scan(&id); err != nil {
		t.Fatalf("create user: %v", err)
	}
	return id
}

func TestTopUpAndTransfer(t *testing.T) {
	svc, pool, cipher := newService(t)
	ctx := context.Background()
	alice := createUser(t, pool, cipher, "+50688881111", true)
	_ = createUser(t, pool, cipher, "+50688882222", true) // bob

	if _, err := svc.TopUp(ctx, alice, 10000, "USD", "topup-1"); err != nil {
		t.Fatalf("topup: %v", err)
	}
	res, err := svc.Transfer(ctx, alice, "+50688882222", 2500, "USD", "xfer-1")
	if err != nil {
		t.Fatalf("transfer: %v", err)
	}
	if res.NewBalanceMinor != 7500 {
		t.Fatalf("sender balance = %d, want 7500", res.NewBalanceMinor)
	}

	// Idempotent replay must not double-debit.
	res2, err := svc.Transfer(ctx, alice, "+50688882222", 2500, "USD", "xfer-1")
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if res2.NewBalanceMinor != 7500 {
		t.Fatalf("after replay balance = %d, want 7500 (no double debit)", res2.NewBalanceMinor)
	}
}

func TestTransferInsufficientFunds(t *testing.T) {
	svc, pool, cipher := newService(t)
	ctx := context.Background()
	alice := createUser(t, pool, cipher, "+50688881111", true)
	_ = createUser(t, pool, cipher, "+50688882222", true)

	if _, err := svc.Transfer(ctx, alice, "+50688882222", 100, "USD", "x"); !errors.Is(err, transfer.ErrInsufficientFunds) {
		t.Fatalf("expected ErrInsufficientFunds, got %v", err)
	}
}

func TestTransferSelfAndUnknownAndUnverified(t *testing.T) {
	svc, pool, cipher := newService(t)
	ctx := context.Background()
	alice := createUser(t, pool, cipher, "+50688881111", true)
	_ = createUser(t, pool, cipher, "+50688883333", false) // carol, unverified
	if _, err := svc.TopUp(ctx, alice, 10000, "USD", "t"); err != nil {
		t.Fatalf("topup: %v", err)
	}

	if _, err := svc.Transfer(ctx, alice, "+50688881111", 100, "USD", "s"); !errors.Is(err, transfer.ErrSelfTransfer) {
		t.Fatalf("expected ErrSelfTransfer, got %v", err)
	}
	if _, err := svc.Transfer(ctx, alice, "+50699990000", 100, "USD", "u"); !errors.Is(err, transfer.ErrRecipientNotFound) {
		t.Fatalf("expected ErrRecipientNotFound, got %v", err)
	}
	if _, err := svc.Transfer(ctx, alice, "+50688883333", 100, "USD", "v"); !errors.Is(err, transfer.ErrRecipientUnverified) {
		t.Fatalf("expected ErrRecipientUnverified, got %v", err)
	}
}

func TestTransferToUserByID(t *testing.T) {
	svc, pool, cipher := newService(t)
	ctx := context.Background()
	alice := createUser(t, pool, cipher, "+50688881111", true)
	bob := createUser(t, pool, cipher, "+50688882222", true)

	if _, err := svc.TopUp(ctx, alice, 10000, "USD", "t"); err != nil {
		t.Fatalf("topup: %v", err)
	}
	res, err := svc.TransferToUser(ctx, alice, bob, 2500, "USD", "qr-1")
	if err != nil {
		t.Fatalf("transfer to user: %v", err)
	}
	if res.NewBalanceMinor != 7500 {
		t.Fatalf("balance = %d, want 7500", res.NewBalanceMinor)
	}
	// Unknown and self recipients are rejected.
	if _, err := svc.TransferToUser(ctx, alice, "00000000-0000-0000-0000-000000000000", 100, "USD", "u"); !errors.Is(err, transfer.ErrRecipientNotFound) {
		t.Fatalf("expected ErrRecipientNotFound, got %v", err)
	}
	if _, err := svc.TransferToUser(ctx, alice, alice, 100, "USD", "s"); !errors.Is(err, transfer.ErrSelfTransfer) {
		t.Fatalf("expected ErrSelfTransfer, got %v", err)
	}
}

func TestTransferKYCLimit(t *testing.T) {
	svc, pool, cipher := newService(t)
	ctx := context.Background()
	alice := createUser(t, pool, cipher, "+50688881111", true) // level 0: daily 20000
	_ = createUser(t, pool, cipher, "+50688882222", true)

	if _, err := svc.TopUp(ctx, alice, 100000, "USD", "t"); err != nil {
		t.Fatalf("topup: %v", err)
	}
	// 25000 exceeds the L0 daily limit of 20000, even though funds are available.
	if _, err := svc.Transfer(ctx, alice, "+50688882222", 25000, "USD", "big"); !errors.Is(err, transfer.ErrLimitExceeded) {
		t.Fatalf("expected ErrLimitExceeded, got %v", err)
	}
	// A transfer within the limit succeeds.
	if _, err := svc.Transfer(ctx, alice, "+50688882222", 15000, "USD", "ok"); err != nil {
		t.Fatalf("within-limit transfer: %v", err)
	}
	// A second transfer that would push the daily total over 20000 is rejected.
	if _, err := svc.Transfer(ctx, alice, "+50688882222", 6000, "USD", "over"); !errors.Is(err, transfer.ErrLimitExceeded) {
		t.Fatalf("expected daily cumulative limit, got %v", err)
	}
}
