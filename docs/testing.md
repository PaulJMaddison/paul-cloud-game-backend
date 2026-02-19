# Testing Guide

## Test tiers

- Unit: `go test ./...`
- Integration: `go test -tags=integration ./...`
- E2E: `./scripts/test-e2e.sh`

## Full quality pass (recommended)

Run these in order:

```bash
gofmt -w $(git ls-files '*.go')
go test ./...
go test -tags=integration ./...
golangci-lint run ./...
go build ./...
for d in cmd/*; do [ -d "$d" ] && go build "./$d"; done
go mod tidy
```

## Environment variables

- `TEST_TIMEOUT_SECONDS` (default `10`): common timeout used by tests/harnesses.
- `LOGIN_JWT_SECRET` (default `local-dev-secret`): JWT secret for auth tests and local services.
- `MATCHMAKING_JWT_SECRET` (default falls back to `LOGIN_JWT_SECRET`).
- `ADMIN_TOKEN` (service default: unset; tests set explicitly when needed).

## Dependency availability and skips

Integration and E2E tests are Docker-dependent.

- `./scripts/test.sh` runs unit tests first, then integration tests only when Docker is available.
- `./scripts/test-e2e.sh` runs E2E only when Docker is available.
- When Docker is unavailable (sandboxed CI/workstation), those scripts **skip with explicit messages** and exit successfully.

## CI recommendations

1. Run `./scripts/test.sh` on every PR.
2. Run `golangci-lint run ./...` on every PR.
3. Run `make build` on every PR.
4. Keep a Docker-enabled CI lane for full integration and e2e reliability checks.
