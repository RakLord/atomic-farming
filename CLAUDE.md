# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

An incremental/idle farming game built with **Ebitengine** in Go, targeting the browser via WASM (deployed to GitHub Pages) with desktop as the iteration target.

The game concept, terminology, and phasing live in `docs/overview.md` — **read it before making non-trivial changes**; it is the canonical source of truth for the game model. Architectural decisions live in `docs/adr/`, feature-level detail in `docs/features/`.

The current build is **Phase 0.5**. The pipeline works end to end — tick loop, save round-trip, desktop and WASM builds — and the plant genome and procedural plant generator are in. Plants can be generated, bred, mutated and drawn, but no gameplay exists yet: nothing is planted, grown, harvested or sold.

## Commands

```bash
# Desktop iteration (primary dev loop)
go run ./cmd/game

# Look at the procedural plant generator (it cannot be judged from test output)
go run ./cmd/game                            # press L for the genetics lab
go run ./cmd/plantsheet -mode population     # also: growth, mutations, archetypes

# Standard Go tooling
go build ./...
go vet ./...
go test ./...
go test ./internal/sim -run TestResetRules   # a single test

# WASM build (what CI does for GitHub Pages)
GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o web/game.wasm ./cmd/game
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" web/wasm_exec.js
```

CI (`.github/workflows/deploy.yml`) vets, tests, builds WASM on push to `main`, and publishes `web/` to GitHub Pages. `web/game.wasm` and `web/wasm_exec.js` are gitignored build artifacts.

## Architecture

Entry point: `cmd/game/main.go` loads the save (or starts fresh), wires state into a `render.Game`, and runs Ebitengine.

- `internal/sim` — grid, plots, crops, modifiers, layers, resets, tick loop, save format. **No Ebitengine imports.** All game logic lives here.
- `internal/render` — the `ebiten.Game` implementation. Reads `sim` state; never mutates it outside `Update`'s call to `Tick`. Owns layout geometry, colours, fonts.
- `internal/input` — maps an already-resolved `sim.Position` to UI intent. No Ebitengine, so it tests headless. Screen-to-plot mapping is `render.cellAt`.
- `internal/ui` — transient view state (hover, selection) that is never saved.
- `internal/save` — persistence, split by build tag: `localstorage_js.go` (`//go:build js && wasm`) uses `syscall/js`; `file_desktop.go` (`//go:build !(js && wasm)`) writes JSON under `os.UserConfigDir()/atomic-farming`. **Keep this split** — the WASM file imports `syscall/js`, which does not compile on desktop.
- `internal/plant` — the diploid genome: representation, expression, breeding, mutation, and the deterministic hash every gameplay roll goes through. Imports neither Ebitengine nor `sim`; `sim` depends on it, not the reverse.
- `internal/plant/morph` — genome to geometry. Pure Go, **no Ebitengine**, so the generator is testable without a GPU.
- `internal/bignum` — large-number decimal core, copied from the `ParticleAcceleratorAtHome` reference project.
- `internal/sim/crops` — concrete crops, one per file, self-registering. Deliberately empty until Phase 1.

## Load-bearing invariants

**Tick model.** Logical state advances **only** in `sim.Tick`, and `Tick` never reads the wall clock. This is what makes `for i := 0; i < n; i++ { s.Tick() }` a valid offline-progress fast-forward. Don't put state changes in `Draw` or make `Update` wall-clock-dependent. See `docs/adr/0008-tick-model.md`.

**Modifiers are derived, never authoritative.** `GameState.Unlocks` is the source of truth; `Modifiers` is a cache rebuilt by `rebuildModifiers` after every purchase, every load, and every reset. It is not persisted. This is what lets an upgrade's effect be retuned and apply retroactively to existing saves with no migration. See `docs/adr/0006-global-modifier-pipeline.md`.

**Every `GameState` field needs a reset decision.** Adding a field fails `TestResetRulesCoverEveryField` until it is either claimed by a `ResetRule` or added to `resetExemptFields` with a reason. That is intentional — decide what a prestige does to it. See `docs/adr/0007-layer-model-and-reset-registry.md`.

**Adding a crop touches one directory.** Drop a file in `internal/sim/crops/` that registers itself from `init()`. No edits to `tick.go`, `save.go`, or the registry. `CropKind` strings are save identifiers: unique, and never renamed once shipped.

**The grid is runtime-dimensioned.** `Grid.W`/`H` with a flat `Plots` slice, not a fixed array — the farm grows. Render geometry is computed from the current dimensions; `cellAt` must stay the exact inverse of `cellRect`. See `docs/adr/0004-dynamic-grid-dimensions.md`.

**Gameplay randomness is integer-only and seed-derived.** Every random outcome goes through `plant.Chance`/`plant.Roll` with a persisted seed — never `math/rand`, never a live stream, never a float. A float-derived chance can differ between the desktop and WASM builds and silently desynchronise an offline tick replay. Probabilities are integers in basis points; `Phenotype.Scaled` is the integer accessor. Float accessors (`Unit`, `Lerp`, `UnitFloat`) are for pixels only. See `docs/adr/0010-determinism.md`.

**The gene catalog is append-only.** A gene's index is a save-format identifier, like `CropKind`. Add to the end; never reorder or delete. Same for `Purpose` values and archetype enum ordering.

**Small genome change, small visual change.** This is a guarantee, not an aspiration, and it is what makes breeding legible. It is asserted by `TestOneStepMutationBarelyMovesGeometry`. It is easy to break by accident: seeding jitter from a hash of the genome would pass every other test while making a mutated plant look nothing like its parent. See the comment on `morph.wobble`, and `docs/adr/0011-procedural-plant-rendering.md`.

**The generator cannot be tuned from test output.** Four real defects — floating plants, absolutely-sized leaves, a rosette re-spacing its own whorl, and a plant shrinking when its flower opened — were found only by rendering contact sheets and looking at them. Use `cmd/plantsheet` and the in-game lab.

**Save tests must isolate the config directory.** Use `t.Setenv("XDG_CONFIG_HOME", t.TempDir())` and `t.Setenv("HOME", ...)`, or the test writes into the developer's real config directory. See `isolateSave` in `internal/sim/save_test.go`.

## Terminology

`docs/overview.md` has the full table. Two words are banned because they are ambiguous:

- **"Size"** alone — say **farm size** (plot dimensions) or **crop mass** (how much matter a crop holds).
- **"Growth"** as a synonym for progression — it means Plot growth state and nothing else.
- **"Seed"** is overloaded: a **plant seed** is what you buy and sow, a **roll seed** (`Plot.Seed`, `GameState.WorldSeed`) is randomness input, and a **strain code** is the genome's shareable string. Say which.

## Reference project

`../ParticleAcceleratorAtHome` is a shipped Ebitengine incremental using the same stack, and the source of the `bignum` and `save` packages. Its `docs/adr/` is worth consulting for patterns. Two of its foundations were **deliberately not** copied — a fixed-size grid and an ad-hoc prestige reset — for the reasons in ADR 0004 and ADR 0007.
