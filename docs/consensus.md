# Consensus Mechanism

## Proof of Work (PoW)

NeoCoin uses a Proof of Work consensus algorithm based on SHA-256 hashing. The mining process involves finding a nonce value that produces a hash meeting the current difficulty target.

### Mining Algorithm

```
For each candidate nonce:
    hash = SHA256(block_header + nonce)
    if hash < target:
        block is valid
```

The block header includes:
- Version
- Previous block hash
- Merkle root
- Timestamp
- Difficulty target
- Nonce

## Difficulty Adjustment

NeoCoin implements an adaptive difficulty adjustment algorithm to maintain a consistent block time regardless of network hashrate variations.

### Algorithm

```go
targetBlockTime = 600 seconds (10 minutes)
adjustmentInterval = 2016 blocks (approx. 2 weeks)

newDifficulty = previousDifficulty * (actualTime / targetTime)
```

### Difficulty Calculation

```
actualTime = timestamp[lastBlock] - timestamp[firstBlock]
expectedTime = adjustmentInterval * targetBlockTime
difficultyRatio = actualTime / expectedTime

newDifficulty = clamp(previousDifficulty * difficultyRatio, 0.25, 4.0)
```

The difficulty adjustment is clamped to prevent extreme changes:
- Minimum adjustment: 0.25x (4x easier)
- Maximum adjustment: 4.0x (4x harder)

## Fork Resolution

When multiple valid chains exist, NeoCoin uses the **longest chain rule** to resolve forks.

### Rules

1. **Chain Weight**: The chain with the most accumulated work is preferred
2. **Block Time**: If equal weight, the chain with earlier timestamp wins
3. **Orphan Handling**: Orphaned blocks are kept in memory for potential reorg

### Reorganization

When a longer chain is discovered:
1. Validate all new blocks
2. Revert blocks on current chain (up to 100 blocks)
3. Apply new blocks
4. Update UTXO set

## Block Reward Schedule

The block reward follows a decay schedule similar to Bitcoin:

### Halving Schedule

| Era | Blocks | Reward (NEO) |
|-----|--------|--------------|
| 1   | 0-210,000 | 50 |
| 2   | 210,001-420,000 | 25 |
| 3   | 420,001-630,000 | 12.5 |
| ... | ... | ... |

### Subsidy Calculation

```go
halvings = blockHeight / 210000
reward = 50 >> halvings
```

The reward halves approximately every 210,000 blocks (~4 years).

### Coinbase Maturity

Coinbase transactions become spendable after 100 block confirmations.

## Transaction Selection

The mempool uses fee-based transaction selection:

1. Sort by fee per byte (descending)
2. Select transactions up to block size limit
3. Prioritize higher fee transactions

### Fee Market

- Minimum fee: 0.01 NEO
- Recommended fee: 0.1 NEO/kB
- Fees go to the miner who includes the transaction