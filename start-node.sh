#!/bin/bash
#
# NeoCoin Start Script - NEVER deletes files
# Use this to start your node and it will persist forever

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

echo "========================================"
echo "  NeoCoin - Starting Node"
echo "========================================"

# Use relative paths - keeps user's home private
mkdir -p data
mkdir -p genesis
mkdir -p data/tor

# Check if Tor config exists
TORRC="data/tor/torrc2"
if [ ! -f "$TORRC" ]; then
    TOR_DATA="$SCRIPT_DIR/data/tor"
    cat > "$TORRC" << EOF
SocksPort 19051
RunAsDaemon 1
DataDirectory $TOR_DATA
Log notice stdout
HiddenServiceDir $TOR_DATA
HiddenServicePort 80 127.0.0.1:8080
HiddenServicePort 9090 127.0.0.1:9090
EOF
fi

# Set environment variables for mining
# Use existing ADMIN_TOKEN if set, otherwise generate new random token
export MINER_ADDRESS="${MINER_ADDRESS:-NEO0049c3cf477a9fce2622d18245d04f011f788f7b2e248bdeb38d4ef459c37857be3d0293c3}"
export CHAIN_ID="${CHAIN_ID:-3}"
export GENESIS_PATH="${GENESIS_PATH:-../genesis/smoke.json}"
export AUTO_MINE="${AUTO_MINE:-true}"
if [ -z "$ADMIN_TOKEN" ]; then
    ADMIN_TOKEN=$(head -c 32 /dev/urandom | base64 | tr -dc 'a-zA-Z0-9' | head -c 16)
    echo "Generated new ADMIN_TOKEN: $ADMIN_TOKEN"
fi
export ADMIN_TOKEN

# Check if NeoCoin is already running
if pgrep -f "./neocoin" > /dev/null 2>&1; then
    echo "NeoCoin is already running!"
else
    # Build NeoCoin if not built
    if [ ! -f ./neocoin ]; then
        echo "Building NeoCoin..."
        go build -o neocoin .
    fi
    
    # Start NeoCoin with mining enabled
    echo "Starting NeoCoin with mining enabled..."
    cd blockchain
    ./neocoin server &
    cd ..
    sleep 3
fi

# Check if Tor is already running with our config
if pgrep -f "tor.*torrc2" > /dev/null 2>&1; then
    echo "Tor is already running!"
else
    # Start Tor with hidden service
    echo "Starting Tor..."
    cd data/tor
    pkill -f "tor " 2>/dev/null || true
    sleep 1
    tor -f torrc2 > tor.log 2>&1 &
    cd ../..
    sleep 5
fi

# Get onion address
ONION=$(cat data/tor/hostname 2>/dev/null || echo "unknown")

echo ""
echo "========================================"
echo "  Your Node is LIVE!"
echo "========================================"
echo ""
echo "Local:     http://localhost:8080"
echo "Tor:       http://${ONION}"
echo "P2P:       localhost:9090"
echo "Explorer:  http://${ONION}/explorer/"
echo "Miner:     ${MINER_ADDRESS}"
echo ""
echo "Mining is ENABLED!"
echo "The onion address will stay the same forever!"
echo ""
echo "To check mining status:"
echo "  curl http://localhost:8080/chain/info"
echo ""