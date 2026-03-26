#!/bin/bash
set -e

cd "$(dirname "$0")/.."

echo "=== NeoCoin Local Node ==="

# Build
echo "Building..."
go build -o neocoin ./cmd/node/

# Clean old data
echo "Cleaning old data..."
rm -rf data/

# Configure for local smoke test (Chain 3 with genesis)
export CHAIN_ID=3
export GENESIS_PATH=genesis/smoke.json
export NODE_PORT=8080
export P2P_ENABLE=false
export LOG_LEVEL=info
export METRICS_ENABLED=true

# Start node in background
echo "Starting node..."
nohup ./neocoin > /tmp/neocoin.log 2>&1 &

# Wait for node to be ready
echo "Waiting for node..."
sleep 5

# Verify node is running
if ! ss -tlnp | grep -q ":8080"; then
    echo "ERROR: Node failed to start"
    cat /tmp/neocoin.log
    exit 1
fi

echo "Node running on port 8080"

# Test endpoints
echo ""
echo "=== Testing endpoints ==="
echo "Chain Info:"
curl -s http://127.0.0.1:8080/chain/info | head -5

echo ""
echo "Metrics (first 5 lines):"
curl -s http://127.0.0.1:8080/metrics | head -5

echo ""
echo "Mempool:"
curl -s http://127.0.0.1:8080/mempool

echo ""
echo "=== Node is ready ==="
echo "To stop: pkill neocoin"
echo "To view logs: cat /tmp/neocoin.log"

# Keep running
wait
