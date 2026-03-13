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

RBF_DATA_DIR="${ROOT_DIR}/.rbf-data"
rm -rf "$RBF_DATA_DIR"
mkdir -p "$RBF_DATA_DIR"
export CHAIN_DATA_DIR="$RBF_DATA_DIR"

COMPOSE=(docker compose -f docker-compose.yml -f docker-compose.rbf.yml)

echo "Cleaning previous rbf stack..."
"${COMPOSE[@]}" down -v --remove-orphans >/dev/null 2>&1 || true

echo "[1/7] Build images"
"${COMPOSE[@]}" build blockchain

echo "[2/7] Generate miner wallet"
MINER_JSON="$("${COMPOSE[@]}" run --rm blockchain ./blockchain create_wallet)"
MINER_ADDR="$(printf '%s' "$MINER_JSON" | python -c 'import json,sys; print(json.loads(sys.stdin.read())["address"])')"
MINER_PRIV="$(printf '%s' "$MINER_JSON" | python -c 'import json,sys; print(json.loads(sys.stdin.read())["privateKey"])')"

echo "[3/7] Write genesis + start stack with AUTO_MINE=false"
GENESIS_PATH="genesis/rbf.json"
mkdir -p "$(dirname "$GENESIS_PATH")"
cat >"$GENESIS_PATH" <<EOF
{
  "network": "neocoin-rbf",
  "chainId": 4,
  "timestamp": 1700000000,
  "genesisMinerAddress": "$MINER_ADDR",
  "initialSupply": "1000000",
  "genesisMessage": "Neocoin RBF test genesis",
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

CHAIN_ID=4 GENESIS_PATH="$GENESIS_PATH" MINER_ADDRESS="$MINER_ADDR" "${COMPOSE[@]}" up -d --remove-orphans blockchain

echo "Waiting for HTTP..."
for i in $(seq 1 30); do
  if curl -fsS http://localhost:8080/health >/dev/null 2>&1; then
    break
  fi
  sleep 1
done
curl -fsS http://localhost:8080/health >/dev/null

echo "[4/7] Generate recipient wallet"
RECIP_JSON="$("${COMPOSE[@]}" exec -T blockchain ./blockchain create_wallet)"
RECIP_ADDR="$(printf '%s' "$RECIP_JSON" | python -c 'import json,sys; print(json.loads(sys.stdin.read())["address"])')"

echo "[5/7] Create tx nonce=1 fee=1 (sign only) and submit"
TX1="$("${COMPOSE[@]}" exec -T blockchain ./blockchain sign_tx "$MINER_PRIV" "$RECIP_ADDR" 10 1 1 "rbf test" http://localhost:8080)"
curl -sS -X POST -H "Content-Type: application/json" -d "$TX1" http://localhost:8080/tx >/dev/null

echo "[6/7] Replace tx nonce=1 fee=2 (RBF) and submit"
TX2="$("${COMPOSE[@]}" exec -T blockchain ./blockchain sign_tx "$MINER_PRIV" "$RECIP_ADDR" 10 2 1 "rbf test" http://localhost:8080)"
curl -sS -X POST -H "Content-Type: application/json" -d "$TX2" http://localhost:8080/tx >/dev/null

echo "[7/7] Verify mempool has fee=2 for nonce=1"
MP_JSON="$(curl -sS http://localhost:8080/mempool)"
FEE="$(printf '%s' "$MP_JSON" | python -c 'import json,sys; mp=json.loads(sys.stdin.read()); print(mp["txs"][0]["fee"])')"
NONCE="$(printf '%s' "$MP_JSON" | python -c 'import json,sys; mp=json.loads(sys.stdin.read()); print(mp["txs"][0]["nonce"])')"
SIZE="$(printf '%s' "$MP_JSON" | python -c 'import json,sys; mp=json.loads(sys.stdin.read()); print(mp["size"])')"

if [ "$SIZE" != "1" ] || [ "$NONCE" != "1" ] || [ "$FEE" != "2" ]; then
  echo "RBF test failed. mempool=$MP_JSON"
  exit 1
fi

echo "OK"

"${COMPOSE[@]}" down -v --remove-orphans >/dev/null 2>&1 || true
