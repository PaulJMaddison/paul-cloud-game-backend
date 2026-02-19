#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

mkdir -p .tmp

cleanup() {
  for pid in ${PIDS:-}; do
    kill "$pid" >/dev/null 2>&1 || true
  done
}
trap cleanup EXIT INT TERM

echo "[demo] starting infra"
make docker-up
echo "[demo] running migrations"
make migrate-up

start_service() {
  local svc="$1"
  local port="$2"
  echo "[demo] starting ${svc} on :${port}"
  PORT="$port" go run "./cmd/${svc}" >".tmp/${svc}.log" 2>&1 &
  PIDS+=" $!"
}

PIDS=""
start_service gateway 8080
start_service login 8081
start_service router 8082
start_service sessions 8083
start_service matchmaking 8084

echo "[demo] services started. Logs in .tmp/*.log"
echo "[demo] press Ctrl+C to stop"
wait
