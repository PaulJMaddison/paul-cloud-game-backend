#!/usr/bin/env bash
set -euo pipefail

if docker compose version >/dev/null 2>&1; then
  docker compose -f deploy/docker-compose.yml up -d
  docker compose -f deploy/docker-compose.yml ps
elif command -v docker-compose >/dev/null 2>&1; then
  docker-compose -f deploy/docker-compose.yml up -d
  docker-compose -f deploy/docker-compose.yml ps
else
  echo "Neither 'docker compose' nor 'docker-compose' is available."
  exit 1
fi
