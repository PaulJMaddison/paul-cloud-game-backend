#!/usr/bin/env bash
set -euo pipefail

log() { echo "[test] $*"; }
fail() { echo "[test][error] $*" >&2; exit 1; }

if ! command -v go >/dev/null 2>&1; then
  fail "Go toolchain is required but was not found in PATH"
fi

log "running unit tests"
go test ./...

if ! command -v docker >/dev/null 2>&1; then
  log "SKIP: docker CLI not found; skipping integration tests"
  exit 0
fi
if ! docker info >/dev/null 2>&1; then
  log "SKIP: docker daemon unavailable; skipping integration tests"
  exit 0
fi

log "running integration tests"
go test -tags=integration ./...
