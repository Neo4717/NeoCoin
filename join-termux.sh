#!/bin/bash
# NeoCoin Node Join Script for Termux (Android)
# Install dependencies and run node

echo "==================================="
echo "  NeoCoin - Follow The White Rabbit"
echo "  Termux Installation"
echo "==================================="
echo ""

# Check if Termux
if [ ! -d "/data/data/com.termux" ]; then
    echo "⚠️ This script is for Termux on Android"
fi

# Update packages
echo "📦 Updating packages..."
pkg update -y 2>/dev/null || apt update -y 2>/dev/null

# Install Go
echo "📦 Installing Go..."
pkg install -y golang git 2>/dev/null || apt install -y golang git 2>/dev/null

# Check Go installation
if ! command -v go &> /dev/null; then
    echo "❌ Go installation failed!"
    echo "Try: pkg install golang"
    exit 1
fi

echo "✅ Go installed: $(go version)"

# Clone repo
if [ ! -d "$HOME/NeoCoin" ]; then
    echo "📦 Cloning NeoCoin..."
    cd $HOME
    git clone https://github.com/Neo4717/NeoCoin.git
fi

cd $HOME/NeoCoin

# Build
echo "🔨 Building NeoCoin..."
cd blockchain
go build -o neocoin .

if [ ! -f "./neocoin" ]; then
    echo "❌ Build failed!"
    exit 1
fi

echo "✅ Build successful!"

# Create wallet
echo ""
echo "👛 Creating wallet..."
WALLET=$(./neocoin create_wallet 2>/dev/null)
ADDRESS=$(echo "$WALLET" | grep -o '"address":"[^"]*"' | cut -d'"' -f4)

if [ -z "$ADDRESS" ]; then
    echo "❌ Failed to create wallet"
    exit 1
fi

echo ""
echo "✅ Your wallet: $ADDRESS"
echo "💾 Save this address!"
echo ""

# Ask for peer
echo "Enter peer IP (or press Enter to run solo):"
read -r PEER

if [ -z "$PEER" ]; then
    echo "Running solo (no peer)..."
    PEER_CMD=""
else
    echo "Connecting to peer: $PEER"
    PEER_CMD="P2P_PEERS=$PEER:9090"
fi

echo ""
echo "🚀 Starting node..."
echo ""

cd blockchain
eval "MINER_ADDRESS=$ADDRESS GENESIS_PATH=../genesis/smoke.json CHAIN_ID=3 P2P_ENABLE=true AUTO_MINE=true MINE_FORCE_EMPTY_BLOCKS=true ADMIN_TOKEN=test $PEER_CMD ./neocoin server" &

sleep 5

echo ""
echo "✅ Node started!"
echo "Check: curl http://127.0.0.1:8080/chain/info"
echo "Blocks: curl http://127.0.0.1:8080/chain/info | grep height"
