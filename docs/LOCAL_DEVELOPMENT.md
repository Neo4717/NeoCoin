# Local Development Guide

## Prerequisites

- Go 1.23+
- Docker & Docker Compose
- Git

## Quick Start

### 1. Clone and Build

```bash
git clone https://github.com/Neo4717/NeoCoin.git
cd NeoCoin
go mod download
go build -o neocoin ./cmd/node/
```

### 2. Run a Node

```bash
# Default testnet
./neocoin

# Or with Docker
docker compose -f docker-compose.testnet.yml up
```

### 3. Verify It's Running

```bash
curl http://127.0.0.1:8080/chain/info
curl http://127.0.0.1:8080/metrics
```

## Development Workflow

### Run Tests

```bash
# Unit tests
go test ./...

# With race detector
go test -race ./...

# Fuzzing (30 seconds)
go test -fuzz=Fuzz -fuzztime=30s ./internal/consensus/
```

### Run Integration Tests

```bash
export RUN_INTEGRATION=1
chmod +x tests/integration/test.sh
./tests/integration/test.sh
```

### Build Docker Image

```bash
docker build -f docker/Dockerfile -t neocoin:local .
docker run -p 8080:8080 neocoin:local
```

### Multi-Node Testnet

```bash
# Start 3 nodes
docker compose -f docker-compose.testnet.yml up -d node1 node2 node3

# View logs
docker compose -f docker-compose.testnet.yml logs -f

# Stop
docker compose -f docker-compose.testnet.yml down -v
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| CHAIN_ID | 1 | Chain ID (1=mainnet) |
| NODE_PORT | 8080 | HTTP API port |
| P2P_PORT | 9090 | P2P networking port |
| P2P_ENABLE | false | Enable P2P networking |
| AUTO_MINE | false | Enable auto mining |
| MINER_ADDRESS | - | Mining reward address |
| STORE_MODE | pruned | Storage mode (full/pruned/archive) |
| LOG_LEVEL | info | Log level (debug/info/warn/error) |
| ENABLE_PROTOBUF | true | Use protobuf encoding |
| METRICS_ENABLED | true | Enable /metrics endpoint |

### Example: Mining Node

```bash
export CHAIN_ID=1
export NODE_PORT=8080
export P2P_ENABLE=true
export P2P_PORT=9090
export MINER_ADDRESS=NEOxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
export AUTO_MINE=true
export LOG_LEVEL=debug
./neocoin
```

## API Reference

### Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| /chain/info | GET | Chain information |
| /balance/{addr} | GET | Account balance |
| /tx | POST | Submit transaction |
| /tx/{txid} | GET | Get transaction |
| /block/height/{n} | GET | Get block by height |
| /mempool | GET | Mempool contents |
| /wallet/create | POST | Create wallet |
| /wallet/sign | POST | Sign transaction |
| /peers | GET | Peer list |
| /metrics | GET | Prometheus metrics |

### Example: Create Wallet and Send Transaction

```bash
# Create wallet
WALLET=$(curl -s -X POST http://127.0.0.1:8080/wallet/create)
ADDR=$(echo $WALLET | jq -r '.address')
PRIVKEY=$(echo $WALLET | jq -r '.privateKey')
echo "Address: $ADDR"

# Check balance
curl http://127.0.0.1:8080/balance/$ADDR

# Send transaction
curl -X POST http://127.0.0.1:8080/tx \
  -H "Content-Type: application/json" \
  -d "{\"privateKey\":\"$PRIVKEY\",\"to\":\"RECEIVER_ADDR\",\"amount\":1000}"
```

## Metrics

Prometheus metrics available at `/metrics`:

```bash
# View metrics
curl http://127.0.0.1:8080/metrics

# With Prometheus
prometheus --config.file=prometheus.yml
```

Key metrics:
- `neocoin_chain_height` - Current block height
- `neocoin_mempool_size` - Transactions in mempool
- `neocoin_peer_count` - Connected peers
- `neocoin_mining_hashrate` - Current hashrate
- `neocoin_http_request_duration_seconds` - API latency

## Troubleshooting

### Node Won't Start

```bash
# Check logs
./neocoin 2>&1

# Reset data
rm -rf data/
./neocoin
```

### P2P Not Connecting

```bash
# Enable debug logging
export LOG_LEVEL=debug
./neocoin

# Check firewall
curl http://127.0.0.1:9090 2>&1
```

### Build Errors

```bash
# Clean build cache
go clean -cache

# Re-download deps
go mod download
go mod tidy

# Rebuild
go build -o neocoin ./cmd/node/
```

## Architecture

```
cmd/node/          - Entry point
internal/blockchain/ - Core chain logic
internal/consensus/  - PoW, difficulty
internal/crypto/     - Ed25519, AES
internal/mempool/    - TX pool
internal/mining/     - Miner + Stratum
internal/networking/ - P2P
internal/storage/    - BoltDB
internal/cache/      - LRU cache
internal/metrics/    - Prometheus
api/http/            - REST API
api/websocket/       - WebSocket
config/              - Configuration
```
