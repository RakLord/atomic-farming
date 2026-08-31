# ADR 0004 — Dynamic grid dimensions

**Status:** accepted.
**Date:** 2026-08-31.

## Context

The reference project `ParticleAcceleratorAtHome` declares its grid as a compile-time constant backing a fixed array:

```go
const GridSize = 5
type Grid struct { Cells [GridSize][GridSize]Cell; ... }
```

That works there because the grid never changes size during play. In Atomic Farming, **expanding the farm is a core mechanic and a primary Cash sink**. A fixed array would mean either shipping the maximum farm from the start and hiding most of it, or a save migration every time the farm grows.

The render layer inherits the same problem: the reference derives `cellSize` and padding from the `GridSize` constant. Plots that must shrink as the farm grows cannot be laid out from a constant.

## Decision

**1. `Grid` carries its own dimensions and a flat, row-major slice.**

```go
type Grid struct {
    W, H  int
    Plots []Plot   // len == W*H; the plot at (x, y) is Plots[y*W+x]
}
```

A flat slice keeps a resize to a single reallocation and serialises to JSON without a nested-array shape change.

**2. `Grid.Resize(w, h)` preserves plots by coordinate.**

Every plot still inside the new bounds keeps its contents at the same `(x, y)`; plots outside are discarded. This is what farm expansion calls. Growing is non-destructive; shrinking is not, and no caller should shrink a farm holding crops the player wants.

**3. `Grid.normalize()` repairs a plot slice whose length disagrees with `W*H`.**

`Load` calls it, so a truncated or hand-edited save cannot panic the tick loop. Dimensions are authoritative; the slice is resized to match.

**4. Render geometry is computed, not constant.**

`gridGeometry` fits plots to the grid area and centres them, clamped to `[minCellSize, maxCellSize]`. `cellAt` is written as the exact inverse of `cellRect` and tested as such across the whole design range, because a mismatch between them is a click landing on the wrong plot — the kind of bug that is invisible until a player reports it.

**5. Free plots granted by prestige are a budget, not a dimension.**

`GlobalModifiers.ExtraPlots` is an additive integer that `StartingGridSize` spends on whole rows and columns, alternating so the farm stays near-square and stopping at `MaxGridW`/`MaxGridH`. Modelling it as a budget rather than as `+1 width` keeps the modifier composable across several unlocks.

## Consequences

**Wins**
- Farm expansion is `Resize`, with no save migration.
- The renderer handles any farm from 1×1 to 12×12 with no layout changes.
- A corrupt grid is repaired at load instead of crashing at tick.

**Costs**
- Plot access is `g.At(p)` returning `(*Plot, bool)` rather than direct indexing, so every caller handles the bounds case.
- `W`, `H`, and `len(Plots)` can disagree in a hand-edited save, which is why `normalize` exists.
- Bounds checks are now runtime rather than compile-time.

## Alternatives considered

- **Fixed maximum array with an active sub-rectangle.** Rejected: every iteration would need the active bounds anyway, and the save carries dead plots forever.
- **`[][]Plot` nested slices.** Rejected: two allocations per resize, and a ragged grid becomes representable.
