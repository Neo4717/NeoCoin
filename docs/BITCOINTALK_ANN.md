# NeoCoin (NEO) - ANN Thread

## ⛏️ PROOF-OF-WORK BLOCKCHAIN IS NOW LIVE ⛏️

---

### WARNING - READ BEFORE CONTINUING

**This is experimental software.** NeoCoin is a functioning PoW blockchain but has **NO real economic value**. There is no mainnet with token exchanges. Do NOT invest real money. This is an educational/research project.

---

## What is NeoCoin?

NeoCoin is a **Layer-1 Proof-of-Work blockchain** written in Go. It implements:

- ✅ **21M max supply** (Bitcoin-style)
- ✅ **Halving every 210,000 blocks** 
- ✅ **Ed25519 cryptography** (fast & secure)
- ✅ **Working Web Wallet** - /wallet/
- ✅ **Block Explorer** - /explorer/
- ✅ **HTTP + WebSocket API**
- ✅ **Optional AI Transaction Auditor** (policy check only, NOT consensus)
- ✅ **Docker deployment**

---

## Quick Start

```bash
# Clone
git clone https://github.com/Neo4717/NeoCoin.git
cd NeoCoin

# Run node
docker compose -f docker-compose.public.yml up -d

# Open wallet
# Visit http://localhost:8080/wallet/
```

---

## Features

### Web Wallet
- Create new wallet
- Import existing private key
- Check balance
- Send transactions

### Block Explorer
- View blocks
- Search transactions
- View chain stats
- Real-time updates via WebSocket

### Mining
```bash
export MINER_ADDRESS=your_wallet_address
export AUTO_MINE=true
docker compose -f docker-compose.public.yml up -d
```

---

## Tokenomics

| Parameter | Value |
|-----------|-------|
| Max Supply | 21,000,000 NEO |
| Initial Reward | 50 NEO/block |
| Halving | Every 210,000 blocks |
| Premine | **NONE** |
| Consensus | Proof-of-Work |

---

## Important Disclaimers

1. **NO MAINNET VALUE** - Tokens have NO real monetary value
2. **NO TOKEN SALE** - We will never sell tokens
3. **P2P is unauthenticated** - Only connect to trusted peers
4. **AI Auditor is optional** - NOT part of consensus
5. **Use at your own risk** - No warranties

---

## Links

- **Repository**: https://github.com/Neo4717/NeoCoin
- **Web Wallet**: http://localhost:8080/wallet/
- **Explorer**: http://localhost:8080/explorer/
- **License**: MIT

---

## Status

- ✅ Blockchain consensus works
- ✅ Mining functional
- ✅ Transaction signing/sending works
- ✅ Web wallet UI
- ✅ Block explorer
- ⚠️ P2P networking (experimental)
- ⚠️ No real value

---

**DONATIONS**: Not accepted. This is a free, open-source project.

---

*Last updated: March 2026*
