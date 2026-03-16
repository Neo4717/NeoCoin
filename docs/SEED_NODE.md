# Seed Node Deployment Guide

This guide explains how to deploy a public seed node for the NeoCoin testnet.

## What is a Seed Node?

A seed node is a publicly accessible node that new nodes connect to for initial peer discovery. Seed nodes help bootstrap the P2P network.

## Prerequisites

- A VPS with at least 2GB RAM, 20GB SSD
- Ubuntu 20.04+ or similar Linux distribution
- Docker and Docker Compose installed
- Static IP or DNS hostname

## Quick Start

### 1. Clone the repository

```bash
git clone https://github.com/Neo4717/NeoCoin.git
cd NeoCoin
```

### 2. Configure environment

Create a `.env` file:

```bash
cp .env.example .env
```

Edit `.env` and set:

```bash
# Chain configuration
CHAIN_ID=2
GENESIS_PATH=genesis/testnet.json

# Node identity
NODE_ID=seed-node-01
MINER_ADDRESS=YOUR_TESTNET_ADDRESS

# P2P configuration
P2P_ENABLE=true
P2P_LISTEN_ADDR=0.0.0.0:9090

# Mining (optional - seed nodes can mine to generate new coins for the network)
AUTO_MINE=true
MINE_INTERVAL_MS=5000

# API
HTTP_ADDR=0.0.0.0:8080
WS_ENABLE=true
```

### 3. Start the node

```bash
docker compose -f docker-compose.seed.yml up -d
```

### 4. Verify

```bash
curl http://localhost:8080/chain/info
```

You should see the chain height and peer count.

### 5. Register as a seed node

Announce your node's IP/DNS in the community:
- Discord: https://discord.gg/neocoin
- Telegram: https://t.me/neocoinofficial

## Docker Compose for Seed Node

Create `docker-compose.seed.yml`:

```yaml
services:
  blockchain:
    image: neocoin-blockchain:latest
    build:
      context: ./blockchain
      dockerfile: Dockerfile
    user: "0:0"
    ports:
      - "8080:8080"
      - "9090:9090"
    volumes:
      - neocoin_seed_data:/app/data
      - ./genesis:/app/genesis:ro
    environment:
      - CHAIN_ID=2
      - GENESIS_PATH=genesis/testnet.json
      - NODE_ID=seed-01
      - MINER_ADDRESS=${MINER_ADDRESS}
      - P2P_ENABLE=true
      - P2P_LISTEN_ADDR=0.0.0.0:9090
      - AUTO_MINE=true
      - MINE_INTERVAL_MS=5000
      - HTTP_ADDR=0.0.0.0:8080
      - WS_ENABLE=true
      - SYNC_ENABLE=true
    restart: unless-stopped
    command: ["./blockchain", "server"]

volumes:
  neocoin_seed_data:
```

## Security Considerations

1. **Firewall**: Only open ports 8080 and 9090
2. **Rate limiting**: Enable rate limiting in production
3. **Monitor**: Set up monitoring for node health
4. **Updates**: Keep your node updated with the latest code

## P2P Authentication (Production)

For production deployments, enable P2P authentication to ensure nodes connect to trusted peers:

```bash
# Generate a node key (Ed25519)
NODE_PRIVATE_KEY=$(openssl genpkey -algorithm ED25519 2>/dev/null | base64)
```

Add to your environment:
```bash
P2P_AUTH_ENABLE=true
P2P_TRUSTED_PEERS=peer1_pubkey,peer2_pubkey
```

## Getting Help

- Issues: https://github.com/Neo4717/NeoCoin/issues
- Discord: https://discord.gg/neocoin
- Telegram: https://t.me/neocoinofficial
