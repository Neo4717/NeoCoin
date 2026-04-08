# Networking

## P2P Protocol

NeoCoin uses a peer-to-peer network for block and transaction propagation. Nodes communicate over TCP connections.

### Connection Setup

```
Node A                    Node B
   │                         │
   │──── Version ────────────▶│
   │                         │
   │◀─── Version ────────────│
   │                         │
   │──── Verack ────────────▶│
   │                         │
   │◀─── Verack ────────────│
   │                         │
   │──── addr ──────────────▶│ (peer list exchange)
   │◀─── addr ──────────────│
```

## Handshake Procedure

### Version Message

The version message is the first message exchanged between peers:

```json
{
  "version": 70015,
  "services": 1,
  "timestamp": 1234567890,
  "addr_recv": "ip:port",
  "addr_from": "ip:port",
  "nonce": 12345678,
  "user_agent": "/neo:1.0.0/",
  "start_height": 12345,
  "relay": true
}
```

### Verack

After receiving and validating the version message, nodes exchange verack messages to complete the handshake.

## Message Types

### Inventory Messages

| Message | Description |
|---------|-------------|
| `inv` | Advertise available blocks/transactions |
| `getdata` | Request specific blocks/transactions |
| `tx` | Transaction message |
| `block` | Block message |

### Block Sync

| Message | Description |
|---------|-------------|
| `getblocks` | Request block inventory range |
| `getheaders` | Request block headers |
| `headers` | Block headers response |
| `notfound` | Requested item not found |

### Network Messages

| Message | Description |
|---------|-------------|
| `ping` | Keep-alive ping |
| `pong` | Keep-alive pong |
| `addr` | Peer address list |
| `reject` | Message rejection |

## Block Propagation

NeoCoin uses a gossip protocol for block propagation:

### Propagation Flow

```
Miner finds block
    │
    ▼
Broadcast to all connected peers
    │
    ├──▶ Peer A validates block
    │        │
    │        ▼
    │    If valid: store and relay
    │    If invalid: reject
    │
    ├──▶ Peer B validates block
    │        │
    │        ▼
    │    If valid: store and relay
    │
    └──▶ Peer C validates block
             │
             ▼
         If valid: store and relay
```

### Relay Policy

1. Validate block structure
2. Validate proof of work
3. Validate transactions
4. Validate previous block reference
5. If valid: relay to other peers

### Compact Blocks

For bandwidth efficiency, compact block messages are supported:
- Partial block with short transaction IDs
- Request missing transactions by ID

## Transaction Gossip

Transactions are gossiped using a similar protocol:

### Mempool Exchange

On connection, nodes exchange their mempool contents:
1. Send `mempool` message
2. Response contains transaction inventory
3. Request missing transactions

### Anti-DoS

- Rate limiting on inv messages
- Validation before relay
- Small inventory message limits

## Peer Discovery

### DNS Seeds

Bootstrap nodes are obtained from DNS seeds:
- `seed.neocoin.io`

### Manual Peers

Peers can be manually specified via:
- `-peers` command line flag
- Configuration file
- API endpoint `/p2p/peers`

### Address Advertisement

Nodes advertise their address to connected peers, which is propagated through the network.

## Tor Onion Service

NeoCoin can be exposed as a Tor hidden service for anonymous P2P connections.

### Setup with Docker

```bash
# Start with Tor
docker-compose -f docker-compose.tor.yml up -d

# Get your .onion address
./scripts/tor-onion.sh
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `P2P_LISTEN_ADDR` | P2P listen address (default `:9090`) |
| `HTTP_ADDR` | HTTP API address (default `127.0.0.1:8080`) |

### Connecting to Onion Nodes

To connect to another node's .onion address:

```bash
P2P_PEERS=abcd1234567890.onion:9090 ./neocoin server
```

### Onion Address Format

- HTTP API: `http://<onion>.onion`
- P2P: `<onion>.onion:9090`

The Tor hidden service automatically generates a persistent address stored in Docker volume `tor_keys`. Backup this volume to keep your address.