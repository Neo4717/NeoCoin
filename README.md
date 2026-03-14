# NeoCoin

**Educational Proof-of-Work blockchain with optional AI policy auditing.**

> ⚠️ **Warning:** This is a **prototype** — no mainnet exists yet. Use for learning/development only. No economic value.

## Overview

NeoCoin is an experimental cryptocurrency blockchain built with Go. It implements:

- **Proof-of-Work** consensus (SHA-256-like)
- **Bitcoin-style** halving schedule (21M max supply)
- **Optional AI Auditor** for policy checking (via external API)
- **Ed25519** public key cryptography
- **WebSocket** support for real-time updates

## Quick Start

### Prerequisites

- Docker & Docker Compose
- ~2GB RAM available

### Run a Node

```bash
# Clone the repository
git clone https://github.com/YOUR_USERNAME/neocoin.git
cd neocoin

# Generate a wallet (optional)
docker compose run --rm blockchain ./blockchain create_wallet

# Start mining (replace with your address)
docker compose run -e MINER_ADDRESS=YOUR_ADDRESS -e AUTO_MINE=true blockchain
```

Or without mining:
```bash
docker compose up -d blockchain
```

### API

| Endpoint | Description |
|----------|-------------|
| `GET /chain/info` | Chain status |
| `GET /block/height/{n}` | Block by height |
| `GET /balance/{address}` | Account balance |
| `POST /tx` | Submit transaction |
| `GET /mempool` | Pending transactions |
| `GET /ws` | WebSocket for events |

Example:
```bash
curl http://127.0.0.1:8080/chain/info
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CHAIN_ID` | 1 | Chain ID (1=mainnet, 2=testnet) |
| `GENESIS_PATH` | genesis/mainnet.json | Genesis file |
| `MINER_ADDRESS` | - | Mining reward address |
| `AUTO_MINE` | false | Enable automatic mining |
| `MINE_FORCE_EMPTY_BLOCKS` | false | Mine even with no txs |
| `AI_AUDITOR_URL` | - | AI auditor endpoint |

### Networks

- **Mainnet**: `docker compose -f docker-compose.mainnet.yml up`
- **Testnet**: `docker compose -f docker-compose.testnet.yml up`
- **Smoke test**: `docker compose -f docker-compose.smoke.yml up`

## Documentation

- [TOKENOMICS.md](./docs/TOKENOMICS.md) - Token economics
- [SPEC.md](./docs/SPEC.md) - Technical specification
- [SECURITY.md](./docs/SECURITY.md) - Security considerations
- [ROADMAP.md](./docs/ROADMAP.md) - Future plans

## Status

- ✅ Basic blockchain works
- ✅ Mining functional
- ✅ Transaction support
- ✅ WebSocket events
- ⚠️ **No mainnet** — testnet only
- ⚠️ **No public P2P network** — single node only

## Limitations

1. **No mainnet** — This is testnet/prototype only
2. **No value** — Tokens have no economic value
3. **Single node** — No peer-to-peer network yet
4. **Educational** — For learning purposes only
5. **No support** — No warranty, no support team

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md).

## License

MIT — see [LICENSE](./LICENSE).

## Warnings

- ⚠️ **Experimental software** — Use at your own risk
- ⚠️ **No mainnet** — Numbers may change before launch
- ⚠️ **No token sale** — We will never sell tokens
- ⚠️ **No financial advice** — This is not financial advice
