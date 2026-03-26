# Zero → Hero roadmap (practical)

"Most secure coin" isn't something you can claim by code alone. You need a defendable model, incentives, decentralization, and years of adversarial testing. This roadmap is an actionable sequence to evolve this repo into something that can survive hostile environments.

## Phase 0: Safety baseline (COMPLETED)

- ✅ Separate **operator/admin** actions from public access:
  - Admin token required for admin routes.
  - Rate limiting enabled for any exposed deployment.
- ✅ Remove non-determinism from critical paths:
  - Keep any AI-based policy checks out of consensus.
- ✅ Container hardening (least privilege)
- ✅ Monitoring hooks and structured logs.

## Phase 1: Protocol specification (COMPLETED)

- ✅ Public spec (v1.0 stable)
- ✅ Test vectors for encoding
- ✅ Deterministic builds + reproducible releases

## Phase 2: Real P2P networking (COMPLETED)

- ✅ Secure P2P protocol with TLS + Ed25519 authentication
- ✅ Peer scoring system
- ✅ Fuzzing for network message parsers

## Phase 3: Consensus + incentives (IN PROGRESS)

- PoW security model defined
- Fee market implementation
- Reorg policy

## Phase 4: Wallet ecosystem (PENDING)

- Hardware wallet support.
- Safer signing flows (PSBT-like) and offline signing.
- Key rotation, multisig, recovery procedures.

## Phase 5: External verification (PENDING)

- Independent audits (in progress)
- Bug bounty program
- Public attack simulations + incident response playbooks