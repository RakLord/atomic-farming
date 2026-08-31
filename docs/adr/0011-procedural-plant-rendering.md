# ADR 0011 — Procedural plant rendering

**Status:** accepted.
**Date:** 2026-08-31.

## Context

A plant's appearance must be generated from its genome, deterministically, such that **a small genome change produces a small visual change**. That property is what makes breeding legible: without it, an offspring would look nothing like its parents and the whole mechanic reads as random.

The generator is also the part of the system least knowable from a specification. Whether the output looks good, varies enough, and grows smoothly is something you have to look at.

## Decision

**1. Geometry and rasterisation are separate packages.**

`internal/plant/morph` turns a phenotype into a `Blueprint` — normalised shapes with HSL colours and paint order — and imports no Ebitengine. `internal/render` turns a Blueprint into pixels and knows nothing about genes.

The split is the point. The geometry is the hard, tunable part, and keeping it renderer-free means it is unit-testable without a GPU. That mattered more than expected: Ebitengine cannot read pixels back before the game loop starts, so pixel-level assertions are impossible in tests. Every guarantee below is asserted on geometry instead, which is both possible and more precise than a pixel diff.

**2. Locality comes from indexed noise, not from hashing the genome.**

This is the load-bearing trick. The obvious way to vary leaf and petal placement is to hash the genome and seed a jitter stream from it. That gives excellent variety and *destroys* locality: one allele step reshuffles every offset.

Instead, `wobble(phenotype, index, channel)` takes its variety from the discrete index — which node, which petal — while genes only shift a low-frequency phase:

```go
phase := float64(index)*2.3999632 + float64(channel)*1.1701
drift := p.Unit(GeneJitter)*6 + p.Unit(GeneSymmetry)*4
return math.Sin(phase + drift)
```

A one-step gene change moves the phase by roughly 0.02 radians, nudging the result rather than randomising it. `TestOneStepMutationBarelyMovesGeometry` asserts that no point moves more than 0.012 normalised units for a one-step change to any continuous gene; measured worst case is 0.0034.

The four archetype selectors and the four count genes are excluded, because a one-step change to them legitimately crosses a bucket boundary. That is a real discontinuity, not a bug.

**3. One maturity parameter covers the whole life cycle.**

`Build(phenotype, t)` with `t` from 0 to 1. Stem height eases in, each node emerges over its own window, the flower opens, fruit sets. There are no discrete growth-stage sprites — a plant at any stage is the same function at a different `t` — so growth animation is free and a new stage costs nothing.

**4. Archetypes are parameters, not separate code paths.**

`stemProfile` bends a single construction. A change of archetype therefore moves the plant rather than replacing it, which keeps cross-archetype breeding coherent.

**5. Fit is measured on the mature plant and applied at every stage.**

Sizing the canvas to the largest plant the genome space can produce was measured at nearly three times the average plant, leaving the typical plot looking half empty. Instead the rare giant is compressed to fit, uniformly about the base of its stem.

Crucially, that fit is computed from the **mature** plant and applied throughout life. Re-fitting per stage made a plant visibly shrink — by up to 16% — the moment a flower opened or fruit set, because the new organ widened it and the whole plant was scaled to compensate. Measuring once costs a second geometry build per cache miss and is worth it.

**6. Canvas bounds are measured, not guessed.**

The canvas was set from the observed extremes across 20,000 genomes at 21 maturities each, then given margin, and `TestBlueprintStaysWithinCanvas` holds the guarantee. Height is capped at exactly one unit because the renderer scales by `CanvasMaxY`; the floor carries extra margin because fit is measured on the mature plant and a seedling's low leaves reach further below the soil, relative to its size, than a grown plant's do.

**7. Every organ is drawn with a darker edge.**

Flat fills of similar colour merge where they overlap. Without edges a rosette rendered as one solid dome rather than a whorl of blades. This is the single change that most improved legibility.

**8. Sprites are cached, keyed on the phenotype, with maturity quantised.**

Keying on the phenotype rather than the genome keeps species out of the key: two species express the same genome differently, but the same phenotype always draws the same plant, and a field of one strain rasterises once.

Maturity is quantised into `StageBuckets` (16). Without it, a plant growing through a tick would allocate a new texture every frame. The cache is LRU with a linear eviction scan — the cache is small and capped, evictions are rare, and a scan avoids a second ordering structure that could drift out of step with the map.

## What this cost to get right

Four defects were found by rendering contact sheets and looking at them, none of which any test would have caught:

- Plants floated a quarter of a tile above the soil, because the sprite was bottom-aligned rather than positioned by the plant's own origin.
- Leaves were sized absolutely, so every short plant was a pile of foliage with the stem buried inside it. They are now sized relative to the stem that carries them.
- A rosette's fan was spaced by the number of leaves *currently visible*, so the whole whorl re-spaced every time one sprouted.
- The mature-fit issue above.

`cmd/plantsheet` and the in-game lab (press `L`) exist because of this. The generator cannot be tuned from test output.

## Consequences

**Wins**
- Locality is guaranteed by an assertion, not by hope.
- Resolution independent: plants redraw crisply as the farm expands and plots shrink.
- Adding an archetype is one function and one enum entry.

**Costs**
- Two geometry builds per cache miss, for the mature-fit measurement.
- Rasterisation itself is not unit-tested; it is verified by eye through the lab and the contact sheet.
- The canvas constants are empirical, and a new archetype that reaches further will need them re-measured — which the canvas test will demand.

## Related

`internal/plant/morph/`, `internal/render/plantsprite.go`, `internal/render/spritecache.go`, `internal/render/lab.go`, `cmd/plantsheet/`.
