#!/bin/bash
# NeoCoin Local Connect - Auto-find and connect to computer
# Run this on Termux when on same WiFi as your computer

echo "==================================="
echo "  NeoCoin - Local Connect"
echo "==================================="

# Your termux IP
MY_IP=$(ip addr show | grep "inet " | grep -v "127.0.0.1" | awk '{print $2}' | cut -d'/' -f1)
echo "Your IP: $MY_IP"

# Get base IP (e.g., 192.168.1)
BASE=$(echo $MY_IP | sed 's/\.[0-9]*$//')
echo "Scanning for computer..."

# Try common computer IPs (1-30)
COMPUTER_IP=""
for i in $(seq 1 30); do
    IP="${BASE}.${i}"
    # Skip your own IP
    if [ "$IP" = "$MY_IP" ]; then
        continue
    fi
    
    # Check if port 9090 is open
    if nc -zv -w 1 $IP 9090 2>/dev/null; then
        echo "✅ Found computer at: $IP"
        COMPUTER_IP=$IP
        break
    fi
done

if [ -z "$COMPUTER_IP" ]; then
    echo "❌ Computer not found!"
    echo "Make sure NeoCoin is running on your computer"
    echo ""
    echo "Try running manually - enter computer IP:"
    read -p "Computer IP (e.g., 192.168.1.12): " COMPUTER_IP
fi

if [ -z "$COMPUTER_IP" ]; then
    echo "❌ No IP provided. Exiting."
    exit 1
fi

echo ""
echo "Connecting to: $COMPUTER_IP:9090"
echo ""

# Go to NeoCoin
cd ~/NeoCoin/blockchain

# Get or create wallet
WALLET=$(./neocoin create_wallet 2>/dev/null)
ADDRESS=$(echo "$WALLET" | grep -o '"address":"[^"]*"' | cut -d'"' -f4)

echo "Your wallet: $ADDRESS"
echo "Starting synced node..."
echo ""

# Run connected to computer
exec env \
    MINER_ADDRESS=$ADDRESS \
    P2P_PEERS=$COMPUTER_IP:9090 \
    GENESIS_PATH=../genesis/smoke.json \
    CHAIN_ID=3 \
    P2P_ENABLE=true \
    AUTO_MINE=true \
    MINE_FORCE_EMPTY_BLOCKS=true \
    ADMIN_TOKEN=test \
    ./neocoin server