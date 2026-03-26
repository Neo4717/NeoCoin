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

## Bug Bounty Program

We have an active bug bounty program. Rewards based on severity:

| Severity | Reward Range |
|----------|--------------|
| Critical | $5,000 - $50,000 |
| High | $1,000 - $5,000 |
| Medium | $250 - $1,000 |
| Low | $50 - $250 |

To report, contact: neo4717@atomicmail.io

**Scope:**
- Consensus bugs
- Signature validation issues
- Storage vulnerabilities
- P2P security
- RPC endpoint vulnerabilities

**Exclusions:**
- Social engineering
- DDoS
- Physical security
- Third-party integrations

## Dependencies

We use Go modules with verified dependencies. Run `go mod verify` to confirm integrity.
