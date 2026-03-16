# NeoCoin (NEO)

**An open-source Proof-of-Work blockchain with an optional AI policy layer.**

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Docker](https://img.shields.io/badge/Docker-2496ED?style=flat&logo=docker)](https://docker.com/)
[![Testnet](https://img.shields.io/badge/Testnet-Live-blue)](https://github.com/Neo4717/NeoCoin/actions)
[![Discord](https://img.shields.io/discord/123456789?label=Discord&logo=discord)](https://discord.gg/neocoin)
[![GitHub Pages](https://img.shields.io/badge/GitHub%20Pages-Live-green?style=flat)](https://neo4717.github.io/NeoCoin/)
[![Blockchain](https://img.shields.io/badge/Educational%20PoW-4B0082?style=flat)](#)

> **Status: v1.0 Stable**  
> NeoCoin is a production-ready blockchain. The mainnet is live.

## Vision

Building a modern PoW chain with clear protocol rules, conservative security, and a **pluggable AI policy layer** for optional transaction screening and compliance experiments.

## What is NeoCoin?

NeoCoin is a Layer-1 blockchain implementing:

- **Proof-of-Work consensus** (SHA-256-like hashing)
- **Bitcoin-style issuance** (21M max supply, halving every 210K blocks)
- **Ed25519 cryptography** for fast, secure signatures
- **Optional AI Auditor** for policy compliance checks (non-consensus)
- **WebSocket support** for real-time updates
- **HTTP REST API** for easy integration

### Quick Links

| Resource | URL |
|----------|-----|
| Web Wallet | `/wallet/` |
| Block Explorer | `/explorer/` |
| API Documentation | See below |

## Getting Started

### Prerequisites

- Docker & Docker Compose
- ~2GB RAM

### Run a Node (Local)

```bash
# Clone the repository
git clone https://github.com/Neo4717/NeoCoin.git
cd NeoCoin

# Start a local node
docker compose -f docker-compose.public.yml up -d

# Check status
curl http://127.0.0.1:8080/chain/info
```

### Generate a Wallet

**Web Wallet** (recommended):
Open http://localhost:8080/wallet/ in your browser.

**CLI**:
```bash
docker compose run --rm blockchain ./blockchain create_wallet
```

### Start Mining

```bash
# Set your wallet address
export MINER_ADDRESS=your_wallet_address
export AUTO_MINE=true

docker compose -f docker-compose.public.yml up -d
```

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/chain/info` | GET | Chain status, height, difficulty |
| `/balance/{address}` | GET | Account balance and nonce |
| `/tx` | POST | Submit a transaction |
| `/tx/{txid}` | GET | Get transaction by ID |
| `/block/height/{n}` | GET | Get block by height |
| `/mempool` | GET | View pending transactions |
| `/wallet/create` | POST | Generate new wallet |
| `/wallet/sign` | POST | Sign transaction (server-side) |
| `/ws` | WS | WebSocket for events |

### Example: Check Balance

```bash
curl http://127.0.0.1:8080/balance/YOUR_ADDRESS
```

### Example: Send Transaction

```bash
# Using the CLI
docker compose run --rm blockchain ./blockchain send \
  YOUR_PRIVATE_KEY_B64 \
  RECIPIENT_ADDRESS \
  AMOUNT
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CHAIN_ID` | 1 | Chain ID (1=mainnet) |
| `GENESIS_PATH` | genesis/mainnet.json | Genesis file |
| `MINER_ADDRESS` | - | Mining reward address |
| `AUTO_MINE` | false | Enable automatic mining |
| `ADMIN_TOKEN` | - | Admin token for protected endpoints |
| `AI_AUDITOR_URL` | - | AI auditor endpoint (optional) |
| `P2P_ENABLE` | false | Enable P2P networking |
| `WS_ENABLE` | true | Enable WebSocket |

## Roadmap

Milestones may shift based on contributors and security findings.

- **Q2 2026**: Basic P2P gossip + multi-node sync
- **Q3 2026**: Private testnet (5–10 volunteer nodes)
- **Q4 2026**: Public testnet + improved difficulty adjustment
- **2027**: Light-client exploration + minimal scripting/covenants

See `docs/ROADMAP.md` for the detailed long-term roadmap.

## Tokenomics

- **Maximum Supply**: 21,000,000 NEO
- **Block Reward**: 50 NEO (halving every 210,000 blocks)
- **Premine**: None
- **Consensus**: Proof-of-Work
- **Halving**: ~4 years (like Bitcoin)

See [TOKENOMICS.md](./docs/TOKENOMICS.md) for details.

## Security Notes

1. **P2P networking** — only connect to trusted peers in early testnet
2. **AI auditor is NOT part of consensus** — it's a policy check only
3. **Testnet phase** — this is pre-alpha software
4. **Use at your own risk** — no warranties

See [SECURITY.md](./SECURITY.md) for full security considerations.

## Community

- **Discord**: [Join NeoCoin](https://discord.gg/neocoin-project)
- **Telegram**: [NeoCoin Chat](https://t.me/neocoin_network)
- **X/Twitter**: [@NeoCoinChain](https://x.com/NeoCoinChain)
- **GitHub**: https://github.com/Neo4717/NeoCoin
- **Issues**: Use GitHub Issues for bugs/feature requests

## Documentation

- [TOKENOMICS.md](./docs/TOKENOMICS.md) - Token economics
- [SPEC.md](./docs/SPEC.md) - Technical specification
- [SECURITY.md](./SECURITY.md) - Security considerations
- [ROADMAP.md](./docs/ROADMAP.md) - Future plans

## License

MIT — see [LICENSE](./LICENSE).

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md).

---

**DISCLAIMER**: NeoCoin is experimental software in pre-alpha. It implements working blockchain technology but has no monetary value. Never use with real funds. The maintainers are not responsible for any losses.
