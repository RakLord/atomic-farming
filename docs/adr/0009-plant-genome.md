# ADR 0009 — Diploid plant genome

**Status:** accepted.
**Date:** 2026-08-31.

## Context

Breeding, mutation, individual variation, and every per-plant stat read off one data structure. Its shape is expensive to change once saves contain plants, so it is worth settling before the first crop exists.

Two properties are required and pull in different directions. Breeding must let the player *push stats upward* over generations, which is the progression loop. And the genome must drive a drawing in which **small genome changes produce small visual changes**, so that an offspring visibly resembles its parents.

## Decision

**1. Diploid: two alleles per gene, with expression separate from inheritance.**

```go
type Allele uint8
type GenePair struct{ A, B Allele }
type Genome [GeneCount]GenePair
```

The obvious cheaper model — one value per gene, averaged when breeding — fails the first requirement outright. Averaging regresses to the mean: an offspring can never exceed its best parent, the gene pool homogenises, and breeding stops being a progression mechanic within a few generations.

Diploid fixes this precisely because expression and inheritance are different operations. A plant with alleles `(240, 40)` *expresses* 140 under co-dominance but can still *pass on* the 240. Extremes survive in the pool while the visible plant stays smooth. `TestExtremeAllelesSurviveGenerations` asserts exactly this: after twelve generations of crossing a high line with a low one, a high homozygote reappears and no allele was ever invented.

It also buys a discovery mechanic for free — hidden alleles are invisible until bred — which later pays for a "genome scanner" upgrade.

`Allele` is a `uint8` so that a one-step mutation is 1/255 of the range: small enough to preserve visual locality, coarse enough that steps are meaningful. `Genome` is a fixed-length array rather than a slice, so it stays a comparable value type that copies cheaply and cannot be aliased between plots.

**2. Three expression rules, not four.**

`ExprAverage` (co-dominant) for numeric and visual traits, `ExprDominant` for categorical archetypes, `ExprRecessive` for traits that should hide.

A fourth "best-of" rule was planned and dropped: it is the same operation as `ExprDominant`, and two names for one behaviour is noise. `GeneVitality` simply uses `ExprDominant`, which reads correctly as "the hardier allele is dominant".

`ExprRecessive` earns its place on `GeneMutability`: high mutability being recessive means it hides in the pool and takes deliberate inbreeding to express, which matches how rare it should be.

**3. Species ranges remap; they do not clamp.**

```go
func Express(g Genome, r SpeciesRanges) Phenotype
```

Clamping to a species range would waste every mutation at a boundary — a plant already at its species maximum could never vary again. Remapping `[0,255]` onto `[Min,Max]` keeps the whole genetic range meaningful for every species, and means two species with identical genomes look different, which is correct.

**4. The gene catalog is append-only.**

A gene's index is a save-format identifier, exactly like `CropKind`. New genes go on the end; retired ones leave reserved slots. A genome encoding fewer genes than the current catalog is filled from catalog defaults on parse, which is what makes appending safe for existing saves; one encoding more has the extras ignored, so a save touched by a newer build still loads.

All 45 genes were declared at once, though only the ~32 morphology genes are read today. Declaring the gameplay genes up front means breeding and mutation operate over the complete genome from the first plant. Adding them later would default-fill every existing plant to an identical value — strictly worse than having varied them all along.

**5. Two strings, for two jobs.**

`Genome.String()` is lossless (`"1:"` + hex, ~180 chars) and is the shareable strain code. `Genome.Label()` is a short stable hash (`4F2A-91BC`) for naming a strain in the UI. Conflating them would mean either an unwieldy label or a lossy save.

**6. Species is `CropKind`; the genome varies within it.**

This reuses the registry from ADR 0005. A species supplies gene ranges that bound its genome, so every tomato is recognisably a tomato; cross-species breeding consults a cross table, keeping "new plant unlocked" a discrete event rather than a fuzzy gradient.

## Consequences

**Wins**
- Breeding is a real progression axis rather than a convergence to the mean.
- Hidden recessives give the breeding loop depth and a natural upgrade to sell.
- Adding a gene is one catalog entry and breaks no save.

**Costs**
- 90 bytes per plant, and every gene needs an expression rule chosen deliberately.
- Genotype and phenotype are different things, and the UI has to show both without confusing the player.
- The append-only rule is a discipline, not a compiler guarantee.

## Related

`internal/plant/`, ADR 0005 (crop registry), ADR 0010 (the rolls that breeding depends on), ADR 0011 (what the genome is drawn as).
