# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run Commands

```bash
# Build
go build -o server .

# Run all tests (integration tests skip automatically without DYNAMODB_ENDPOINT)
go test ./...

# Run a single test
go test -run TestGetAll_Empty -v ./...

# Run integration tests (requires DynamoDB Local on port 8000)
DYNAMODB_ENDPOINT=http://localhost:8000 go test -run Integration -v ./...

# Vet
go vet ./...

# Local dev with Docker (starts DynamoDB Local + creates table + runs app)
docker compose up

# Run without Docker (source .env first for local defaults)
set -a && source .env && set +a && go run .
```

## Architecture

Single `package main` Go API for user preference CRUD, backed by DynamoDB. Uses only stdlib for HTTP routing (`net/http` with Go 1.22+ method patterns), logging (`log/slog`), and JSON. Two external dependencies: AWS SDK v2 and `golang-jwt/jwt/v5`.

**Request flow:** Recovery → CORS → RequestLogging → JWTAuth → ServeMux → PreferencesHandler → Store (DynamoDB)

**Key types:**
- `Store` interface (store.go) — 6 methods for preference CRUD. `DynamoStore` is the production implementation; tests use `mockStore` in handler_test.go.
- `PreferencesHandler` (handler.go) — holds `Store` + `*slog.Logger`, methods are HTTP handlers. Each handler calls `authorize()` to verify JWT subject matches the `{userId}` path param.
- `Claims` / `ClaimsFromContext()` (middleware.go) — JWT subject stored in request context by auth middleware, extracted by handlers.

**DynamoDB schema:** Single table, partition key `PK` = `USER#{userId}`, no sort key. Preferences stored as a DynamoDB Map attribute. Partial updates use `UpdateItem` with `SET preferences.#key = :val` expressions.

**Config:** All env vars, loaded in `LoadConfig()`. App refuses to start without `JWT_SECRET`. Set `DYNAMODB_ENDPOINT` for local dev (empty = real AWS).

## Testing

Handler tests use a `mockStore` (in-memory map) and inject JWT claims via `withClaims()` helper. Integration tests in dynamo_store_test.go auto-skip when `DYNAMODB_ENDPOINT` is unset. All middleware tests create real JWT tokens with `makeToken()`/`makeTokenWithExp()` helpers.
