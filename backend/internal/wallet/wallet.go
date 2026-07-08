// Package wallet exposes read models over the ledger: a user's multi-currency
// balances and their recent transaction history. All movement of value happens
// through the ledger engine; this package only reads.
package wallet

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Balance is a user's balance in one currency, in minor units.
type Balance struct {
	Currency     string `json:"currency"`
	BalanceMinor int64  `json:"balanceMinor"`
}

// Transaction is a ledger movement as seen from the user's side.
type Transaction struct {
	ID          string    `json:"id"`
	Kind        string    `json:"kind"` // "in" (credit) or "out" (debit)
	Description string    `json:"counterparty"`
	AmountMinor int64     `json:"amountMinor"`
	Currency    string    `json:"currency"`
	CreatedAt   time.Time `json:"createdAt"`
}

// Service reads wallet data.
type Service struct{ pool *pgxpool.Pool }

// NewService builds a Service.
func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

// Balances returns every wallet balance for a user.
func (s *Service) Balances(ctx context.Context, userID string) ([]Balance, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT currency, cached_balance_minor
		 FROM ledger_accounts
		 WHERE type = 'user_wallet' AND user_id = $1
		 ORDER BY currency`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Balance{}
	for rows.Next() {
		var b Balance
		if err := rows.Scan(&b.Currency, &b.BalanceMinor); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// Transactions returns a user's most recent ledger entries.
func (s *Service) Transactions(ctx context.Context, userID string, limit int) ([]Transaction, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx,
		`SELECT p.id, e.direction, e.amount_minor, e.currency, p.description, p.created_at
		 FROM journal_entries e
		 JOIN journal_postings p ON p.id = e.posting_id
		 JOIN ledger_accounts a ON a.id = e.account_id
		 WHERE a.type = 'user_wallet' AND a.user_id = $1
		 ORDER BY p.created_at DESC
		 LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Transaction{}
	for rows.Next() {
		var t Transaction
		var direction string
		if err := rows.Scan(&t.ID, &direction, &t.AmountMinor, &t.Currency, &t.Description, &t.CreatedAt); err != nil {
			return nil, err
		}
		if direction == "credit" {
			t.Kind = "in"
		} else {
			t.Kind = "out"
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
