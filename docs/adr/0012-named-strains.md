# ADR 0012 — Named strains

**Status:** accepted.
**Date:** 2026-08-31.

## Context

The genome layer gives every plant 45 genes of variation, and nothing in the game reacts to any of it. Variation nobody has a reason to care about is noise.

A naming system fixes that: a plant whose genes match a known pattern gets a name, and finding one is a discovery. But how a plant is matched to a name decides whether the system feeds the breeding loop or merely labels things the player was handed.

## Decision

**1. Two matching mechanisms, for two different jobs.**

```go
type NamedStrain struct {
    Match     func(p plant.Phenotype) bool  // a predicate over expressed traits
    Signature *plant.Genome                 // one exact genome
    // ... plus ID, Name, Goal, Rarity, Kind, Specificity
}
```

A **predicate** is a condition over visible traits — "Density ≥ 210 on a stem thicker than 200". Because it describes conditions rather than a point in gene-space, it is *reachable by breeding*, and its `Goal` field doubles as a target the UI can state plainly. This is what turns breeding from a slot machine into a puzzle.

A **signature** is a single exact genome. Reaching it by breeding is effectively impossible — 45 genes × 2 alleles all matching — which is precisely why it suits handcrafted legendaries that arrive as a bought seed. Voidshoot is one.

Exactly one must be set; `RegisterStrain` panics otherwise, because a strain with neither would silently never match and one with both would have ambiguous semantics.

**2. The name is derived, never persisted.**

Same rule as `GlobalModifiers` (ADR 0006). A stored name could only ever go stale. Deriving it means adding a strain, retuning a predicate, or renaming one applies retroactively to plants already sitting in a save, with no migration.

The only persisted part is the discovery log:

```go
GameState.DiscoveredStrains map[StrainID]bool
```

`StrainID` is therefore a save-format constant. `DiscoveredCount` ignores unknown IDs, so a save touched by a build that knew more strains still loads.

**3. `IdentifyStrain` iterates `StrainCatalogOrder`, not the catalog map.**

Go randomises map iteration. Resolving through the map would let a plant that satisfies two strains name itself differently between frames — a bug that looks like a rendering glitch and is chased in entirely the wrong place.

**4. Ties break on `Specificity`, highest wins.**

A stricter strain sits inside a looser one; without an explicit ordering, which name a plant gets would depend on registration order. Voidshoot's specificity is far above every predicate so a legendary is never demoted to whatever else it happens to satisfy.

**5. Strains are scoped to a species, and must be reachable within it.**

`Kind` restricts a strain to one crop. More importantly, a predicate's thresholds have to fall inside that species' gene ranges — `Express` remaps into the species window, so a strain demanding Density ≥ 210 from a species capped at 120 can never be grown, and nothing else in the system would report it.

`TestEveryPredicateStrainIsReachable` samples homozygous genomes through each species' ranges and fails if a strain never matches. It samples *homozygotes* deliberately: random heterozygous genomes average their alleles, which crowds every expressed value toward the middle and would make a rare-but-reachable strain look impossible.

**6. The discovery log is durable across prestige.**

Listed in `resetExemptFields`: a collection you lose on prestige is not a collection.

## Consequences

**Wins**
- Adding a strain is one catalog entry; retuning one applies to existing saves.
- Predicates give breeding explicit goals rather than hoping the player invents their own.
- Unreachable content fails a test instead of shipping.

**Costs**
- A predicate's thresholds and a species' ranges are coupled: widening a species window can make an existing strain far more common, and narrowing one can strand it. The reachability test catches the second case, not the first.
- Every strain that matches is evaluated on every identification. The catalog is small and identification is not per-tick, so this is fine until it is not.

## Related

`internal/sim/strains.go`, `internal/sim/crops/stem.go`, ADR 0006 (the derived-cache rule this follows), ADR 0009 (the genome being matched).
