#!/usr/bin/env bash
set -euo pipefail

if ! command -v docker >/dev/null 2>&1 || ! docker info >/dev/null 2>&1; then
  echo "SKIP: Docker unavailable; skipping e2e"
  exit 0
fi

set +e
go run ./cmd/e2e
status=$?
if [[ $status -ne 0 ]]; then
  docker compose --env-file .env.example -f deploy/docker-compose.yml logs || true
fi
exit $status
