#!/bin/bash
#
# NeoCoin Start Script - NEVER deletes files
# Use this to start your node and it will persist forever
#

set -e

echo "========================================"
echo "  NeoCoin - Starting Node"
echo "========================================"

cd "$(dirname "$0")"

# Check if data directory exists, if not create it
mkdir -p data
mkdir -p genesis

echo "Starting NeoCoin with Tor..."
echo ""

# Start docker-compose (this preserves all data)
docker compose -f docker-compose.tor.yml up -d

echo "Waiting for node to start..."
sleep 5

# Get onion address
ONION=$(docker exec neocoin-tor cat /var/lib/tor/hidden_service/hostname 2>/dev/null || echo "unknown")

echo ""
echo "========================================"
echo "  Your Node is LIVE!"
echo "========================================"
echo ""
echo "Local:  http://localhost:8080"
echo "Tor:    http://${ONION}"
echo "Explorer: http://${ONION}/explorer/"
echo ""
echo "The onion address will stay the same forever!"
echo "Just keep this script running or restart with:"
echo "  docker compose -f docker-compose.tor.yml start"
echo ""

# Keep running and show logs
docker compose -f docker-compose.tor.yml logs -f