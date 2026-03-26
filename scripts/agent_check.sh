#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
FIX="${1:-}"

if [ "$FIX" != "" ] && [ "$FIX" != "--fix" ]; then
  echo "usage: scripts/agent_check.sh [--fix]"
  exit 2
fi

cd "$ROOT_DIR/blockchain"

export GOCACHE="${GOCACHE:-/tmp/gocache}"

echo "[1/3] gofmt"
if [ "$FIX" = "--fix" ]; then
  gofmt -w .
else
  UNFORMATTED="$(gofmt -l . | wc -l | tr -d ' ')"
  if [ "$UNFORMATTED" != "0" ]; then
    echo "gofmt needed. Re-run with --fix."
    gofmt -l .
    exit 1
  fi
fi

echo "[2/3] go test"
go test ./...

echo "[3/3] go vet"
go vet ./...

echo "OK"

