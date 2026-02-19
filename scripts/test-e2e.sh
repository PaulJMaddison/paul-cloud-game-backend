#!/usr/bin/env bash
set -euo pipefail

log() { echo "[e2e] $*"; }
warn() { echo "[e2e][warn] $*"; }

if ! command -v go >/dev/null 2>&1; then
  echo "[e2e][error] Go toolchain is required but missing from PATH" >&2
  exit 1
fi

if ! command -v docker >/dev/null 2>&1; then
  warn "SKIP: docker CLI not found; skipping e2e"
  exit 0
fi
if ! docker info >/dev/null 2>&1; then
  warn "SKIP: docker daemon unavailable; skipping e2e"
  exit 0
fi

log "running e2e suite"
set +e
go run ./cmd/e2e
status=$?
set -e
if [[ $status -ne 0 ]]; then
  warn "e2e failed; collecting docker compose logs"
  docker compose --env-file .env.example -f deploy/docker-compose.yml logs || warn "unable to collect docker compose logs"
fi
exit $status
