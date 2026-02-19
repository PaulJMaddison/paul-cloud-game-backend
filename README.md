# paul-cloud-game-backend

Go monorepo for backend services powering `paul-cloud-game-backend`.

## Services

Each service builds and runs as a separate binary under `/cmd`:

- `gateway`: edge HTTP entrypoint + WebSocket sessions
- `router`: internal request routing layer
- `login`: authentication/login flows
- `sessions`: session lifecycle management
- `matchmaking`: matchmaking orchestration

All services expose:

- `GET /healthz` -> `200 ok`
- `GET /readyz` -> `200 ready`
- `GET /metrics` -> Prometheus text format counters/gauges placeholder

## Observability

### Structured logging

All HTTP services use structured `zerolog` request logs with:

- `request_id` (from `X-Request-Id` or generated)
- `correlation_id` (when `X-Correlation-Id` header is present)
- `method`, `path`, `status`, `duration`

### Metrics endpoint placeholder

`/metrics` now emits minimal Prometheus-style metrics:

- `pcgb_http_requests_total`
- `pcgb_http_requests_all_total`
- `pcgb_process_uptime_seconds`

### OpenTelemetry scaffolding

OpenTelemetry bootstrap is wired in as optional scaffolding and is off by default.

- `ENABLE_OTEL=false` (default)
- `OTEL_EXPORTER_OTLP_ENDPOINT` (optional endpoint for future exporter wiring)

> Current behavior: when enabled, the services log that OTEL scaffolding mode is active.

## Shared packages

- `pkg/config`: environment variable parsing and defaults
- `pkg/logging`: zerolog logger configured with app/service/env metadata
- `pkg/httpserver`: server helpers, diagnostics endpoints, request logging middleware
- `pkg/observability`: optional OTEL initialization scaffold
- `pkg/storage`: `database/sql` setup with pgx + Redis client
- `pkg/bus`: NATS connection helper and event subjects

## Requirements

- Go 1.22+
- Docker + Docker Compose plugin
- `wscat` (or equivalent) for WebSocket demo

## Local development

1. Copy the sample env file and adjust values if needed:

   ```bash
   cp .env.example .env
   ```

2. Start local dependencies:

   ```bash
   make docker-up
   ```

3. Run DB migrations:

   ```bash
   make migrate-up
   ```

4. Run a service (example: gateway):

   ```bash
   make run-local
   ```

5. Verify health endpoint:

   ```bash
   curl -i http://localhost:8080/healthz
   ```

6. Run checks:

   ```bash
   make test
   make lint
   ```

7. Stop dependencies:

   ```bash
   make docker-down
   ```

## Local demo (end-to-end)

### Start infra + migrations + all services

```bash
scripts/local-demo.sh
```

This script performs:

- `make docker-up`
- `make migrate-up`
- starts `gateway`, `login`, `router`, `sessions`, `matchmaking`

### Sample flow: login -> connect WebSocket -> enqueue matchmaking -> receive match_found

Create two demo users and get tokens:

```bash
ALICE_LOGIN=$(curl -s -X POST http://localhost:8081/v1/login \
  -H 'Content-Type: application/json' \
  -H 'X-Correlation-Id: demo-alice-login' \
  -d '{"username":"alice","password":"demo-pass"}')

BOB_LOGIN=$(curl -s -X POST http://localhost:8081/v1/login \
  -H 'Content-Type: application/json' \
  -H 'X-Correlation-Id: demo-bob-login' \
  -d '{"username":"bob","password":"demo-pass"}')

ALICE_TOKEN=$(echo "$ALICE_LOGIN" | sed -E 's/.*"token":"([^"]+)".*/\1/')
BOB_TOKEN=$(echo "$BOB_LOGIN" | sed -E 's/.*"token":"([^"]+)".*/\1/')
```

Connect both users to gateway WS in separate terminals:

```bash
wscat -c "ws://localhost:8080/v1/ws?token=${ALICE_TOKEN}"
wscat -c "ws://localhost:8080/v1/ws?token=${BOB_TOKEN}"
```

Enqueue both users for matchmaking:

```bash
curl -i -X POST http://localhost:8084/v1/matchmaking/enqueue \
  -H "Authorization: Bearer ${ALICE_TOKEN}" \
  -H 'X-Correlation-Id: demo-mm-alice'

curl -i -X POST http://localhost:8084/v1/matchmaking/enqueue \
  -H "Authorization: Bearer ${BOB_TOKEN}" \
  -H 'X-Correlation-Id: demo-mm-bob'
```

Within a few seconds, both WS clients should receive a message like:

```json
{"type":"match_found","match_id":"<uuid>","other_user_id":"<user-id>"}
```

## Environment variables

### App services

- `APP_NAME` (default: `paul-cloud-game-backend`)
- `APP_ENV` (default: `development`)
- `PORT` (default: `8080`)
- `POSTGRES_URL` (default: `postgres://postgres:postgres@localhost:5432/paul_cloud_game?sslmode=disable`)
- `REDIS_ADDR` (default: `localhost:6379`)
- `NATS_URL` (default: `nats://localhost:4222`)
- `SHUTDOWN_TIMEOUT_SECONDS` (default: `10`)
- `ENABLE_OTEL` (default: `false`)
- `OTEL_EXPORTER_OTLP_ENDPOINT` (default: empty)

### Docker dependency services

- `POSTGRES_DB` (default: `paul_cloud_game`)
- `POSTGRES_USER` (default: `postgres`)
- `POSTGRES_PASSWORD` (default: `postgres`)

## Database migrations

Migrations live in `deploy/sql/migrations` using paired files:

- `<version>.up.sql`
- `<version>.down.sql`

Run migrations with:

```bash
make migrate-up
make migrate-down
```

## Make targets

- `make test`
- `make lint`
- `make build`
- `make run-local`
- `make docker-up`
- `make docker-down`
- `make migrate-up`
- `make migrate-down`
