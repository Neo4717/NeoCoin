#!/bin/bash
#
# NeoCoin Start Script - NEVER deletes files
# Use this to start your node and it will persist forever

set -e

echo "========================================"
echo "  NeoCoin - Starting Node"
echo "========================================"

cd "$(dirname "$0")"

# Check if data directory exists
mkdir -p data
mkdir -p genesis
mkdir -p data/tor

# Check if Tor config exists
if [ ! -f data/tor/torrc2 ]; then
    cat > data/tor/torrc2 << 'EOF'
SocksPort 19051
RunAsDaemon 1
DataDirectory /home/neo/Download/Neocoin/data/tor
Log notice stdout
HiddenServiceDir /home/neo/Download/Neocoin/data/tor
HiddenServicePort 80 127.0.0.1:8080
HiddenServicePort 9090 127.0.0.1:9090
EOF
fi

# Check if NeoCoin is already running
if pgrep -f "./neocoin" > /dev/null 2>&1; then
    echo "NeoCoin is already running!"
else
    # Build NeoCoin if not built
    if [ ! -f ./neocoin ]; then
        echo "Building NeoCoin..."
        go build -o neocoin .
    fi
    
    # Start NeoCoin
    echo "Starting NeoCoin..."
    ./neocoin &
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
    tor -f torrc2 2>&1 &
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
echo ""
echo "The onion address will stay the same forever!"
echo "Use Tor port 19051 for SOCKS proxy"
echo ""
echo "To access from another computer via Tor:"
echo "  curl --socks5-hostname localhost:19051 http://${ONION}/"
echo ""