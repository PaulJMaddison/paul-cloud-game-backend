# paul-cloud-game-backend

Go monorepo for backend services powering `paul-cloud-game-backend`.

## Services

Each service builds and runs as a separate binary under `/cmd`:

- `gateway`: edge HTTP entrypoint
- `router`: internal request routing layer
- `login`: authentication/login flows
- `sessions`: session lifecycle management
- `matchmaking`: matchmaking orchestration

All services currently use shared bootstrap logic and expose:

- `GET /healthz` -> `200 ok`
- `GET /readyz` -> `200 ready`
- `GET /metrics` -> `501 metrics placeholder`

## Shared packages

- `pkg/config`: environment variable parsing and defaults
- `pkg/logging`: zerolog logger configured with
  - `app="paul-cloud-game-backend"`
  - `service="<name>"`
- `pkg/httpserver`: server helpers and health/ready/metrics handlers
- `pkg/storage`: `database/sql` setup with pgx + Redis client
- `pkg/bus`: NATS connection helper and event subjects

## Requirements

- Go 1.22+
- Docker (for local Postgres/Redis/NATS)

## Local development

1. Start local dependencies:

   ```bash
   make docker-up
   ```

2. Run a service (example: gateway):

   ```bash
   make run-local
   ```

3. Verify health endpoint:

   ```bash
   curl -i http://localhost:8080/healthz
   ```

4. Run checks:

   ```bash
   make test
   make lint
   ```

5. Stop dependencies:

   ```bash
   make docker-down
   ```

## Environment variables

Defaults are suitable for local development:

- `APP_NAME=paul-cloud-game-backend`
- `APP_ENV=development`
- `PORT=8080`
- `POSTGRES_URL=postgres://postgres:postgres@localhost:5432/paul_cloud_game?sslmode=disable`
- `REDIS_ADDR=localhost:6379`
- `NATS_URL=nats://localhost:4222`
- `SHUTDOWN_TIMEOUT_SECONDS=10`

## Make targets

- `make test`
- `make lint`
- `make run-local`
- `make docker-up`
- `make docker-down`
- `make migrate`
