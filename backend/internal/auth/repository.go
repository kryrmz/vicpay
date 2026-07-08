package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository is the Postgres persistence for auth.
type Repository struct{ pool *pgxpool.Pool }

// NewRepository builds a Repository.
func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// CreateUser inserts a new user and returns its id.
func (r *Repository) CreateUser(ctx context.Context, phoneHMAC, phoneCipher []byte, passwordHash string) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (phone_hmac, phone_cipher, password_hash)
		 VALUES ($1, $2, $3) RETURNING id`,
		phoneHMAC, phoneCipher, passwordHash).Scan(&id)
	return id, err
}

// FindByPhoneIndex looks a user up by its blind-index HMAC. Returns nil if absent.
func (r *Repository) FindByPhoneIndex(ctx context.Context, phoneHMAC []byte) (*userRow, error) {
	return r.scanUser(ctx,
		`SELECT id, password_hash, phone_cipher, phone_verified, kyc_level, status
		 FROM users WHERE phone_hmac = $1`, phoneHMAC)
}

// GetUser fetches a user by id.
func (r *Repository) GetUser(ctx context.Context, id string) (*userRow, error) {
	return r.scanUser(ctx,
		`SELECT id, password_hash, phone_cipher, phone_verified, kyc_level, status
		 FROM users WHERE id = $1`, id)
}

func (r *Repository) scanUser(ctx context.Context, query string, arg any) (*userRow, error) {
	var u userRow
	err := r.pool.QueryRow(ctx, query, arg).Scan(
		&u.id, &u.passwordHash, &u.phoneCipher, &u.phoneVerified, &u.kycLevel, &u.status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// SetPhoneVerified marks a user's phone as verified.
func (r *Repository) SetPhoneVerified(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET phone_verified = true, updated_at = now() WHERE id = $1`, id)
	return err
}

// InsertRefresh records a new refresh token in its rotation family.
func (r *Repository) InsertRefresh(ctx context.Context, row refreshRow) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO refresh_tokens (jti, user_id, family_id, parent_jti, token_hash, family_origin, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		row.jti, row.userID, row.familyID, row.parentJTI, row.tokenHash, row.familyOrigin, row.expiresAt)
	return err
}

// GetRefresh fetches a refresh token row by jti. Returns nil if absent.
func (r *Repository) GetRefresh(ctx context.Context, jti string) (*refreshRow, error) {
	var row refreshRow
	err := r.pool.QueryRow(ctx,
		`SELECT jti, user_id, family_id, parent_jti, issued_at, family_origin, expires_at, used_at, revoked_at
		 FROM refresh_tokens WHERE jti = $1`, jti).Scan(
		&row.jti, &row.userID, &row.familyID, &row.parentJTI, &row.issuedAt,
		&row.familyOrigin, &row.expiresAt, &row.usedAt, &row.revokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// MarkRefreshUsed stamps used_at, making a second use detectable as reuse.
func (r *Repository) MarkRefreshUsed(ctx context.Context, jti string, at time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE refresh_tokens SET used_at = $2 WHERE jti = $1 AND used_at IS NULL`, jti, at)
	return err
}

// RevokeFamily revokes every unrevoked token in a rotation family.
func (r *Repository) RevokeFamily(ctx context.Context, familyID string, at time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = $2 WHERE family_id = $1 AND revoked_at IS NULL`, familyID, at)
	return err
}
