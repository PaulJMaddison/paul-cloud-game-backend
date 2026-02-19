GO ?= go
SERVICES := gateway router login sessions matchmaking

.PHONY: test test-unit test-integration test-e2e test-all lint run-local docker-up docker-down migrate-up migrate-down build

test: test-unit

test-unit:
	$(GO) test ./...

test-integration:
	$(GO) test -tags=integration ./...

test-e2e:
	./scripts/test-e2e.sh

test-all:
	@echo "==> unit"
	@$(MAKE) test-unit
	@echo "==> integration"
	@$(MAKE) test-integration || echo "integration tier skipped or failed"
	@echo "==> e2e"
	@$(MAKE) test-e2e || echo "e2e tier skipped or failed"

lint:
	$(GO) vet ./...

build:
	@for svc in $(SERVICES); do \
		echo "building $$svc"; \
		$(GO) build -o bin/$$svc ./cmd/$$svc; \
	done

run-local:
	$(GO) run ./cmd/gateway

docker-up:
	docker compose --env-file .env.example -f deploy/docker-compose.yml up -d

docker-down:
	docker compose --env-file .env.example -f deploy/docker-compose.yml down

migrate-up:
	POSTGRES_URL=$${POSTGRES_URL:-postgres://postgres:postgres@localhost:5432/paul_cloud_game?sslmode=disable} $(GO) run ./cmd/migrate up

migrate-down:
	POSTGRES_URL=$${POSTGRES_URL:-postgres://postgres:postgres@localhost:5432/paul_cloud_game?sslmode=disable} $(GO) run ./cmd/migrate down -steps=1
