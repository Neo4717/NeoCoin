#!/bin/bash
# NeoCoin Auto-Sync - Connect to Computer Automatically
# Run this on Termux when on same WiFi as your computer

echo "NeoCoin - Auto Sync"
echo "=================="

# Common router IPs (try these first)
ROUTER_IPS=("192.168.1.1" "192.168.0.1" "192.168.1.254")

# Get your local IP range
LOCAL_IP=$(ip addr show | grep "inet " | grep -v "127.0.0.1" | awk '{print $2}' | cut -d'/' -f1 | head -1)

if [ -z "$LOCAL_IP" ]; then
    echo "❌ Cannot detect local IP"
    exit 1
fi

# Get base IP (e.g., 192.168.1)
BASE_IP=$(echo $LOCAL_IP | sed 's/\.[0-9]*$//')

echo "Local IP: $LOCAL_IP"
echo "Scanning for computer on local network..."

# Scan common computer IPs
FOUND=false
for i in {2..20}; do
    TEST_IP="${BASE_IP}.${i}"
    if nc -zv -w 1 $TEST_IP 9090 2>/dev/null; then
        echo "✅ Found computer at: $TEST_IP"
        COMPUTER_IP=$TEST_IP
        FOUND=true
        break
    fi
done

if [ "$FOUND" = false ]; then
    echo "❌ Computer not found on local network"
    echo "Make sure NeoCoin is running on your computer"
    echo "Then enter computer's IP manually:"
    read -p "IP: " COMPUTER_IP
fi

cd $HOME/NeoCoin/blockchain

# Create wallet
WALLET=$(./neocoin create_wallet 2>/dev/null)
ADDRESS=$(echo "$WALLET" | grep -o '"address":"[^"]*"' | cut -d'"' -f4)

echo "Wallet: $ADDRESS"
echo "Connecting to: $COMPUTER_IP:9090"
echo ""

# Run synced
MINER_ADDRESS=$ADDRESS \
P2P_PEERS=$COMPUTER_IP:9090 \
GENESIS_PATH=../genesis/smoke.json \
CHAIN_ID=3 \
AUTO_MINE=true \
MINE_FORCE_EMPTY_BLOCKS=true \
ADMIN_TOKEN=test \
./neocoin server