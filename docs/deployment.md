# Deployment Guide

## Docker Deployment

### Quick Start

```bash
# Clone the repository
git clone https://github.com/neocoin/neocoin.git
cd neocoin

# Start the node
docker compose -f docker/docker-compose.yml up -d
```

### Production Deployment

```bash
# Build the image
docker build -f docker/Dockerfile -t neocoin:latest .

# Run with environment variables
docker run -d \
  --name neocoin \
  -p 8080:8080 \
  -p 3030:3030 \
  -e CHAIN_ID=1 \
  -e GENESIS_PATH=genesis/mainnet.json \
  -e MINER_ADDRESS=your_address \
  -v neocoin_data:/data \
  neocoin:latest
```

## Configuration Reference

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CHAIN_ID` | 1 | Chain identifier (1=mainnet, 2=testnet) |
| `GENESIS_PATH` | genesis/mainnet.json | Path to genesis file |
| `HTTP_ADDR` | 127.0.0.1:8080 | HTTP server address |
| `P2P_LISTEN_ADDR` | :3030 | P2P listen address |
| `P2P_ENABLE` | false | Enable P2P networking |
| `P2P_PEERS` | | Comma-separated peer addresses |
| `MINER_ADDRESS` | | Miner reward address |
| `AUTO_MINE` | false | Enable automatic mining |
| `MINE_INTERVAL_MS` | 1000 | Mining interval in milliseconds |
| `ADMIN_TOKEN` | | Admin API token |
| `RATE_LIMIT_REQUESTS` | 100 | Rate limit requests per second |
| `RATE_LIMIT_BURST` | 20 | Rate limit burst |
| `WS_ENABLE` | true | Enable WebSocket |
| `WS_MAX_CONNECTIONS` | 100 | Max WebSocket connections |

### Configuration Files

#### .env file

Create a `.env` file:

```bash
CHAIN_ID=1
GENESIS_PATH=genesis/mainnet.json
MINER_ADDRESS=your_miner_address
ADMIN_TOKEN=your_admin_token
AUTO_MINE=false
P2P_ENABLE=true
P2P_PEERS=seed1.neocoin.io:3030,seed2.neocoin.io:3030
```

## Security Considerations

### Network Security

1. **Bind to localhost**: By default, HTTP API binds to 127.0.0.1
2. **Use firewall**: Restrict access to P2P port (3030)
3. **TLS**: Configure reverse proxy with TLS for production

### Key Management

1. **Private keys**: Store wallet private keys securely
2. **Admin token**: Use strong random tokens for admin API
3. **Mining rewards**: Secure your miner address private keys

### Resource Limits

1. **Rate limiting**: Enable rate limiting in production
2. **Connection limits**: Limit WebSocket connections
3. **Memory**: Monitor memory usage for large mempool

## Docker Compose Variants

### Mainnet

```bash
docker compose -f docker-compose.mainnet.yml up -d
```

### Testnet

```bash
docker compose -f docker-compose.testnet.yml up -d
```

### Development

```bash
docker compose -f docker/docker-compose.dev.yml up -d
```

## Monitoring

### Health Check

```bash
curl http://localhost:8080/chain/info
```

### Logs

```bash
# Docker logs
docker logs neocoin

# Follow logs
docker logs -f neocoin
```

### Metrics

Access metrics at `/metrics` endpoint (if enabled).

## Backup and Recovery

### Data Directory

The data directory contains:
- Blockchain data
- Wallet data
- Mempool

### Backup

```bash
# Backup data volume
docker run --rm -v neocoin_data:/data -v $(pwd):/backup alpine tar czf /backup/neocoin_backup.tar.gz /data
```

### Restore

```bash
# Restore from backup
docker run --rm -v neocoin_data:/data -v $(pwd):/backup alpine tar xzf /backup/neocoin_backup.tar.gz -C /
```