#!/usr/bin/env bash
set -euo pipefail

if docker compose version >/dev/null 2>&1; then
  docker compose -f deploy/docker-compose.yml down -v
elif command -v docker-compose >/dev/null 2>&1; then
  docker-compose -f deploy/docker-compose.yml down -v
else
  echo "Neither 'docker compose' nor 'docker-compose' is available."
  exit 1
fi
