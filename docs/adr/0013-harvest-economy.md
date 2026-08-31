# ADR 0013 — The harvest economy

**Status:** accepted.
**Date:** 2026-08-31.

## Context

Phase 1 needs a loop: buy a seed, sow it, wait, gather it, get paid, buy another. Every step of that touches determinism, because ADR 0008 makes tick replay the mechanism for offline progress, and half the steps are random.

## Decision

**1. Growth is integer progress against an integer threshold.**

A plant accumulates `GrowthUnitsPerStage` (1000) of progress per stage, gaining 8 to 50 units a tick depending on its `GrowthRate`. No fractions, so growth replays exactly. A default Stem matures in about 94 ticks — nine seconds.

`Modifiers.GrowthRateMul` is a `bignum.Decimal`, so it reaches integer growth maths through `modifierPermille`. That is a sanctioned exception to the integer-only rule and is safe for a specific reason: `Decimal.Float64` is a `math.Pow10` table lookup and one IEEE-754 multiply, both bit-identical on every platform, and the scale and truncation after it are exact.

**2. The death roll happens once per stage, not once per tick.**

This is the decision most worth writing down, because the wrong version looks harmless. A plant lives for hundreds of ticks; a per-tick roll compounds across all of them, so even one basis point kills nearly everything before maturity. Rolling on stage advance means `MaxStageDeathBP` (3%) is what it appears to be: about 9% total mortality across a Stem's three transitions, for the frailest possible plant.

`TestDeathIsRolledPerStageNotPerTick` grows two hundred minimum-Vitality plants and fails if fewer than three quarters survive.

Death lives in the tick loop rather than in `Crop.Grow`, so it applies to every crop uniformly and a new crop cannot forget it.

**3. Value is shaped by two genes, and Density dominates.**

Unit value is `(1 + Quality) × (1 + 4·Density)` in permille, times `SellPriceMul`. Density is weighted four times as heavily as Quality on purpose: the prestige goal is matter dense enough to collapse the universe, so Density must be the stat worth chasing. `TestDensityDominatesValue` asserts it outranks Quality.

Counts are integers and money is `bignum.Decimal`, per ADR 0010.

**4. Harvested seeds carry the parent genome, mutated.**

This is what keeps the game moving. Without it every plant would stay genetically identical to the shop seed it came from, the gene pool would never drift, and no predicate strain could ever be discovered by playing — the whole naming system would only label things the player bought.

Genetics alone contributes at most 40 bp of mutation, which would take hundreds of harvests to drift anywhere, so self-seeding adds `SelfSeedBonusBP` (120) on top. A strain visibly wanders within a session while `Mutability` still matters.

**5. Seed return averages slightly below one per harvest.**

Two rolls at 25–65% each, scaled by `Yield Amount`. A farm that never needs to buy a seed makes the shop pointless; one that always does is a treadmill. Slightly lossy means the shop stays relevant without being mandatory. `TestSeedReturnAveragesBelowOneForAnOrdinaryPlant` pins the range.

**6. Harvest is player-initiated but plant-determined.**

`Harvest` lives outside `Tick`, yet every roll it makes derives from the plot's stamped seed. The outcome belongs to the plant, not to when the player happened to click — so a harvest cannot be re-rolled by saving and reloading.

**7. Rare seeds reuse the unlock pipeline.**

A rare offer sets `RequiresUnlock` and the shop consults `IsUnlocked`. The unlock's `Apply` is nil, which `rebuildModifiers` already skips. Rare seeds therefore need no economy of their own — buying one is two ordinary cash purchases.

## Consequences

**Wins**
- The whole loop replays identically from a save, so offline progress can reuse it verbatim.
- Every balance lever is a named constant with a test pinning its effect.
- Rare content costs no new systems.

**Costs**
- `modifierPermille` is a float bridge in otherwise integer code, and it is only safe because of a property of `math.Pow10` that a future refactor could break unknowingly.
- The mutation rate is tuned for a session, not a run. Once breeding lands properly the self-seed bonus will likely need lowering.

## Related

`internal/sim/growth.go`, `internal/sim/harvest.go`, `internal/sim/shop.go`, ADR 0008 (tick replay), ADR 0010 (determinism), ADR 0012 (what the drift is for).
