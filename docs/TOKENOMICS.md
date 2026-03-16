# NeoCoin Tokenomics

## Basic Parameters

| Parameter | Value |
|-----------|-------|
| Maximum Supply | 21,000,000 NEO |
| Decimal Places | 8 |
| Genesis Supply | 0 (pure PoW start) |
| Premine | None |
| Team Allocation | None |
| ICO / Sale | None |

## Block Rewards

| Phase | Blocks | Reward per Block |
|-------|--------|------------------|
| Era 1 | 1 – 210,000 | 50 NEO |
| Era 2 | 210,001 – 420,000 | 25 NEO |
| Era 3 | 420,001 – 630,000 | 12.5 NEO |
| Era 4 | 630,001 – 840,000 | 6.25 NEO |
| ... | ... | halving continues |

- **Halving interval:** 210,000 blocks (~4 years at 1 block/minute target)
- **Tail emission:** 0 (rewards stop after ~133 years)

## Fees

- **Transaction fees:** 100% goes to miner
- **No fee burn** in v1.0
- **Minimum fee:** 1 unit (subject to change)

## Emission Schedule

```
Year 1 (approx):  ~15,768,000 NEO mined
Year 2:           ~7,884,000 NEO  
Year 3:           ~3,942,000 NEO
Year 4:           ~1,971,000 NEO
...
```

After ~33 halvings (~133 years), block reward approaches zero. Maximum supply of 21M reached.

## Comparison to Bitcoin

| Metric | Bitcoin | NeoCoin (v1.0) |
|--------|---------|----------------|
| Max Supply | 21,000,000 | 21,000,000 |
| Initial Reward | 50 BTC | 50 NEO |
| Halving | 210,000 blocks | 210,000 blocks |
| Consensus | PoW | PoW |
| Premine | None | None |

## Important Notes

1. These parameters apply to the mainnet.
2. **AI Auditor does NOT affect tokenomics.** It's a policy check only, not part of consensus.
3. Parameters subject to future governance decisions.

## Future Considerations (not implemented)

- Fee burn (EIP-1559 style)
- Dynamic block size
- DAO / governance
- Smart contracts (future roadmap)

---

*Last updated: March 2026*
