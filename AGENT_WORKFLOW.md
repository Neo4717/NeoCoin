# Neocoin "Autopilot" Workflow (for Codex/ChatGPT sessions)

This repo can be advanced with minimal back-and-forth by keeping a written backlog and using a consistent “autopilot” prompt each session.

## What I can/can't do

- I **can** implement features, refactors, tests, docs, and run local commands inside this workspace.
- I **cannot** run “in the background” without you starting a session and giving a prompt.
- I **will not** publish anything online or use network access unless you explicitly ask.

## One prompt to run a full cycle

Use this prompt anytime you want me to keep working without extra coordination:

> Read `AGENT_BACKLOG.md`. Pick the top 1–3 **P0/P1** unchecked items. Implement them end-to-end (code + docs + tests when appropriate). Keep changes small and non-breaking. Run `scripts/agent_check.sh --fix`. Update `AGENT_BACKLOG.md` (mark done/notes). Summarize what changed and what to test next.

If you want a smaller cycle:

> Do exactly one P0 item from `AGENT_BACKLOG.md`, run `scripts/agent_check.sh --fix`, update the backlog, and stop.

## How to add tasks (so I can just pick them up)

Edit `AGENT_BACKLOG.md` and add items using this format:

- [ ] **P0** Short title
  - Scope: 1–3 bullets
  - DoD: 1–3 bullets (what “done” means)
  - Notes: constraints or “don’t break X”

Priority guidance:
- **P0**: consensus correctness / funds safety / remote crash bugs
- **P1**: durability, performance, observability, UX blockers
- **P2**: nice-to-have features and polish

## Guardrails (assumed unless you override)

- Avoid new third-party deps that require downloading modules (network is restricted).
- Prefer deterministic consensus rules; avoid “local policy” leaking into consensus.
- Preserve backward compatibility unless a task explicitly calls for a hard fork / migration.
- Keep admin-only actions protected if `ADMIN_TOKEN` is set.

## Validation expectations

Every cycle should end with:

- `scripts/agent_check.sh --fix`
- If changes touch networking/docker/testnet behavior: `scripts/smoke_test.sh` (optional but recommended)

## Notes for you (repo owner)

- If you want me to do longer autonomous runs, just say: “Do as many P0/P1 items as you can this session.”
- If something is risky (consensus/storage migrations), I’ll implement behind feature flags and document the migration path in `README.md`.

