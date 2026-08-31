# ADR 0008 — Fixed tick model

**Status:** accepted.
**Date:** 2026-08-31.

## Context

A farming incremental is built on timers: a crop is ready N units of time after planting. How that time is measured determines whether saves, offline progress, and testing are tractable or a source of drift.

Two options: advance state by wall-clock delta each frame, or advance it in fixed logical steps.

## Decision

**1. Logical state advances only on fixed-rate ticks** (`sim.DefaultTickRate`, 10 Hz). Growth is counted in ticks, not seconds.

**2. `Tick` never reads the wall clock, and nothing outside `Tick` mutates logical state.**

Rendering may use delta time for interpolation, but `Draw` must not change state. `render.Game.Update` calls `Tick` exactly once and is the only place that does.

**3. The payoff is that fast-forward is trivially correct.**

```go
for i := 0; i < n; i++ { s.Tick() }
```

is by construction identical to having played those ticks. Offline progress is then "compute elapsed ticks from `LastSaveUnix`, replay them" — no separate catch-up code path that can disagree with the live one. `Save` stamps `LastSaveUnix` for exactly this; nothing reads it yet.

**4. Determinism is the constraint that keeps this true.**

No wall-clock reads, no map-iteration-order dependence in anything that affects state, and no randomness without a seed stored in the save.

## Consequences

**Wins**
- Growth timers are exact and replayable.
- Offline progress reuses the live tick path rather than approximating it.
- Tests advance time by calling `Tick`, with no sleeping or faking of clocks.

**Costs**
- Long offline periods cost real replay time: 8 hours at 10 Hz is 288,000 ticks. Cheap on a small grid, but this bounds how far tick replay scales.
- The tick rate is a gameplay parameter, not a rendering one. Raising it changes how fast everything grows, so growth durations must be expressed in seconds and converted, never hard-coded as tick counts.

## Open question

Whether Phase 3 needs a closed-form growth solution (compute the end state directly from elapsed ticks) or whether bounded replay with a cap on offline time is enough. Recorded in `docs/overview.md` under open questions; the decision does not need making until offline progress is built, and this ADR's determinism guarantee is what keeps both options available.
