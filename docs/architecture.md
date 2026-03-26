# Architecture

## Overview

NeoCoin is a cryptocurrency blockchain implementation written in Go. The system follows a modular architecture with clear separation of concerns.

## Component Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                        API Layer                            │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │
│  │  REST API   │  │  WebSocket  │  │   JSON-RPC Server    │ │
│  └──────┬──────┘  └──────┬──────┘  └──────────┬──────────┘ │
└─────────┼────────────────┼─────────────────────┼────────────┘
          │                │                     │
┌─────────┴────────────────┴─────────────────────┴────────────┐
│                     Service Layer                            │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │   Wallet    │  │ Transaction │  │    Block     │        │
│  │  Service    │  │   Service   │  │   Service    │        │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘        │
└─────────┼────────────────┼────────────────┼────────────────┘
          │                │                │
┌─────────┴────────────────┴────────────────┴────────────────┐
│                      Core Layer                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                   Blockchain                         │   │
│  │  ┌───────────┐  ┌───────────┐  ┌───────────┐      │   │
│  │  │   UTXO    │  │  Mempool  │  │  Merkle   │      │   │
│  │  │  Database │  │  Manager  │  │   Tree    │      │   │
│  │  └───────────┘  └───────────┘  └───────────┘      │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
          │                │
┌─────────┴────────────────┴────────────────────────────────┐
│                  Consensus Layer                            │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │
│  │    PoW     │  │  Difficulty │  │   Fork Resolution   │ │
│  │  Miner     │  │  Adjuster   │  │      Engine         │ │
│  └─────────────┘  └─────────────┘  └─────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
          │
┌─────────┴────────────────────────────────────────────────┐
│                   Network Layer                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │
│  │ P2P Server  │  │  Protocol   │  │   Block Propagation  │ │
│  │             │  │   Handler   │  │       (Gossip)      │ │
│  └─────────────┘  └─────────────┘  └─────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

## Module Relationships

### API Layer
- **REST API**: Handles HTTP requests for wallet, transaction, and chain operations
- **WebSocket**: Real-time block and transaction notifications
- **JSON-RPC**: Alternative RPC interface for programmatic access

### Service Layer
- **Wallet Service**: Key management, address generation, balance tracking
- **Transaction Service**: Transaction creation, validation, signing
- **Block Service**: Block retrieval, submission, chain information

### Core Layer
- **Blockchain**: Main chain state, block validation
- **UTXO Database**: Unspent transaction outputs
- **Mempool**: Unconfirmed transactions
- **Merkle Tree**: Transaction merkle root calculation

### Consensus Layer
- **PoW Miner**: Block mining with SHA-256
- **Difficulty Adjuster**: Dynamic difficulty based on block time
- **Fork Resolution**: Longest chain selection

### Network Layer
- **P2P Server**: Peer-to-peer connections
- **Protocol Handler**: Message serialization/deserialization
- **Block Propagation**: Gossip protocol for block distribution

## Data Flow

1. **Transaction Submission**:
   ```
   Client → REST API → Transaction Service → Mempool → P2P Network
   ```

2. **Block Mining**:
   ```
   Miner → PoW Algorithm → New Block → Blockchain Validation → Add to Chain → P2P Network
   ```

3. **Block Synchronization**:
   ```
   P2P Network → Protocol Handler → Block Validation → Blockchain → UTXO Update
   ```

## Configuration

Key configuration files:
- `config/`: Configuration definitions
- `.env`: Environment variables
- `genesis/`: Genesis block definitions