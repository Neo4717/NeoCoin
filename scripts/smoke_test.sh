#!/usr/bin/env bash
set -euo pipefail

if ! command -v docker >/dev/null 2>&1; then
  echo "docker not found"
  exit 1
fi
if ! docker compose version >/dev/null 2>&1; then
  echo "docker compose not found"
  exit 1
fi

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

SMOKE_DATA_DIR="${ROOT_DIR}/.smoke-data"
rm -rf "$SMOKE_DATA_DIR"
mkdir -p "$SMOKE_DATA_DIR"
export CHAIN_DATA_DIR="$SMOKE_DATA_DIR"

COMPOSE=(docker compose -f docker-compose.yml -f docker-compose.smoke.yml)

echo "Cleaning previous smoke stack..."
"${COMPOSE[@]}" down -v --remove-orphans >/dev/null 2>&1 || true

echo "[1/6] Build images"
"${COMPOSE[@]}" build blockchain

echo "[2/6] Generate miner wallet"
MINER_JSON="$("${COMPOSE[@]}" run --rm blockchain ./blockchain create_wallet)"
MINER_ADDR="$(printf '%s' "$MINER_JSON" | python -c 'import json,sys; print(json.loads(sys.stdin.read())["address"])')"
echo "Miner address: $MINER_ADDR"

echo "[3/6] Write genesis + start stack"
GENESIS_PATH="genesis/smoke.json"
mkdir -p "$(dirname "$GENESIS_PATH")"
cat >"$GENESIS_PATH" <<EOF
{
  "network": "neocoin-smoke",
  "chainId": 3,
  "timestamp": 1700000000,
  "genesisMinerAddress": "$MINER_ADDR",
  "initialSupply": "1000000",
  "genesisMessage": "Neocoin smoke test genesis",
  "monetaryPolicy": {
    "initialBlockReward": "50",
    "halvingInterval": 100,
    "minerFeeShare": 100,
    "tailEmission": "0"
  },
  "consensusParams": {
    "difficultyEnable": false,
    "difficultyTargetMs": 1000,
    "difficultyWindow": 10,
    "difficultyMaxStepBits": 1,
    "difficultyMinBits": 1,
    "difficultyMaxBits": 255,
    "genesisDifficultyBits": 1,
    "medianTimePastWindow": 11,
    "maxTimeDrift": 7200,
    "maxBlockSize": 1000000,
    "merkleEnable": false,
    "merkleActivationHeight": 0,
    "binaryEncodingEnable": false,
    "binaryEncodingActivationHeight": 0
  }
}
EOF

CHAIN_ID=3 GENESIS_PATH="$GENESIS_PATH" MINER_ADDRESS="$MINER_ADDR" ADMIN_TOKEN=test MINE_FORCE_EMPTY_BLOCKS=true "${COMPOSE[@]}" up -d --remove-orphans blockchain

echo "Waiting for HTTP..."
for i in $(seq 1 30); do
  if "${COMPOSE[@]}" exec -T blockchain wget -qO- http://127.0.0.1:8080/health >/dev/null 2>&1; then
    break
  fi
  sleep 1
done
"${COMPOSE[@]}" exec -T blockchain wget -qO- http://127.0.0.1:8080/health >/dev/null

echo "[4/6] Generate user wallet"
USER_JSON="$("${COMPOSE[@]}" exec -T blockchain ./blockchain create_wallet)"
USER_PRIV="$(printf '%s' "$USER_JSON" | python -c 'import json,sys; print(json.loads(sys.stdin.read())["privateKey"])')"
USER_ADDR="$(printf '%s' "$USER_JSON" | python -c 'import json,sys; print(json.loads(sys.stdin.read())["address"])')"
echo "User address: $USER_ADDR"

echo "[5/6] Send coins from miner -> user"
MINER_PRIV="$(printf '%s' "$MINER_JSON" | python -c 'import json,sys; print(json.loads(sys.stdin.read())["privateKey"])')"
"${COMPOSE[@]}" exec -T blockchain ./blockchain send "$MINER_PRIV" "$USER_ADDR" 10 1 1 "smoke test" http://127.0.0.1:8080 >/dev/null

echo "Waiting for miner to include transaction..."
mined=false
for i in $(seq 1 30); do
  USER_BAL="$("${COMPOSE[@]}" exec -T blockchain ./blockchain get_balance "$USER_ADDR" http://127.0.0.1:8080 | python -c 'import json,sys; print(json.loads(sys.stdin.read())["balance"])')"
  if [ "$USER_BAL" -ge 10 ]; then
    mined=true
    break
  fi
  sleep 1
done
if [ "$mined" != "true" ]; then
  echo "transaction was not mined within timeout"
  exit 1
fi

echo "[6/6] Check balances"
echo "Miner:"
"${COMPOSE[@]}" exec -T blockchain ./blockchain get_balance "$MINER_ADDR" http://127.0.0.1:8080
echo "User:"
"${COMPOSE[@]}" exec -T blockchain ./blockchain get_balance "$USER_ADDR" http://127.0.0.1:8080

echo "OK"
