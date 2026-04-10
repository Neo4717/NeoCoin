# NeoCoin (NEO) 🐰

> **"Neo_HIT: Follow The White Rabbit"**
> Genesis Block Message

## ⚖️ License

**This software is NOT standard open source. Commercial use requires a license.**

- See [COMMERCIAL_LICENSE.md](./COMMERCIAL_LICENSE.md) for full terms
- Non-commercial use is permitted for learning, research, and personal use
- Commercial use is prohibited without written agreement from NeoCoin Protocol

---

## Quick Join (3 Steps)

### Step 1: One Command to Join
```bash
bash -c "$(curl -s https://raw.githubusercontent.com/Neo4717/NeoCoin/main/join.sh)"
```

### Step 2: Get Peer Address
Ask in our community for a peer address, or run solo first!

### Step 3: Start Mining
Your node will start automatically and begin mining.

---

## Easy Start (With Tor - Permanent Onion Address)

```bash
# Just run this script - never deletes anything!
./start-node.sh
```

Your node will start and keep the **same onion address forever**!

---

## Manual Setup

### 1. Clone & Build
```bash
git clone https://github.com/Neo4717/NeoCoin.git
cd NeoCoin/blockchain
go build -o neocoin .
```

### 2. Create Wallet
```bash
./neocoin create_wallet
```
Save your wallet address!

### 3. Run Node
```bash
# Solo (no peers yet)
MINER_ADDRESS=YOUR_ADDRESS \
GENESIS_PATH=../genesis/smoke.json \
CHAIN_ID=3 \
AUTO_MINE=true \
ADMIN_TOKEN=test \
./neocoin server

# With Peer
MINER_ADDRESS=YOUR_ADDRESS \
P2P_PEERS=PEER_ADDRESS:9090 \
GENESIS_PATH=../genesis/smoke.json \
CHAIN_ID=3 \
AUTO_MINE=true \
ADMIN_TOKEN=test \
./neocoin server
```

---

## Network Info

| | |
|---|---|
| **Chain ID** | 3 |
| **Genesis** | Neo_HIT: Follow The White Rabbit |
| **Supply** | 500 NEO (fair launch) |
| **Block Reward** | 50 NEO |
| **Halving** | Every 100 blocks |

---

## Tor Onion Service (Anonymous Node)

Run your node as a Tor hidden service for anonymous P2P connections.

### Setup Tor
```bash
# Install Tor (if not installed)
sudo apt install tor

# Configure Tor hidden service
cat > /tmp/tor-onionrc << 'EOF'
SocksPort 9050
ControlPort 9051
HiddenServiceDir /tmp/tor-onion-data
HiddenServicePort 80 127.0.0.1:8080
HiddenServicePort 443 127.0.0.1:9090
HiddenServiceVersion 3
RunAsDaemon 1
EOF

# Start Tor
tor -f /tmp/tor-onionrc &
sleep 15
```

### Get Your Onion Address
```bash
cat /tmp/tor-onion-data/hostname
```

### Connect to Onion Node
```bash
# Replace with actual onion address
P2P_PEERS=your-onion-address.onion:9090 \
MINER_ADDRESS=YOUR_ADDRESS \
GENESIS_PATH=../genesis/smoke.json \
CHAIN_ID=3 \
AUTO_MINE=true \
ADMIN_TOKEN=test \
./neocoin server
```

---

## Community

- **Discord**: [Join](#)
- **Twitter**: [@NeoCoin](#)
- **GitHub**: [Discussions](https://github.com/Neo4717/NeoCoin/discussions)

---

**Neo_HIT** 🐰
