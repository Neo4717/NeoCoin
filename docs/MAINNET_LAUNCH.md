# NeoCoin Mainnet Launch Guide

## Pre-Launch Checklist

- [x] Core blockchain code complete
- [x] Security audit (self-audit)
- [x] Testnet running smoothly
- [x] Documentation complete
- [ ] Genesis block configured
- [ ] Bootstrap nodes ready
- [ ] Community aware

## Mainnet Specification

| Parameter | Value |
|-----------|-------|
| Chain ID | 1 |
| Symbol | NEO |
| Decimals | 8 |
| Max Supply | 21,000,000 |
| Block Time | ~1 minute |
| Consensus | PoW |
| Initial Reward | 50 NEO |
| Halving | Every 210,000 blocks |

## Launch Commands

### 1. Generate Miner Wallet
```bash
docker compose -f docker-compose.mainnet.yml run --rm blockchain ./blockchain create_wallet
```

### 2. Configure Environment
```bash
export ADMIN_TOKEN=$(openssl rand -hex 32)
export MINER_ADDRESS=<your_wallet_address>
export AUTO_MINE=true
```

### 3. Start Mainnet Node
```bash
docker compose -f docker-compose.mainnet.yml up -d
```

### 4. Verify Chain
```bash
curl http://localhost:8080/chain/info
```

## Network Endpoints

- **RPC**: http://localhost:8080
- **P2P**: localhost:9090
- **WebSocket**: ws://localhost:8080/ws

## Bootstrap Nodes

Add these to your config:
```
node1.neocoin.dev:9090
node2.neocoin.dev:9090
```

## Monitoring

```bash
# Check logs
docker compose -f docker-compose.mainnet.yml logs -f

# Check health
curl http://localhost:8080/health

# Check metrics
curl http://localhost:8080/metrics
```

## Post-Launch

1. Announce on GitHub
2. Share bootstrap node IPs
3. Monitor network health
4. Community engagement
5. List on block explorers