# API Reference

## Base URL

```
http://localhost:8080
```

## Endpoints

### Chain Information

**GET** `/chain/info`

Get current blockchain information.

**Response:**
```json
{
  "chain_id": 1,
  "height": 12345,
  "hash": "0000000000000000000000000000000000000000000000000000000000000000",
  "difficulty": 1.234,
  "median_time": 1234567890,
  "reward": 50
}
```

### Block

**GET** `/block/{hash|height}`

Get block by hash or height.

**Response:**
```json
{
  "hash": "0000000000000000000000000000000000000000000000000000000000000000",
  "height": 12345,
  "version": 1,
  "prev_block": "0000000000000000000000000000000000000000000000000000000000000001",
  "merkle_root": "0000000000000000000000000000000000000000000000000000000000000000",
  "timestamp": 1234567890,
  "difficulty": 1.234,
  "nonce": 12345678,
  "tx": ["txid1", "txid2"]
}
```

**GET** `/block/header/{hash|height}`

Get block header only.

### Transaction

**GET** `/tx/{txid}`

Get transaction by ID.

**Response:**
```json
{
  "txid": "0000000000000000000000000000000000000000000000000000000000000000",
  "version": 1,
  "vin": [
    {
      "txid": "0000000000000000000000000000000000000000000000000000000000000001",
      "vout": 0,
      "script_sig": "..."
    }
  ],
  "vout": [
    {
      "value": 10.0,
      "script_pub_key": "..."
    }
  ],
  "lock_time": 0,
  "size": 250
}
```

**POST** `/tx/create`

Create and broadcast a new transaction.

**Request:**
```json
{
  "from": "NEO...",
  "to": "NEO...",
  "amount": 10.0,
  "fee": 0.01
}
```

**Response:**
```json
{
  "txid": "0000000000000000000000000000000000000000000000000000000000000000"
}
```

**GET** `/tx/mempool`

List transactions in the mempool.

### Wallet

**POST** `/wallet/create`

Create a new wallet.

**Response:**
```json
{
  "address": "NEO1234567890abcdef",
  "public_key": "02...",
  "private_key": "..."
}
```

**GET** `/wallet/balance?address={address}`

Get balance for an address.

**Response:**
```json
{
  "address": "NEO1234567890abcdef",
  "balance": 100.5,
  "unconfirmed": 0.0,
  "utxo_count": 5
}
```

**GET** `/wallet/utxo?address={address}`

List unspent outputs for an address.

**Response:**
```json
{
  "utxos": [
    {
      "txid": "0000000000000000000000000000000000000000000000000000000000000000",
      "vout": 0,
      "value": 50.0,
      "confirmations": 6
    }
  ]
}
```

### Mempool

**GET** `/mempool/info`

Get mempool statistics.

**Response:**
```json
{
  "size": 100,
  "bytes": 50000,
  "fee_min": 0.01,
  "fee_max": 1.0
}
```

### P2P

**GET** `/p2p/peers`

List connected peers.

**Response:**
```json
{
  "peers": [
    {
      "addr": "192.168.1.1:3030",
      "services": 1,
      "last_send": 1234567890,
      "last_recv": 1234567890,
      "version": 70015
    }
  ]
}
```

**POST** `/p2p/connect`

Connect to a new peer.

**Request:**
```json
{
  "address": "192.168.1.1:3030"
}
```

### Mining

**GET** `/mining/info`

Get mining information.

**Response:**
```json
{
  "status": "mining",
  "hashrate": 1000000,
  "difficulty": 1.234,
  "block_reward": 50,
  "total_blocks": 12345
}
```

## WebSocket Events

Connect to `ws://localhost:8080/ws`

### Event Types

**new_block**
```json
{
  "event": "new_block",
  "data": {
    "hash": "0000000000000000000000000000000000000000000000000000000000000000",
    "height": 12345
  }
}
```

**new_transaction**
```json
{
  "event": "new_transaction",
  "data": {
    "txid": "0000000000000000000000000000000000000000000000000000000000000000",
    "fee": 0.01
  }
}
```

**mempool_update**
```json
{
  "event": "mempool_update",
  "data": {
    "size": 100
  }
}
```

## Error Codes

| Code | Description |
|------|-------------|
| 400 | Bad Request |
| 404 | Not Found |
| 422 | Validation Error |
| 500 | Internal Server Error |
| 503 | Service Unavailable |

### Error Response

```json
{
  "error": {
    "code": 422,
    "message": "Invalid address format"
  }
}
```