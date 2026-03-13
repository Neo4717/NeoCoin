# Neocoin Backlog (agent-driven)

Keep this list as the single source of truth for what to build next. I will take the highest-priority unchecked items first.

## P0 (security / consensus correctness)

- [x] **P0** Persist full block DAG + state safely
  - Scope: replace/extend `store.go` to persist blocks-by-hash and canonical tip atomically
  - DoD: node restart preserves chain + balances; reorg works after restart; no full rewrite on every block
  - Notes: keep a migration path from current store; prefer simple KV (BoltDB) if possible without net deps, otherwise file-based indexes
  - Done: BoltDB-backed store (`blockchain/store_bolt.go`) persists canonical + blocks-by-hash; best-effort migration from legacy gob.

- [x] **P0** Merkle root + proof-of-inclusion endpoint
  - Scope: compute Merkle root for transfers; include in block header; add `GET /tx/proof/{txid}` returning Merkle branch
  - DoD: proof verifies against header root; stable across nodes; covered by unit tests
  - Notes: consensus change → gate behind `MERKLE_ENABLE` until ready
  - Done: v2 blocks commit to `merkleRootHex` in PoW header; `GET /tx/proof/{txid}` implemented; unit tests added.

- [x] **P0** Consensus test suite (table-driven)
  - Scope: unit tests for coinbase economics, cumulative-work fork choice, timestamp rules (MTP), difficulty schedule
  - DoD: `go test ./...` covers critical rules; tests are deterministic and fast
  - Done: tests cover coinbase economics, MTP, cumulative work fork choice, and difficulty enforcement.

## P1 (durability / usability)

- [x] **P1** Address history index + API
  - Scope: index canonical transfers by `from/to`; add `GET /address/{addr}/txs?limit=&cursor=`
  - DoD: explorer can show address page; handles reorg rebuild
  - Done: in-memory canonical address index now stays up-to-date on mining + reorg; HTTP API added at `GET /address/{addr}/txs` (enables explorer address pages).

- [x] **P1** WebSocket subscriptions by topic
  - Scope: allow `{"type":"subscribe","topic":"address","address":"..."}` messages from client
  - DoD: reduces bandwidth vs broadcasting everything; remains compatible with current broadcast behavior
  - Done: WS server now supports `subscribe`/`unsubscribe` for `topic=address`, `topic=type`, and `topic=all`; legacy clients that send no messages still receive the full broadcast stream.

- [x] **P1** Keystore UX hardening
  - Scope: support password from stdin (non-echo if possible), avoid passing secrets in argv, add `WALLET_PASSWORD_FILE`
  - DoD: docs updated; CLI ergonomics improved without breaking old commands
  - Done: keystore CLI reads password from `WALLET_PASSWORD_FILE`, `WALLET_PASSWORD`, argv (warns), interactive prompt (non-echo when supported), or stdin (`WALLET_PASSWORD=-`).

## P2 (polish)

- [ ] **P2** Explorer UX polish
  - Scope: pagination, better errors, copy-to-clipboard, mobile tweaks
  - DoD: no breaking API changes
