# ADR 0001 — Package layout and the sim/render split

**Status:** accepted.
**Date:** 2026-08-31.

## Context

Atomic Farming ships to two targets: a desktop binary for iteration and a WASM build for GitHub Pages. It is an incremental game, so the parts most likely to churn — crop behaviour, upgrade catalogs, balance numbers — are also the parts most in need of fast, headless tests.

The reference project `ParticleAcceleratorAtHome` established a layout that survived five phases of feature work. There is no reason to re-derive it.

## Decision

**1. Game logic lives in `internal/sim` and imports no Ebitengine.**

`sim` owns the grid, plots, crops, modifiers, layers, resets, the tick loop, and the save format. It is a plain Go package that compiles and tests without a graphics context. This is the constraint that keeps the whole simulation testable; enforcing it is worth more than the occasional convenience of drawing from inside sim.

**2. `internal/render` implements `ebiten.Game`.**

It reads `sim` state and never mutates it outside `Update`'s call to `Tick`. Layout geometry, colours, and fonts live here, not in sim.

**3. `internal/input` maps resolved plot coordinates to intent.**

Screen-to-plot mapping needs layout geometry, so it happens in `render` (`cellAt`). `input` receives an already-resolved `sim.Position` and updates `ui.UIState`. That keeps input free of Ebitengine and testable without a window — `internal/input/input_test.go` runs headless.

**4. `internal/ui` holds view state that is never saved.**

Hover, selection, and later which panels are open. `sim` never reads it, so gameplay stays reproducible from `GameState` alone.

**5. `internal/save` is the platform seam, split by build tag.**

`localstorage_js.go` (`//go:build js && wasm`) and `file_desktop.go` (`//go:build !(js && wasm)`). Do not collapse these into one file with a runtime branch: the WASM file imports `syscall/js`, which does not compile on desktop.

**6. `internal/bignum` and `internal/save` are copied from the reference project, not shared.**

A shared module would propagate fixes but costs a second repository, versioning, and a `replace` directive during local development. The two games are expected to diverge. Copying is the cheaper trade at this size.

## Consequences

**Wins**
- `go test ./...` runs the whole simulation with no display.
- The WASM and desktop builds differ in exactly one package.
- Render churn cannot break gameplay.

**Costs**
- A bug fixed in the reference project's `bignum` will not propagate here.
- `render` owns geometry that `input` conceptually needs, so click handling is a two-step hop through `cellAt`.

## Related

- `docs/adr/0008-tick-model.md` — the invariant that makes the split load-bearing.
