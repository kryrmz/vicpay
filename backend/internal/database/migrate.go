package database

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// advisoryLockKey gates concurrent migration runners to a single leader.
const advisoryLockKey int64 = 0x766963706179 // "vicpay"

// Migrate applies every *.sql file in dir, in lexical order, exactly once. It
// tracks applied files with their checksum in schema_migrations and refuses to
// proceed if a previously applied file has since been edited. It must run
// against a DIRECT connection (session advisory lock), never through PgBouncer
// in transaction mode.
func Migrate(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("migrate: acquire: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, advisoryLockKey); err != nil {
		return fmt.Errorf("migrate: lock: %w", err)
	}
	defer func() { _, _ = conn.Exec(ctx, `SELECT pg_advisory_unlock($1)`, advisoryLockKey) }()

	if _, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename   text PRIMARY KEY,
			checksum   text NOT NULL,
			applied_at timestamptz NOT NULL DEFAULT now()
		)`); err != nil {
		return fmt.Errorf("migrate: ensure table: %w", err)
	}

	applied, err := loadApplied(ctx, conn)
	if err != nil {
		return err
	}

	files, err := sqlFiles(dir)
	if err != nil {
		return err
	}
	for _, file := range files {
		name := filepath.Base(file)
		content, err := readMigration(file)
		if err != nil {
			return err
		}
		sum := checksum(content)
		if prev, ok := applied[name]; ok {
			if prev != sum {
				return fmt.Errorf("migrate: %s was modified after being applied", name)
			}
			continue
		}
		if err := applyOne(ctx, conn, name, content, sum); err != nil {
			return err
		}
	}
	return nil
}

func applyOne(ctx context.Context, conn *pgxpool.Conn, name, content, sum string) error {
	// Statements needing to run outside a transaction (e.g. CREATE INDEX
	// CONCURRENTLY) are not supported by this minimal runner yet.
	if strings.Contains(content, "-- migrate:no-transaction") {
		return fmt.Errorf("migrate: %s requests no-transaction, unsupported", name)
	}
	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("migrate: begin %s: %w", name, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, content); err != nil {
		return fmt.Errorf("migrate: apply %s: %w", name, err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO schema_migrations (filename, checksum) VALUES ($1, $2)`, name, sum); err != nil {
		return fmt.Errorf("migrate: record %s: %w", name, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("migrate: commit %s: %w", name, err)
	}
	return nil
}

func loadApplied(ctx context.Context, conn *pgxpool.Conn) (map[string]string, error) {
	rows, err := conn.Query(ctx, `SELECT filename, checksum FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("migrate: load applied: %w", err)
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var name, sum string
		if err := rows.Scan(&name, &sum); err != nil {
			return nil, err
		}
		out[name] = sum
	}
	return out, rows.Err()
}

func sqlFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("migrate: read dir: %w", err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".sql" {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(files)
	return files, nil
}

func readMigration(path string) (string, error) {
	b, err := os.ReadFile(path) // #nosec G304 -- path comes from the app's own migrations dir
	if err != nil {
		return "", fmt.Errorf("migrate: read %s: %w", path, err)
	}
	return string(b), nil
}

func checksum(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}
