GO ?= go
SERVICES := gateway router login sessions matchmaking migrate e2e

.PHONY: test test-unit test-integration test-e2e test-all fmt lint run-local docker-up docker-down migrate-up migrate-down build

fmt:
	@echo "==> formatting Go files"
	@$(GO) fmt ./...

test: test-unit

test-unit:
	@echo "==> unit tests"
	@$(GO) test ./...

test-integration:
	@echo "==> integration tests"
	@./scripts/test.sh

test-e2e:
	@echo "==> e2e tests"
	@./scripts/test-e2e.sh

test-all:
	@echo "==> unit"
	@$(MAKE) test-unit
	@echo "==> integration"
	@$(GO) test -tags=integration ./...
	@echo "==> e2e"
	@$(MAKE) test-e2e

lint:
	@echo "==> golangci-lint"
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "golangci-lint is required (https://golangci-lint.run/usage/install/)" >&2; \
		exit 1; \
	fi
	@golangci-lint run ./...

build:
	@echo "==> go build ./..."
	@$(GO) build ./...
	@for svc in $(SERVICES); do \
		echo "building cmd/$$svc"; \
		$(GO) build -o bin/$$svc ./cmd/$$svc; \
	done

run-local:
	@$(GO) run ./cmd/gateway

docker-up:
	@./scripts/docker-up.sh

docker-down:
	@./scripts/docker-down.sh

migrate-up:
	@POSTGRES_URL=$${POSTGRES_URL:-postgres://postgres:postgres@localhost:5432/paul_cloud_game?sslmode=disable} $(GO) run ./cmd/migrate up

migrate-down:
	@POSTGRES_URL=$${POSTGRES_URL:-postgres://postgres:postgres@localhost:5432/paul_cloud_game?sslmode=disable} $(GO) run ./cmd/migrate down -steps=1

local-demo:
	@./scripts/local-demo.sh
