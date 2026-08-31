# ADR 0006 — Global modifier pipeline

**Status:** accepted.
**Date:** 2026-08-31.

## Context

The brief asks for "a middleware style system so we can slip in new multipliers and layers at a later point". Concretely: global upgrades such as "all crops grow 10% faster", "seeds cost half", "sell price ×2" must be addable without touching the code that reads them.

Two questions bound the design.

1. **Where does the aggregated multiplier live?** Scattered, with each feature consulting the upgrade set itself, or centralized in one struct every hot path reads?
2. **What is the source of truth?** The aggregated multipliers, or the set of purchased upgrades?

Scattered reads couple every crop to the upgrade catalog and make every hot path pay a map lookup. Centralizing wins.

The second question is the one that actually matters. Persisting multipliers means that if an upgrade is ever retuned, existing saves keep the old number forever — the player who bought "+10% growth" before it became "+15%" is stuck at 10% permanently, and the only fix is a migration. Persisting the purchase set and **deriving** multipliers means every retuning applies on next load, for free.

## Decision

**1. `GlobalModifiers` aggregates every active global effect into one struct.**

Decimal fields are multiplicative; integer fields are additive. Hot paths read fields directly.

**2. `GameState.Unlocks` is the source of truth; `Modifiers` is a derived cache.**

```go
func rebuildModifiers(s *GameState) {
    s.Modifiers = GlobalModifiers{}
    for id, owned := range s.Unlocks {
        if !owned { continue }
        if u, ok := UnlockCatalog[id]; ok && u.Apply != nil {
            u.Apply(&s.Modifiers)
        }
    }
}
```

`rebuildModifiers` runs after every purchase, after every load, and at the end of every layer reset. Unknown IDs are skipped, so a retired unlock cannot break an old save.

**3. `Modifiers` is not persisted (`json:"-"`).**

The reference project persists its equivalent "so saves are self-describing". In practice that stores values that are *always* discarded on load, and — because `encoding/json` never treats a struct as empty — `omitempty` does not apply to a `bignum.Decimal`, so an untouched cache serialises as a wall of misleading `"0"` multipliers. A save that appears to say "growth rate ×0" is worse than no entry at all. `Unlocks`, the actual source of truth, is in the save and is readable.

**4. `Normalized()` promotes zero-valued Decimals to 1.**

The zero value of a multiplicative field means "no bonus", but multiplying by it would zero the result. `Normalized` makes read sites safe without guards. The **tick loop calls it once per tick** and threads the result through `GrowContext` — not once per crop.

**5. Unlocks carry `Apply(*GlobalModifiers)` closures, not static values.**

A closure lets one unlock touch several fields ("bigger farm *and* faster growth"). Every `Apply` must be idempotent under `rebuildModifiers`: map iteration order is random, so the same owned set must always produce the same result. `mulModifier` exists so a multiplicative fold composes correctly regardless of order.

## Consequences

**Wins**
- Global upgrades are a sealed extension surface: a new one is a catalog entry plus, at most, one struct field.
- Balance retuning applies retroactively to every existing save with no migration.
- One cache, read directly, with no per-crop map lookups.

**Costs**
- Two representations of one concept, with `rebuildModifiers` needing to run in exactly the right places. Missing a call leaves a stale cache — which is why not persisting it matters: a stale in-memory cache dies with the process.
- The catalog is code, not data. Adding or retuning an unlock needs a deploy. Fine for an all-client-state incremental.

## Related

- `internal/sim/modifiers.go`, `internal/sim/unlocks.go`.
- ADR 0007 — layer resets call `rebuildModifiers` last.
