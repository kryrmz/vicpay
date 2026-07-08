.PHONY: build run test test-integration tidy lint gosec vulncheck verify docker-up docker-down

# Local developer commands. Security gates (gosec, govulncheck) also run in CI
# with pinned tool versions; see .github/workflows/ci.yml.

build:
	go build ./...

run:
	go run ./cmd/api

# Unit tests only (no database required).
test:
	go test ./...

# Integration tests: point TEST_DB_DSN at a DIRECT Postgres connection.
test-integration:
	go test ./... -count=1

tidy:
	go mod tidy

lint:
	golangci-lint run ./...

gosec:
	gosec -severity=medium -exclude=G404 -exclude-dir=internal/testutil ./...

vulncheck:
	govulncheck ./...

# Full local gate mirroring CI (unit tests; integration needs TEST_DB_DSN).
verify: build lint gosec test

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down -v
