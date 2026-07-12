// Package testutil provides integration-test helpers. Tests that need a database
// require the TEST_DB_DSN environment variable pointing at a *direct* Postgres
// connection (not a transactional pooler); when it is unset the test is skipped.
//
// The schema is built by applying the real migration files, so integration tests
// exercise the production triggers -- including the append-only immutability
// trigger, which KiramoPay never installed in its test schema.
package testutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// schemaSeq gives every SetupDB call a distinct schema name so concurrently
// running test packages, which share one database, never clobber each other.
var schemaSeq atomic.Int64

// SetupDB returns a pool bound to a fresh, isolated schema with all migrations
// applied, or skips the test if TEST_DB_DSN is not set. Each call gets its own
// schema (dropped on cleanup), so `go test ./...` is safe under any parallelism.
func SetupDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("TEST_DB_DSN not set; skipping database integration test")
	}
	ctx := context.Background()

	schema := fmt.Sprintf("t_%d_%d", os.Getpid(), schemaSeq.Add(1))
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("parse dsn: %v", err)
	}
	// Every pooled connection resolves unqualified names in this test's schema
	// first, then public (for built-in extensions).
	cfg.ConnConfig.RuntimeParams["search_path"] = schema + ",public"

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() {
		dropCtx := context.Background()
		if _, err := pool.Exec(dropCtx, "DROP SCHEMA IF EXISTS "+schema+" CASCADE"); err != nil {
			t.Logf("cleanup drop schema %s: %v", schema, err)
		}
		pool.Close()
	})

	if _, err := pool.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	// pgcrypto is a database-global extension shared by every test schema.
	// `CREATE EXTENSION IF NOT EXISTS` is not atomic, so concurrent test packages
	// can collide on pg_extension; serialize its creation with an advisory lock
	// so the per-schema migrations below only ever see it already present.
	if _, err := pool.Exec(ctx, `BEGIN;
		SELECT pg_advisory_xact_lock(987654321);
		CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;
		COMMIT;`); err != nil {
		t.Fatalf("ensure pgcrypto: %v", err)
	}
	for _, file := range migrationFiles(t) {
		sqlBytes, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read migration %s: %v", file, err)
		}
		if _, err := pool.Exec(ctx, string(sqlBytes)); err != nil {
			t.Fatalf("apply migration %s: %v", filepath.Base(file), err)
		}
	}
	return pool
}

// CreateUser inserts a minimal user row and returns its id. PII columns are
// filled with placeholder bytes; integration tests here exercise the ledger, not
// PII encryption.
func CreateUser(t *testing.T, pool *pgxpool.Pool, tag string) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO users (phone_hmac, phone_cipher, password_hash)
		 VALUES ($1, $2, 'x') RETURNING id`,
		[]byte("hmac-"+tag), []byte("cipher-"+tag)).Scan(&id)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return id
}

func migrationFiles(t *testing.T) []string {
	t.Helper()
	dir := findMigrationsDir(t)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read migrations dir: %v", err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".sql" {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(files)
	return files
}

// findMigrationsDir walks up from the test's working directory to locate the
// backend/migrations folder, so helpers work from any package.
func findMigrationsDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(dir, "migrations")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("could not locate migrations directory")
	return ""
}
