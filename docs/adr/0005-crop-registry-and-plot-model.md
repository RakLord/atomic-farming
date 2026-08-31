# ADR 0005 — Crop registry and the Plot model

**Status:** accepted.
**Date:** 2026-08-31.

## Context

The roadmap adds crops in every phase, and Phase 4 (breeding) turns crop count into a combinatorial matter. If adding a crop means editing a central switch in the tick loop and another in the save layer, that cost is paid dozens of times and each edit is a chance to forget one.

The reference project solved this with a registry plus capability interfaces, after first shipping the central-switch version and finding it painful (its ADR 0003 documents the extraction). Start where it ended up.

## Decision

**1. `Crop` is an interface; concrete crops live in `internal/sim/crops` and self-register from `init()`.**

`sim` owns the interface, `CropKind`, the registry, and the capability interfaces. The binary blank-imports `internal/sim/crops` so registrations run before a save is loaded. Adding a crop is one new file — no edits to `tick.go`, `save.go`, or the registry.

**2. `CropKind` is a save-format identifier.** It is the string stored on disk and used to reconstruct the concrete type. Registering a duplicate panics at startup rather than silently shadowing. Never rename a shipped kind.

**3. Crop values are immutable configuration; mutable state lives in `Growth` on the `Plot`.**

```go
type Plot struct {
    Crop   Crop     // nil when empty
    Growth Growth   // stage, progress, ready
}
```

This is what makes `GridView.PlotAt` safe to return by value: a caller can copy a plot and mutate the copy without touching live grid state, which a test asserts directly.

**4. Optional behaviour is a capability interface, not a fat `Crop`.**

`Harvestable` is checked by type assertion. A crop that cannot be harvested simply does not implement it, rather than implementing a stub that returns nothing.

**Regrowth is expressed through `Harvest`'s return values, not a separate interface.** `Harvest` returns `(yield, next, cleared)`: `cleared` means the plot empties (an annual); otherwise `next` is the state the crop regrows from (a perennial). A separate `Regrower` interface was considered and rejected — it would carry a number that `Harvest` has to agree with, and two sources of truth for one behaviour is how they drift apart.

**5. `GrowContext` is the read-only world view handed to crops.**

It carries the grid view, position, tick, normalized modifiers, and layer. Implementations must not mutate anything reachable through it, nor retain it across ticks. `NewTestGrowContext` returns one with modifiers already normalized so crop tests cannot forget that precondition.

`GrowContext.Grid` exists before anything uses it, deliberately: neighbour-aware crops are an obvious later mechanic, and adding the field afterwards would change every crop's signature.

## Consequences

**Wins**
- Adding a crop touches one directory.
- `ComponentKind`-style save identifiers stay stable even if the Go types move.
- Component tests need no graphics context.

**Costs**
- A crop that is never blank-imported vanishes silently from the registry, and its saves fail to load. The blank import in `cmd/game/main.go` is load-bearing and commented as such.
- Capability interfaces must be remembered at each call site; a harvest path that forgets to check `Harvestable` just does nothing.
