# NeoCoin Documentation

## Quick Links

- [Getting Started](./getting-started.md)
- [API Reference](./API.md)
- [Tokenomics](./TOKENOMICS.md)
- [Security](./SECURITY.md)
- [Roadmap](./ROADMAP.md)

## Version

Current version: **v1.0.0**

For more details, see [CHANGELOG.md](../CHANGELOG.md)

## Quick Start

```bash
# Clone the repository
git clone https://github.com/Neo4717/NeoCoin.git
cd NeoCoin

# Start a local node
docker compose -f docker-compose.public.yml up -d
```

### Check Chain Status

```bash
curl http://127.0.0.1:8080/chain/info
```

### Create Wallet

```bash
docker compose run --rm blockchain ./blockchain create_wallet
```

## Architecture

NeoCoin is a Layer-1 PoW blockchain with:

- **Consensus**: Proof-of-Work (SHA-256-like)
- **Signing**: Ed25519
- **Storage**: BoltDB
- **API**: REST + WebSocket
- **AI Auditor**: Optional policy check layer