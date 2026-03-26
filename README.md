# NeoCoin (NEO) 🐰

> **"Neo_HIT: Follow The White Rabbit"**
> Genesis Block Message

A community-driven Proof-of-Work blockchain. Fair launch, no premine.

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

## Community

- **Discord**: [Join](#)
- **Twitter**: [@NeoCoin](#)
- **GitHub**: [Discussions](https://github.com/Neo4717/NeoCoin/discussions)

---

**Neo_HIT** 🐰
