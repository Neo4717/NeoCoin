# NeoCoin - Join the Network

Welcome to NeoCoin! This guide will help you join the NeoCoin network as a new user.

## Quick Start

### Option 1: Use Public Node (Easiest)

The easiest way to start is using an existing node:

```bash
# Clone NeoCoin
git clone https://github.com/Neo4717/NeoCoin
cd NeoCoin

# Build the node
go build -o neocoin ./cmd/node/

# Run with the public seed node
# Ask the operator for their IP/URL, for example:
PEERS=http://123.45.67.89:8080 ./neocoin server
```

That's it! You're now connected to the network.

---

## Detailed Setup Guide

### Prerequisites

- **Operating System:** Linux, macOS, or Windows (with WSL)
- **Go:** Version 1.21 or higher
- **Git:** For cloning the repository

### Step 1: Install Go

**Linux (Ubuntu/Debian):**
```bash
sudo apt update
sudo apt install -y golang-go
```

**macOS:**
```bash
brew install go
```

**Windows:**
Download from https://go.dev/dl/

Verify installation:
```bash
go version
```

### Step 2: Clone the Repository

```bash
git clone https://github.com/Neo4717/NeoCoin.git
cd NeoCoin
```

### Step 3: Build the Node

```bash
go build -o neocoin ./cmd/node/
```

This creates the `neocoin` executable.

### Step 4: Get Node Address

Ask a node operator for their IP address or URL.

Example: `http://123.45.67.89:8080`

### Step 5: Run the Node

**Basic run (read-only):**
```bash
PEERS=http://123.45.67.89:8080 ./neocoin server
```

**Run with mining:**
```bash
# Create a wallet first
./neocoin wallet create

# Get your mining address (check the output)
# Then run:
MINER_ADDRESS=YOUR_ADDRESS PEERS=http://123.45.67.89:8080 ./neocoin server
```

**Run as full node with P2P:**
```bash
# Set your node as a seed for others
P2P_ENABLE=true P2P_LISTEN_ADDR=:9090 PEERS=http://123.45.67.89:8080 ./neocoin server
```

---

## Configuration Options

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CHAIN_ID` | 1 | Chain ID (1=mainnet, 3=testnet) |
| `GENESIS_PATH` | genesis/mainnet.json | Genesis file |
| `PEERS` | - | Comma-separated list of peer URLs |
| `P2P_ENABLE` | false | Enable P2P networking |
| `P2P_LISTEN_ADDR` | :9090 | P2P listen address |
| `MINER_ADDRESS` | - | Address to receive mining rewards |
| `AUTO_MINE` | false | Enable automatic mining |
| `HTTP_ADDR` | :8080 | HTTP API listen address |
| `DATA_DIR` | ./data | Data directory |

### Example: Production Node

```bash
# Full-featured node with mining
CHAIN_ID=3 \
GENESIS_PATH=genesis/smoke.json \
DATA_DIR=./data \
P2P_ENABLE=true \
P2P_LISTEN_ADDR=:9090 \
P2P_SEEDS=other-seed-node:9090 \
MINER_ADDRESS=NEO00... \
AUTO_MINE=true \
./neocoin server
```

---

## Using the Node

Once running, you can interact with your node:

### Check Chain Info
```bash
curl http://localhost:8080/chain/info
```

### Check Balance
```bash
curl http://localhost:8080/balance/YOUR_ADDRESS
```

### Create Wallet
```bash
curl -X POST http://localhost:8080/wallet/create
```

### Send Transaction
```bash
curl -X POST http://localhost:8080/tx \
  -H "Content-Type: application/json" \
  -d '{
    "toAddress": "RECIPIENT_ADDRESS",
    "amount": 100,
    "fee": 1,
    "privateKey": "YOUR_PRIVATE_KEY"
  }'
```

### Check Mempool
```bash
curl http://localhost:8080/mempool
```

---

## Troubleshooting

### "Connection refused"
- The seed node might be down
- Try a different peer or wait and retry

### "Chain ID mismatch"
- You're using different genesis or chain ID
- Make sure you use the same CHAIN_ID as the network

### "Peer not reachable"
- Check your internet connection
- Firewall might be blocking port 8080 or 9090

### Build errors
- Make sure Go is installed: `go version`
- Update Go: `go install golang.org/dl/linux@latest`

---

## Security Notes

1. **Keep your private keys safe** - Never share them
2. **Use a strong firewall** - Only open ports 8080 and 9090 if needed
3. **Verify the node** - Check /chain/info matches expected values

---

## Need Help?

- GitHub Issues: https://github.com/Neo4717/NeoCoin/issues
- Discussions: https://github.com/Neo4717/NeoCoin/discussions

---

Happy mining! ⛏️
