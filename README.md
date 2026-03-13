# Neocoin

Experimental Proof-of-Work blockchain with optional AI-based transaction policy auditing.

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Docker](https://img.shields.io/badge/Docker-ready-2496ED?style=flat&logo=docker)
[![Status](https://img.shields.io/badge/Status-Prototype-orange?style=flat)

## ⚠️ Status: Early Prototype (March 2026)

**DO NOT USE WITH REAL VALUE.** This is experimental software.

### What Works
- Single-node PoW blockchain with BoltDB persistence
- Ed25519 transaction signing
- Manual and auto-mining (empty blocks)
- HTTP API + WebSocket events
- Address transaction indexing
- Basic block explorer UI

### What Doesn't Work / Is Draft
- Multi-node P2P (unauthenticated, for test networks only)
- AI Auditor is non-deterministic (policy check only, NOT part of consensus)
- No security audits
- No reproducible builds yet
- No bug bounty

## Quick Start

### Prerequisites
- Docker + Docker Compose
- `git`

### Run a Node

```bash
# Clone or navigate to the repo
cd neocoin

# Generate a wallet (miner address)
docker compose run --rm blockchain ./blockchain create_wallet

# Copy the address from output, then start mining:
docker compose run -e MINER_ADDRESS=<your_address> -e AUTO_MINE=true -e MINE_FORCE_EMPTY_BLOCKS=true blockchain

# Or use the default single-node setup:
docker compose up -d blockchain
```

### API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/chain/info` | GET | Chain height, work, difficulty |
| `/balance/{address}` | GET | Account balance + nonce |
| `/tx/{txid}` | GET | Transaction details |
| `/tx/proof/{txid}` | GET | Merkle proof (v2 blocks) |
| `/block/height/{height}` | GET | Block by height |
| `/block/hash/{hash}` | GET | Block by hash |
| `/mempool` | GET | Pending transactions |
| `/address/{addr}/txs` | GET | Address transaction history |
| `/tx` | POST | Submit signed transaction |
| `/mine/once` | POST | Mine one block (admin) |
| `/ws` | WS | WebSocket events |
| `/explorer/` | GET | Block explorer UI |
| `/metrics` | GET | Prometheus metrics |

#### Examples (curl)

```bash
# Chain info
curl http://127.0.0.1:8080/chain/info

# Balance
curl http://127.0.0.1:8080/balance/<address>

# Submit transaction
curl -X POST http://127.0.0.1:8080/tx \
  -H "Content-Type: application/json" \
  -d '{
    "type": "transfer",
    "chainId": 1,
    "fromPubKey": "<base64>",
    "toAddress": "<hex>",
    "amount": 100,
    "fee": 1,
    "nonce": 0,
    "data": "",
    "signature": "<base64>"
  }'

# Mempool
curl http://127.0.0.1:8080/mempool
```

#### WebSocket

```javascript
const ws = new WebSocket('ws://127.0.0.1:8080/ws');
ws.onmessage = (event) => {
  console.log(JSON.parse(event.data));
};
```

### CLI Commands

```bash
# Create wallet
docker compose exec blockchain ./blockchain create_wallet

# Check balance
docker compose exec blockchain ./blockchain get_balance <address>

# Send transaction
docker compose exec blockchain ./blockchain send <private_key> <to_address> <amount> <fee> <nonce> <data> <server_url>
```

## Project Structure

```
neocoin/
├── blockchain/          # Go blockchain node
│   ├── main.go         # Server + CLI entrypoint
│   ├── chain.go        # Consensus + mining
│   ├── store_bolt.go   # BoltDB persistence
│   └── ...
├── ai-auditor/         # Python FastAPI AI policy checker
├── docs/               # Specification & docs
├── scripts/            # Test & deployment scripts
└── docker-compose.yml  # Single-node setup
```

## Documentation

- [SPEC.md](./docs/SPEC.md) — Protocol specification
- [ROADMAP.md](./docs/ROADMAP.md) — Development phases
- [SECURITY.md](./docs/SECURITY.md) — Security policy

## Tokenomics

- **Supply**: 21,000,000 coins (8 decimal places)
- **Block Reward**: Starts at 50 coins, halves every 210,000 blocks
- **Fees**: 100% to miner
- **Premine**: None. Pure PoW start.

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md) for how to report bugs and submit PRs.

## License

MIT — see [LICENSE](./LICENSE)
