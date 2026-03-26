#!/bin/bash
# NeoCoin Node Join Script
# Run this to join the NeoCoin network!

echo "==================================="
echo "  NeoCoin - Follow The White Rabbit"
echo "==================================="
echo ""

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed!"
    echo "Install Go: https://go.dev/doc/install"
    exit 1
fi

# Clone repo if not exists
if [ ! -d "NeoCoin" ]; then
    echo "📦 Cloning NeoCoin repository..."
    git clone https://github.com/Neo4717/NeoCoin.git
    cd NeoCoin
else
    cd NeoCoin
    echo "📂 Updating NeoCoin..."
    git pull origin main
fi

# Build
echo "🔨 Building NeoCoin..."
cd blockchain
go build -o neocoin .

# Create wallet
echo ""
echo "👛 Creating your wallet..."
WALLET=$(./neocoin create_wallet 2>/dev/null)
ADDRESS=$(echo "$WALLET" | grep -o '"address":"[^"]*"' | cut -d'"' -f4)

echo ""
echo "✅ Your wallet address:"
echo "$ADDRESS"
echo ""
echo "💾 Save this address! You'll need it for mining."
echo ""

# Ask for peer
echo "Enter peer address to connect (press Enter for first-time setup):"
read -p "Peer: " PEER

if [ -z "$PEER" ]; then
    PEER_CMD=""
else
    PEER_CMD="P2P_PEERS=$PEER"
fi

echo ""
echo "🚀 Starting your node..."
echo ""

# Run node
cd blockchain
eval "MINER_ADDRESS=$ADDRESS GENESIS_PATH=../genesis/smoke.json CHAIN_ID=3 P2P_ENABLE=true AUTO_MINE=true ADMIN_TOKEN=test $PEER_CMD ./neocoin server"

echo ""
echo "📊 Your node is running!"
echo "Check status: curl http://127.0.0.1:8080/chain/info"
