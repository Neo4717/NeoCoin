# Changelog

All notable changes to this project will be documented in this file.

## [0.1.0] - 2026-03-13

### Added
- PoW blockchain with Ed25519 transactions
- BoltDB persistence with DAG support
- Merkle proofs (v2 blocks)
- HTTP API + WebSocket events
- Address transaction indexing
- Encrypted keystore support
- Docker Compose setup (single-node, testnet, smoke)
- AI Auditor (policy check only - NOT consensus)
- Agent-driven backlog workflow

### Known Limitations
- P2P is unauthenticated (test networks only)
- No security audits
- AI Auditor is non-deterministic (not part of consensus)
- No reproducible builds yet

### Fixed
- Auto-miner now supports empty block mining (MINE_FORCE_EMPTY_BLOCKS=true)
- docker-compose defaults for HTTP_ADDR, ADMIN_TOKEN
