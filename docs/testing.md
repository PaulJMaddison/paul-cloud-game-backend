# Testing Guide

## Test tiers

- Unit: `go test ./...`
- Integration: `go test -tags=integration ./...`
- E2E: `make test-e2e`

## Environment variables

- `TEST_TIMEOUT_SECONDS` (default `10`): common timeout used by tests/harnesses.
- `LOGIN_JWT_SECRET` (default `local-dev-secret`): JWT secret for auth tests and local services.
- `MATCHMAKING_JWT_SECRET` (default falls back to `LOGIN_JWT_SECRET`).
- `ADMIN_TOKEN` (service default: unset; tests set explicitly when needed).

## Dependency availability and skips

Integration/E2E tests detect Docker using `docker info`.

When Docker is unavailable (sandboxed CI/workstation), integration and E2E suites are **skipped with explicit messages** instead of failing.

## CI recommendations

1. Run `make test-unit` on every PR.
2. Run `make test-integration` in Docker-enabled CI workers.
3. Run `make test-e2e` in Docker-enabled CI workers.
4. Use `make test-all` to execute tiers in order with skip-aware behavior.
