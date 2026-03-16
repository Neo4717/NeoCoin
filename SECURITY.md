# Security Policy

## Reporting Security Issues

Please do NOT open public issues for security vulnerabilities.

Contact: security@neocoin.dev

## Security Model

### Consensus Security
- PoW consensus with SHA-256-like hashing
- Ed25519 signature verification
- Cumulative work fork choice
- Difficulty adjustment (EMA)
- MTP (Median Time Past) timestamp validation

### Known Limitations
- P2P is currently unauthenticated (intended for test networks)
- AI Auditor is non-deterministic and NOT part of consensus

### Implemented Defenses
- Admin token required for admin endpoints
- Rate limiting enabled
- Encrypted keystore (AES-256-GCM)
- Binary encoding for deterministic transaction signing

## Bug Bounty

We are establishing a bug bounty program. Contact security@neocoin.dev for details.

## Dependencies

We use Go modules with verified dependencies. Run `go mod verify` to confirm integrity.