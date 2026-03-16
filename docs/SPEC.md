# Neocoin protocol (v1.0)

Status: **v1.0 — stable**. This document describes the current on-disk and over-the-wire behaviors implemented by the node in this repository.

## Chain identity

- `CHAIN_ID` (uint64) defines the network.
- Nodes only accept blocks/transactions for their configured `CHAIN_ID`.

## Transaction

Model: account-based.

Key points:

- Signatures: Ed25519
- Transaction ID: `txid` is `hex(SHA256(signing_preimage))` under the active consensus encoding.
- Base validation: signature verification + encoding checks + `chainId` match
- State validation: account nonce + balance checks (including pending mempool txs)

## Consensus encoding (binary v1)

This project originally used JSON serialization for tx signing and PoW header hashing. For cross-language compatibility, a strict binary encoding can be enabled:

- `BINARY_ENCODING_ENABLE=true|false`
- `BINARY_ENCODING_ACTIVATION_HEIGHT=<uint64>`

When enabled, blocks at heights `>= BINARY_ENCODING_ACTIVATION_HEIGHT` must use the binary encoding described below for:

- transaction signing hash / txid
- PoW header hashing

### Integers and bytes

- `u8`, `u32`, `u64` are **little-endian**
- `int64` is **little-endian two's complement**
- variable-length bytes are prefixed with **ULEB128** length

### Transaction signing preimage (binary v1)

Prefix:

1. `encodingVersion` (`u8`) = `1`
2. `txType` (`u8`) = `0` (coinbase) or `1` (transfer)
3. `chainId` (`u64`)

Then:

- Coinbase (`txType=0`):
  1. `toAddress` (32 bytes; decoded from 64-hex string)
  2. `amount` (`u64`)
  3. `dataLen` (ULEB128) + `data` (bytes)

- Transfer (`txType=1`):
  1. `fromPubKey` (32 bytes Ed25519 public key)
  2. `toAddress` (32 bytes; decoded from 64-hex string)
  3. `amount` (`u64`)
  4. `nonce` (`u64`)
  5. `fee` (`u64`)
  6. `dataLen` (ULEB128) + `data` (bytes)

Signing hash (and txid):

- `txSigningHash = SHA256(signingPreimage)`
- `txidHex = hex(txSigningHash)`

### Block header preimage (binary v1)

1. `encodingVersion` (`u8`) = `1`
2. `blockVersion` (`u32`) = `Block.Version`
3. `height` (`u64`)
4. `timestampUnix` (`int64`)
5. `prevHash` (32 bytes; zeroes for genesis)
6. `commitmentRoot` (32 bytes)
   - `Block.Version=1`: `TxRootLegacy = SHA256(concat(txSigningHash))`
   - `Block.Version=2`: Merkle root of tx signing hashes (domain-separated; see `MerkleRoot`)
7. `difficultyBits` (`u32`)
8. `minerAddress` (32 bytes; decoded from 64-hex string)
9. `nonce` (`u64`)

Block hash / PoW digest:

- `blockHash = SHA256(blockHeaderPreimage)`

### Test vectors

These are enforced by `blockchain/encoding_vectors_test.go`.

**Tx (transfer)**

- signing preimage hex:

```
010101000000000000008a88e3dd7409f195fd52db2d3cba5d72ca6709bf1d94121bf3748801b40f6f5c6a3803d5f059902a1c6dafbc9ba4729212f7caac08634cc3ae76b27529f038270a000000000000000100000000000000010000000000000000
```

- signing hash / txid hex:

```
9c0a2eeef8e708c919c94beee4f23c48498a6abc0653ccf9507abc6c41a8e5d9
```

**Block header (nonce=0)**

- header preimage hex:

```
0101000000010000000000000001f153650000000011111111111111111111111111111111111111111111111111111111111111113f8441fd88b8e5611e1cc4f5e23e52cbbe5c7c3e92e52788f5ae55b35f7686cc12000000b62e867fa2f33afe62d5d6b1642e1621d543307846b2a57b897e710919b767090000000000000000
```

- header hash hex:

```
14cf82e2b30bdc7ff59998f48f0ee194e787d3a9a6b5c924aee61a815579291b
```

## Blocks

Proof of work:

- Blocks include a `DifficultyBits` value.
- Fork choice: **highest cumulative work**, not highest height.

Merkle commitments:

- `Version=1`: legacy tx root
- `Version=2`: Merkle root (optional feature flag + activation height)

## Networking

### HTTP RPC (operator / client)

The node exposes an HTTP API on `HTTP_ADDR` (default `:8080`).

Admin endpoints are disabled unless `ADMIN_TOKEN` is set.

`GET /chain/info` includes:

- `rulesHash`: a 32-byte hex hash of the node's consensus-critical parameters; nodes with different rules must not peer.

### TCP P2P (peer sync; draft)

The node can optionally listen for TCP peers on `P2P_LISTEN_ADDR` (default `:9090`) when `P2P_ENABLE=true`.

Message framing:

- 4-byte big-endian length prefix
- JSON payload of an envelope `{ "type": "...", "payload": ... }`

Handshake:

- client sends `type="hello"` with `{protocol, chainId, rulesHash, nodeId, timeUnix}`
- server replies with `type="hello"` and the same fields
- if `chainId`, `rulesHash`, or `protocol` mismatch, the server rejects

Requests (1 per connection):

- `chain_info_req` → `chain_info`
- `headers_from_req` → `headers`
- `block_by_hash_req` → `block` or `not_found`

This P2P protocol is currently unauthenticated and unencrypted and is intended for test networks only.
