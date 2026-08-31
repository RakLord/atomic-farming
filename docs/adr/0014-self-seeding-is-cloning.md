# ADR 0014 — Self-seeding is cloning

**Status:** accepted. Supersedes the mutation decision in ADR 0013 §4.
**Date:** 2026-09-01.

## Context

Phase 1 shipped with harvested seeds mutating at 121 basis points per allele. Playing it for ten minutes produced a barn like this:

```
Stem  E5FC-9D18   x1
Stem  C25B-0253   x1
Stem  3B4B-2F86   x1
Stem  E8DE-ECCD   x1
...
```

Eight singleton lines, visually identical, none worth choosing between. The inventory keys stacks on the exact genome, so any difference at all splits a row.

The obvious fix — lower the rate — does not work, and understanding why is the whole decision.

`plant.Mutate` rolls **independently for all 90 alleles**. At 121 bp that put roughly two seeds in three through a change, each scattered across several genes. But even at 1 bp per allele, the arithmetic is `1 - (1 - 0.0001)^90`, still about one seed in 111, and each of those still drifts in more than one gene. The rate sets how often the mess appears; the **shape** guarantees that when it does appear, it is a smear rather than an event.

## Decision

**1. A self-seeded seed gets one roll, and a hit moves exactly one allele one step.**

```go
func MutateOnce(g Genome, seed uint64, chancePPM int) (Genome, bool)
```

A mutation becomes something you can point at — *this gene, this line, this generation* — instead of a fog. Every other seed is a clean copy, so a line stays a line.

**2. The per-allele `Mutate` stays, for crossing.**

Shuffling several genes at once is exactly right for a cross and exactly wrong for a plant seeding itself. Keeping both gives the distinction a name: **self-seeding is cloning with a rare copy error; crossing is where variation comes from.** That is also what will give cropsticks a job worth doing rather than a second route to the same outcome.

**3. Rates are in parts per million, not basis points.**

A one-in-ten-thousand event is 1 bp. Scaling 1 bp by a gene and dividing back down truncates straight back to 1, so Mutability would have silently done nothing. `ChancePPM` and `PartsPerMillion` sit beside the existing helpers; `SelfSeedMutationPPM = 100` is the baseline.

**4. Boundaries reflect rather than clamp.**

An allele at 0 that rolls a downward step moves *up* instead. Clamping would let a mutation fire and change nothing, breaking the one-step guarantee the function exists to make.

**5. Mutability and irradiation scale the rate; a ceiling caps it.**

Max Mutability is worth 10×. Two purchasable upgrades — **Seed Irradiator** and **Reactor Bed**, 12× each — take a high-Mutability line to roughly one seed in seven. `MaxSelfSeedMutationPPM` stops any stacking reaching certainty.

This is not decoration. At a base rate of one in ten thousand the bred strains would be unreachable and the naming system would sit inert, so a deliberate way to buy into drift is what keeps it alive. It also happens to be the right upgrade for a game called Atomic Farming.

**6. The inventory groups by species; named strains are promoted.**

`GameState.GroupSeeds` collapses a species' unnamed lines into one row and gives every named strain its own. The stored `Inventory.Stacks` is untouched, so grouping is a view and no save migrates. Groups come out in first-seen order — iterating the lookup map would reshuffle the list between frames, the same trap `IdentifyStrain` avoids.

Clicking a group with one line sows it; a group with several opens the seed index, which shows each line as a rendered plant with its density, growth and yield, and lets the player sow or discard it. Discard takes the whole line, because a lineage is the unit anyone wants to bin.

## Consequences

**Wins**
- Ordinary play no longer fragments the barn: 300 harvests at the base rate produce **one** line, against 42 with both irradiators bought.
- A mutation is legible — one gene, one step — so a drifting line can actually be followed.
- Drift becomes a strategy that is purchased rather than a tax that is paid.

**Costs**
- With co-dominant expression, a single allele step changes the expressed value by 0 or 1, so roughly half of all mutations are phenotypically silent. Realistic, and it makes drift slower than the raw rate suggests.
- Two mutation models now exist, and choosing the wrong one at a future call site would be quiet. The naming carries the distinction, and the doc comments on both point at each other.
- Balance is tuned around the irradiators. Adding a third tier will need the ceiling revisiting.

## Related

`internal/plant/breed.go`, `internal/plant/rng.go`, `internal/sim/harvest.go`, `internal/sim/upgrades.go`, `internal/sim/seeds.go`, `internal/render/seedindex.go`. ADR 0013 (the economy this amends), ADR 0012 (what drift is for).
