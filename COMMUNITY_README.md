# NeoCoin - Decentralized Payment Network

**Status: Mainnet Ready**

A production-ready Proof-of-Work blockchain with AI-powered security for real payments.

## Why NeoCoin?

- **Fast**: 1-minute block times
- **Secure**: AI-powered transaction auditing
- **Fair**: No premine, no VC allocation
- **Open**: 100% open source
- **Community-owned**: Governed by token holders

## Quick Start

```bash
# Clone
git clone https://github.com/Neo4717/NeoCoin.git
cd NeoCoin

# Run mainnet
docker compose -f docker-compose.mainnet.yml up -d

# Check status
curl http://localhost:8080/chain/info
```

## Tokenomics

| Total Supply | 21,000,000 NEO |
| Block Reward | 50 NEO (halving every 210K blocks) |
| Mining | GPU/ASIC compatible |

## Community

Join the movement:
- GitHub: https://github.com/Neo4717/NeoCoin
- Issues: Report bugs, request features

## For Developers

- [API Documentation](./docs/API.md)
- [SDKs](./sdk/)
- [Smart Contracts](./docs/CONTRACTS.md)

## Security

- Rate limiting enabled
- Admin token required for admin functions
- AI auditor for fraud detection
- Audited by community

## License

MIT