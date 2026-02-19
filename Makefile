GO ?= go
SERVICES := gateway router login sessions matchmaking

.PHONY: test lint run-local local-demo docker-up docker-down migrate-up migrate-down build

test:
	$(GO) test ./...

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

local-demo:
	./scripts/local-demo.sh
