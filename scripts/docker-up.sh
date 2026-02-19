#!/usr/bin/env bash
set -euo pipefail

log() { echo "[docker-up] $*"; }

if ! command -v docker >/dev/null 2>&1; then
  log "SKIP: docker CLI not found"
  exit 0
fi

if ! docker info >/dev/null 2>&1; then
  log "SKIP: docker daemon unavailable"
  exit 0
fi

if docker compose version >/dev/null 2>&1; then
  docker compose --env-file .env.example -f deploy/docker-compose.yml up -d
  docker compose --env-file .env.example -f deploy/docker-compose.yml ps
elif command -v docker-compose >/dev/null 2>&1; then
  docker-compose --env-file .env.example -f deploy/docker-compose.yml up -d
  docker-compose --env-file .env.example -f deploy/docker-compose.yml ps
else
  echo "[docker-up][error] Neither 'docker compose' nor 'docker-compose' is available." >&2
  exit 1
fi
