# Security status (prototype)

This repository is a **prototype** blockchain node intended for local demos and experimentation. It is **not** a production-ready cryptocurrency.

If you expose the HTTP API to the public internet, you should assume:

- It will be scanned and attacked quickly (DoS, fuzzing, abuse of any write endpoints).
- Any bug in parsing, validation, storage, or concurrency can become a remote crash or data-corruption issue.
- There is no meaningful “economic security” in a single-node deployment.

## What this repo now does to reduce obvious risk

- **Admin endpoints are disabled unless `ADMIN_TOKEN` is set.**
  - Endpoints like `POST /mine/once`, `POST /audit/chain`, `POST /block` require `Authorization: Bearer <ADMIN_TOKEN>`.
- **Guardrail against accidental internet exposure.**
  - If `HTTP_ADDR` binds to all interfaces (`0.0.0.0` / `::`), the server refuses to start unless `ADMIN_TOKEN` is set.
- **Rate limiting is supported** via `RATE_LIMIT_REQUESTS` and `RATE_LIMIT_BURST`.

## Recommended “online demo” deployment (minimum)

1. Keep the node private (VPN / SSH tunnel) if possible.
2. If you must expose it publicly:
   - Put it behind a reverse proxy with TLS.
   - Enable rate limiting.
   - Set a strong `ADMIN_TOKEN`.
   - Bind explicitly with `HTTP_ADDR=0.0.0.0:8080` only when you understand the risk.

## Roadmap to a defensible public network (“zero to hero”)

To make credible security claims, you need a real decentralized threat model. Typical major milestones:

1. **Protocol hardening**
   - Strict, versioned wire formats; remove ad-hoc HTTP from consensus-critical flows.
   - Deterministic validation only (no non-deterministic “AI policy” in consensus).
2. **P2P network layer**
   - Authenticated transport (Noise/TLS), peer reputation, banning, eclipse resistance.
   - Separate “RPC” (operator/admin) from “P2P” (peer traffic).
3. **Consensus + incentives**
   - Clear consensus rules, reorg limits, fee market, and DoS-resistant mempool policy.
   - Distributed mining / validator set; otherwise there is no economic security.
4. **Key management**
   - Hardware wallet support, safe signing APIs, secure backup/recovery flows.
5. **Operational security**
   - Deterministic builds, reproducible releases, CI security scans, dependency pinning.
6. **External review**
   - Public spec, test vectors, fuzzing, and independent audits.

Until those are in place, treat this project as an educational system, not a production coin.
