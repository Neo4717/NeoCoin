# NeoCoin API Documentation

## Overview

NeoCoin provides a RESTful HTTP API and WebSocket interface for interacting with the blockchain.

## Base URL

```
http://localhost:8080
```

## Chain Information

### Get Chain Info

```
GET /chain/info
```

**Response:**
```json
{
  "chainId": 1,
  "height": 12345,
  "latestHash": "abc123...",
  "genesisHash": "def456...",
  "difficulty": 20,
  "peersCount": 3
}
```

## Wallets

### Create Wallet

```
POST /wallet/create
```

**Response:**
```json
{
  "address": "NEO00...",
  "privateKey": "base64...",
  "publicKey": "base64..."
}
```

### Sign Transaction

```
POST /wallet/sign
```

**Request:**
```json
{
  "privateKey": "base64...",
  "toAddress": "NEO00...",
  "amount": 100,
  "fee": 1,
  "nonce": 1,
  "data": ""
}
```

**Response:**
```json
{
  "txJson": "{...}",
  "txId": "abc123..."
}
```

## Transactions

### Submit Transaction

```
POST /tx
```

**Request:**
```json
{
  "type": "transfer",
  "chainId": 1,
  "fromPubKey": "base64...",
  "toAddress": "NEO00...",
  "amount": 100,
  "fee": 1,
  "nonce": 1,
  "signature": "base64..."
}
```

**Response:**
```json
{
  "accepted": true,
  "txId": "abc123..."
}
```

### Get Transaction

```
GET /tx/{txid}
```

### Get Balance

```
GET /balance/{address}
```

**Response:**
```json
{
  "address": "NEO00...",
  "balance": 1000,
  "nonce": 5
}
```

## Blocks

### Get Block by Height

```
GET /block/height/{height}
```

### Get Block by Hash

```
GET /block/hash/{hash}
```

## Mempool

### Get Mempool

```
GET /mempool
```

**Response:**
```json
{
  "size": 10,
  "transactions": [...]
}
```

## WebSocket

### Connect

```
ws://localhost:8080/ws
```

### Subscribe

```json
{
  "type": "subscribe",
  "topic": "all"
}
```

```json
{
  "type": "subscribe", 
  "topic": "address",
  "address": "NEO00..."
}
```

### Events

- `new_block` - New block mined
- `mempool_added` - Transaction added to mempool
- `mempool_removed` - Transaction removed from mempool

## Light Client (SPV)

### Sync Address

```
POST /spv/sync/{address}
```

### Get Balance (SPV)

```
GET /spv/balance/{address}
```

### Get Transaction Proof

```
GET /spv/proof/{txHash}/{blockHash}
```

## AI Features

### Get Network Anomalies

```
GET /ai/anomalies
```

**Response:**
```json
{
  "anomalies": [
    {
      "type": "timing",
      "severity": "high",
      "metric": "block_time_variance",
      "value": 5000,
      "description": "High variance in block times"
    }
  ]
}
```

### Get Fee Recommendation

```
GET /ai/fee?confirmations=3
```

**Response:**
```json
{
  "fee_rate": 10,
  "confidence": "high",
  "wait_blocks": 3
}
```

### Get Node Health

```
GET /ai/health
```

**Response:**
```json
{
  "node_id": "local",
  "status": "healthy",
  "cpu": 45.2,
  "memory": 62.1,
  "latency": 50,
  "peers": 5
}
```

## Smart Contracts

### Deploy Contract

```
POST /contract/deploy
```

**Request:**
```json
{
  "code": "hex...",
  "gasLimit": 10000
}
```

### Call Contract

```
POST /contract/call
```

**Request:**
```json
{
  "contract": "NEO00...",
  "method": "transfer",
  "params": ["to", "amount"],
  "gasLimit": 5000
}
```

## Rate Limits

Default: 100 requests/second, burst 20

## Error Codes

| Code | Description |
|------|-------------|
| 400 | Bad Request |
| 401 | Unauthorized |
| 403 | Forbidden |
| 404 | Not Found |
| 429 | Rate Limited |
| 500 | Internal Error |
