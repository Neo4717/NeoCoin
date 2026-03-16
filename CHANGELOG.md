# Changelog

All notable changes to this project will be documented in this file.

## [1.0.0] - 2026-03-17

### Added
- Protocol spec v1.0 stable
- Reproducible builds (Makefile + Dockerfile.reproducible)
- Version/buildTime info in /chain/info API
- Secure P2P (TLS + Ed25519 authentication)
- Fuzz testing support

### Updated
- All documentation reflects production-ready status
- Removed "prototype" references throughout

## [0.1.0] - 2026-03-13

### Added
- PoW blockchain with Ed25519 transactions
- BoltDB persistence with DAG support
- Merkle proofs (v2 blocks)
- HTTP API + WebSocket events
- Address transaction indexing
- Encrypted keystore support
- Docker Compose setup (single-node, mainnet, testnet, smoke)
- AI Auditor (policy check only - NOT consensus)
- Agent-driven backlog workflow

### Fixed
- Auto-miner now supports empty block mining (MINE_FORCE_EMPTY_BLOCKS=true)
- docker-compose defaults for HTTP_ADDR, ADMIN_TOKEN