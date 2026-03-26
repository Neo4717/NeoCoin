# NeoCoin (NEO) - Follow The White Rabbit

> **"Follow The White Rabbit - Neo_HIT"**
> 
> Genesis Block Message

A community-driven Proof-of-Work blockchain. Fair launch, no premine, built for everyone.

## Why NeoCoin?

- **Fair Launch** - Everyone mines together, no pre-mined coins
- **Simple** - Clean PoW consensus, no complex mechanics
- **Your Chain** - Community owned, community run
- **Built for AI Agents** - Optional AI policy layer for smart transactions

## Quick Start

### Clone & Build

```bash
# Clone the repo
git clone https://github.com/Neo4717/NeoCoin.git
cd NeoCoin/blockchain

# Build
go build -o neocoin .
```

### Run Your Own Node

```bash
# Generate a wallet address
./neocoin create_wallet

# Run with your miner address
MINER_ADDRESS=YOUR_WALLET_ADDRESS \
GENESIS_PATH=../genesis/smoke.json \
CHAIN_ID=3 \
AUTO_MINE=true \
./neocoin server
```

### Connect to Other Peers

```bash
# Add peer address (get from community)
P2P_ENABLE=true \
P2P_PEERS=PEER_ADDRESS:9090 \
MINER_ADDRESS=YOUR_WALLET_ADDRESS \
GENESIS_PATH=../genesis/smoke.json \
CHAIN_ID=3 \
AUTO_MINE=true \
./neocoin server
```

### Check Status

```bash
# Chain info
curl http://127.0.0.1:8080/chain/info

# Your balance
curl http://127.0.0.1:8080/balance/YOUR_ADDRESS
```

## Network Details

| | |
|---|---|
| **Chain ID** | 3 (Smoke Test) |
| **Genesis Message** | Follow The White Rabbit - Neo_HIT |
| **Initial Supply** | 500 NEO (fair launch) |
| **Block Reward** | 50 NEO |
| **Halving** | Every 100 blocks |
| **P2P Port** | 9090 |
| **API Port** | 8080 |

## Getting Peer Addresses

Join our community to get active peer addresses:
- **Discord**: [Join here](#)
- **Twitter**: [@NeoCoin](#)
- **GitHub Discussions**: [Ask for peers](https://github.com/Neo4717/NeoCoin/discussions)

## Building from Source

```bash
# Requirements
- Go 1.21+
- Docker (optional)

# Build binary
cd blockchain
go build -o neocoin .

# Or use Docker
docker compose -f docker-compose.public.yml up -d
```

## Community

- Discord: [Join our server](#)
- Twitter: [@NeoCoin](#)
- Issues: [GitHub Issues](https://github.com/Neo4717/NeoCoin/issues)

## License

MIT - Open source, forever.

---

**Neo_HIT** 🐰
