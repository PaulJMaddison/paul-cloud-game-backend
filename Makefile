GO ?= go
SERVICES := gateway router login sessions matchmaking

.PHONY: test lint run-local docker-up docker-down migrate build

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
	docker compose -f deploy/docker-compose.yml up -d

docker-down:
	docker compose -f deploy/docker-compose.yml down

migrate:
	@echo "No migrations yet. Add migration tooling (goose/atlas) here."
