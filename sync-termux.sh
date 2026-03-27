#!/bin/bash
# NeoCoin Sync Script - Connect Termux to Computer
# Run this on Termux to sync with your computer

echo "==================================="
echo "  NeoCoin - Sync with Computer"
echo "==================================="
echo ""

# Get computer's local IP (you need to enter this)
echo "Enter your computer's local IP address:"
echo "(On computer, run: ip addr show | grep inet)"
read -p "Local IP (e.g., 192.168.1.12): " COMPUTER_IP

if [ -z "$COMPUTER_IP" ]; then
    echo "❌ No IP entered!"
    exit 1
fi

# Test connection
echo "Testing connection to computer..."
if nc -zv -w 3 $COMPUTER_IP 9090 2>/dev/null; then
    echo "✅ Computer is reachable on port 9090"
else
    echo "❌ Cannot reach computer. Make sure:"
    echo "   - Both on same WiFi"
    echo "   - Computer is running NeoCoin"
    exit 1
fi

# Create wallet if not exists
echo ""
echo "Creating wallet..."
cd $HOME/NeoCoin/blockchain

WALLET=$(./neocoin create_wallet 2>/dev/null)
ADDRESS=$(echo "$WALLET" | grep -o '"address":"[^"]*"' | cut -d'"' -f4)

if [ -z "$ADDRESS" ]; then
    echo "❌ Failed to create wallet"
    exit 1
fi

echo "✅ Wallet: $ADDRESS"

# Ask if want to sync
echo ""
echo "Starting node synced with your computer..."
echo ""

# Run with peer
MINER_ADDRESS=$ADDRESS \
P2P_PEERS=$COMPUTER_IP:9090 \
GENESIS_PATH=../genesis/smoke.json \
CHAIN_ID=3 \
AUTO_MINE=true \
MINE_FORCE_EMPTY_BLOCKS=true \
ADMIN_TOKEN=test \
./neocoin server &

sleep 5

echo ""
echo "✅ Syncing with computer!"
echo "Check: curl http://127.0.0.1:8080/chain/info"
echo "Height: $(curl -s http://127.0.0.1:8080/chain/info | jq -r '.height')"