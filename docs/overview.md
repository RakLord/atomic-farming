# Atomic Farming — Game Overview

This document is the single source of truth for the game's concept, terminology, and scope. Feature-level detail lives in `docs/features/`; architectural decisions live in `docs/adr/`. Update this doc when the game model changes, and push implementation detail down into feature docs.

## Concept

A grid of **Plots** on which the player buys **Seeds**, plants **Crops**, waits for them to grow, and **Harvests** them for **Cash**. Cash expands the farm, buys better seeds, and funds automation. Cross-breeding harvested produce unlocks new crops with denser matter.

The end goal is a single piece of plant matter so dense it collapses the universe — the prestige event that resets the farm and starts the next layer.

Genre: incremental / idle. Platform: browser (Ebitengine → WASM → GitHub Pages), with desktop as the iteration target via `go run ./cmd/game`.

## Terminology (canonical)

| Term | Meaning |
|---|---|
| Cash | The primary currency, displayed with a leading `$`. Stored as a `bignum.Decimal`. |
| Plot | One farmable cell of the grid. Empty or holding exactly one Crop. |
| Crop | The thing growing in a Plot. `CropKind` is its stable save identifier. |
| Seed | The purchasable item that becomes a Crop when planted. |
| Growth | A Plot's mutable growth state: stage, progress toward the next stage, and whether it is ready. |
| Yield | What a harvest produces, before global modifiers. |
| Tick | One logical simulation step. |
| Layer | One rung of the prestige ladder. The base layer is **Field**. |
| Unlock | A purchased global upgrade. The owned set is what persists; its effects are derived. |
| Genome | A plant's 45 genes, each holding two alleles. Determines both how it looks and how it performs. |
| Allele | One of a gene's two copies. A child's alleles are always literally its parents'. |
| Phenotype | The expressed genome — what the plant actually shows, after species ranges are applied. |
| Strain | A genome, named by its short fingerprint (`4F2A-91BC`). |
| Density | The gene measuring mass per unit of produce. The stat the whole game climbs toward. |
| Seed stack | A quantity of seeds sharing one species and one exact genome. |
| Named strain | A plant recognised by name, either by a trait predicate or by an exact genome signature. |
| Discovery | A named strain the player has met. Logged permanently, and survives prestige. |

Two words are **banned** in code and UI because they are ambiguous:

- **"Size"** on its own — say **farm size** (plot dimensions) or **crop mass** (how much matter a crop holds). Only crop mass feeds the density goal.
- **"Growth"** as a synonym for progression — it means Plot growth state and nothing else. Player progression is **progress** or the specific axis.

## Crop model

`CropKind` is the **species**; the **genome** is the individual variation within it. A species supplies gene ranges that bound its genome, so every tomato is recognisably a tomato while no two are identical. See `docs/features/0001-plant-genome.md` for the gene reference and `docs/adr/0009-plant-genome.md` for why the genome is diploid.

A plant's appearance is generated from its genome rather than drawn by hand, deterministically and with a guarantee that a small genome change produces a small visual change — which is what makes a bred offspring visibly resemble its parents. See `docs/adr/0011-procedural-plant-rendering.md`.

A Crop is conceptually a pure function `(GrowContext, Growth) → Growth`. Adding a crop means one file in `internal/sim/crops/` that registers itself from `init()` — no edits to the tick loop, the save layer, or the registry.

Crop values are **immutable configuration**. All per-plot mutable state lives in the `Growth` handed to `Grow` and returned from it, which is why a `GridView` can safely hand out plots by value.

Crops that can be harvested implement the optional `Harvestable` capability. Regrowth is expressed through its return values rather than a second interface: `cleared` reports whether the plot empties (an annual); when it does not, `next` is the state the crop regrows from (a perennial).

See `docs/adr/0005-crop-registry-and-plot-model.md`.

## Simulation model

- **Fixed logical tick rate** (`sim.DefaultTickRate`, 10 Hz). Logical state advances only on ticks; rendering never mutates state.
- `Tick` never reads the wall clock, which makes `for i := 0; i < n; i++ { s.Tick() }` a valid fast-forward. That is how offline progress will work. See `docs/adr/0008-tick-model.md`.
- Growth is measured in ticks, so growth timers are deterministic and replayable.

## Progression axes

1. **Per-crop tiers** — better strains of an existing crop, bought with Cash.
2. **Per-crop mastery** — harvesting a crop repeatedly levels it, multiplying its yield and gating denser crops.
2b. **Breeding** — crossing two plants draws one allele per gene from each parent, so a rare extreme allele passes down intact and a breeding line keeps improving instead of converging on the average. Mutation shifts an allele by at most three steps, rarely.
3. **Global upgrades** — cross-cutting Cash sinks that fold into `GlobalModifiers` (growth rate, yield, sell price, seed cost, free plots). See `docs/adr/0006-global-modifier-pipeline.md`.
4. **Reset layers** — nested prestige. The base layer is **Field**; collapsing a sufficiently dense plant ascends out of it. Which state survives a reset is declared per rule, so persistency-over-prestige upgrades are a rule change rather than a rewrite. See `docs/adr/0007-layer-model-and-reset-registry.md`.

## Grid and expansion

- Starts **3×3**, designed up to **12×12** (`sim.MaxGridW`/`MaxGridH`).
- Dimensions are runtime state, not a compile-time constant: `Grid.Resize` is what farm expansion calls. See `docs/adr/0004-dynamic-grid-dimensions.md`.
- Durable unlocks grant an `ExtraPlots` budget, spent by `StartingGridSize` on whole rows and columns so a post-prestige farm starts larger while staying rectangular.

## Rendering / resolution

- **Logical resolution 1280×720 (16:9)**, scaled to the window by Ebitengine. Assets are authored at 1×.
- Layout: 48 px header, an 800 px grid area, and a 480 px UI column.
- Plot pixel size is **computed from the farm's current dimensions**, not fixed, so plots shrink as the farm grows. A 3×3 farm uses 160 px plots; a 12×12 farm uses about 62 px.
- The scaffold draws with vector primitives. `assets.TileFS` is the seam art drops into.

## Persistence

- One versioned envelope under a single key, written through `internal/save`. See `docs/adr/0002-versioned-save-envelope.md`.
- Platform split by build tag: `localStorage` on WASM, a JSON file under `os.UserConfigDir()/atomic-farming` on desktop.
- Cash and other growth-oriented scalars are `bignum.Decimal`, persisted as canonical scientific strings. See `docs/adr/0003-bignum-core.md`.
- A plant's genome persists as one compact string, and its roll seed as one integer. Random outcomes derive from that seed rather than from a live RNG stream, which is what keeps offline tick replay honest. See `docs/adr/0010-determinism.md`.
- `Load` rebuilds `Modifiers` from the owned unlock set, so retuning an upgrade's effect applies retroactively with no migration.

## Scope and phasing

- **Phase 0 — scaffold.** Module, packages, CI, WASM, save round-trip, tick loop, and the crop / modifier / reset seams.
- **Phase 0.5 — genome and generator.** The diploid genome, expression, breeding and mutation, deterministic rolls, the procedural plant generator, and the genetics lab.
- **Phase 1 — the base loop (this build).** Seeds, the Stem crop, growth and death in the tick loop, harvest and cash, the shop, and named strains with a discovery log. See `docs/features/0002-base-game-loop.md`.
- **Phase 2 — expansion.** Buying plots, more crops, and a deeper upgrade catalog.
- **Phase 3 — automation.** Sprinklers, auto-harvesters, and offline progress.
- **Phase 4 — breeding and unlocks.** Cross-breeding produce into denser crops.
- **Phase 5 — prestige.** The collapse event, the second layer, and persistency-over-prestige upgrades.

## Open questions (resolve in feature docs)

- The yield formula, and which crop properties feed it.
- Whether growth needs a closed-form solution for long offline catch-ups, or whether bounded tick replay is enough.
- What "density" is numerically, and what threshold triggers the collapse.
- Whether crops interact with their neighbours (`GrowContext.Grid` and the Affinity gene exist for this, and nothing uses them yet).
- What the cross-species table contains — which pairings produce which new species, and at what odds.
- How much of a plant's genome the player can see before breeding it, and what a "genome scanner" upgrade reveals.

## Related docs

- `docs/adr/` — architectural decisions.
- `CLAUDE.md` / `AGENTS.md` — agent-facing guidance and the command list.
