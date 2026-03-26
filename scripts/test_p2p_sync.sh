#!/bin/bash
set -e

echo "=== Testing P2P Sync with 2 Nodes ==="

rm -rf /tmp/neocoin_node1_src /tmp/neocoin_node2_src

mkdir -p /tmp/neocoin_node1_src /tmp/neocoin_node2_src
rsync -a --exclude='.smoke-data' --exclude='data' --exclude='node_modules' --exclude='vendor' /home/neo/Download/Neocoin/ /tmp/neocoin_node1_src/
rsync -a --exclude='.smoke-data' --exclude='data' --exclude='node_modules' --exclude='vendor' /home/neo/Download/Neocoin/ /tmp/neocoin_node2_src/

echo "Building node 1..."
cd /tmp/neocoin_node1_src && go build -o neocoin ./cmd/node/
echo "Building node 2..."
cd /tmp/neocoin_node2_src && go build -o neocoin ./cmd/node/

echo "Starting node 1..."
DATA_DIR=/tmp/neocoin_node1_src/data \
GENESIS_PATH=/tmp/neocoin_node1_src/genesis/smoke.json \
CHAIN_ID=3 \
NODE_PORT=8081 \
P2P_ENABLE=true \
P2P_PORT=9091 \
P2P_SEEDS="" \
LOG_LEVEL=info \
MINING_ENABLE=true \
MINER_ADDRESS=NEO0049c3cf477a9fce2622d18245d04f011f788f7b2e248bdeb38d4ef459c37857be3d0293c3 \
/tmp/neocoin_node1_src/neocoin &
NODE1_PID=$!

sleep 3

echo "Starting node 2..."
DATA_DIR=/tmp/neocoin_node2_src/data \
GENESIS_PATH=/tmp/neocoin_node2_src/genesis/smoke.json \
CHAIN_ID=3 \
NODE_PORT=8082 \
P2P_ENABLE=true \
P2P_PORT=9092 \
P2P_SEEDS="localhost:9091" \
LOG_LEVEL=info \
MINING_ENABLE=true \
MINER_ADDRESS=NEO0049c3cf477a9fce2622d18245d04f011f788f7b2e248bdeb38d4ef459c37857be3d0293c3 \
/tmp/neocoin_node2_src/neocoin &
NODE2_PID=$!

sleep 10

echo "=== Node 1 Info ==="
curl -s http://127.0.0.1:8081/chain/info | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'Height: {d[\"height\"]}')" 2>/dev/null || echo "Node 1 unreachable"

echo "=== Node 2 Info ==="
curl -s http://127.0.0.1:8082/chain/info | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'Height: {d[\"height\"]}')" 2>/dev/null || echo "Node 2 unreachable"

echo "=== Node 1 Peers ==="
curl -s http://127.0.0.1:8081/peers 2>/dev/null || echo "No /peers endpoint"

echo "=== Node 2 Peers ==="
curl -s http://127.0.0.1:8082/peers 2>/dev/null || echo "No /peers endpoint"

kill $NODE1_PID $NODE2_PID 2>/dev/null
rm -rf /tmp/neocoin_node1_src /tmp/neocoin_node2_src

echo "=== Done ==="
