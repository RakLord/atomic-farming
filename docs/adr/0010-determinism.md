# ADR 0010 — Determinism of random outcomes

**Status:** accepted.
**Date:** 2026-08-31.

## Context

ADR 0008 makes `for i := 0; i < n; i++ { s.Tick() }` a valid fast-forward, and that is how offline progress will be implemented. Random outcomes threaten it directly: harvest success, death, breeding, and mutation are all rolls, and a live RNG stream replays differently every time.

If replay diverges from live play, a player who closes the game gets different results than one who leaves it open — and worse, the divergence is silent.

## Decision

**1. Every gameplay roll derives from a persisted seed, never from a stream.**

```go
func Hash64(x uint64) uint64                              // splitmix64
func Roll(seed uint64, p Purpose, salt, n uint64) uint64  // → [0, n)
func Chance(seed uint64, p Purpose, salt uint64, bp int) bool
```

There is no RNG state to advance, so nothing can get out of step. The seed and salt fully determine the result.

**2. The hash is implemented in this repository, not taken from `math/rand`.**

Save-visible outcomes derive from this exact stream. Vendoring ten lines of splitmix64 means no toolchain update can ever shift it. `TestHash64MatchesSplitmix64Vectors` pins it to published vectors.

**3. `Purpose` tags decorrelate independent decisions.**

Without them, a plant's death roll would predict its harvest roll, since both would draw on the same seed and salt. Purpose values are effectively persisted through the outcomes they produce, so they are append-only.

**4. Gameplay maths is integer-only; float is for pixels.**

`math.Sin` and friends are not guaranteed bit-identical across platforms, so a float-derived death chance could differ between the desktop and WASM builds and desynchronise a replay. Probabilities are therefore integers in **basis points** (`BasisPoints = 10000`), and `Phenotype.Scaled` exists as the integer accessor gameplay code uses.

Visual geometry may use float freely: a sub-ULP difference in a leaf angle is invisible, and forbidding it would make the generator far harder to write. `Phenotype.Unit`, `Lerp`, and `UnitFloat` are the float accessors, documented as visual-only.

**5. Seeds are stamped, not derived on the fly.**

```go
GameState.WorldSeed    uint64  // roots this run's randomness
GameState.PlantCounter uint64  // makes each planting distinct
Plot.Seed              uint64  // stamped at planting, never changes
```

`NextPlantSeed` combines the world seed with the counter. Every roll a plant ever makes derives from its own `Plot.Seed`, so its outcomes are fixed the moment it is planted and are identical whether the intervening ticks were played or replayed.

**6. The world seed is drawn from the clock once and re-derived thereafter.**

`NewGameState` reads `time.Now()` — the only place the clock touches simulation state — so two players do not farm identical plants. A layer reset re-derives it with `Hash64(WorldSeed)` rather than redrawing from the clock, so the prestige itself stays deterministic and a replayed run lands on the same world.

## Consequences

**Wins**
- Offline progress can reuse the live tick path instead of approximating it.
- A bug report reproduces exactly from a save file.
- Nothing to serialise but two integers.

**Costs**
- Probabilities in basis points are less readable than floats, and every new random gameplay outcome has to remember to route through `Chance`/`Roll` and pick a fresh salt.
- The modulo in `Roll` carries a bias of order n/2^64 — under one part in 10^15 for any n this game uses, and taken deliberately for simplicity.

## Related

`internal/plant/rng.go`, ADR 0008 (the tick model this protects), ADR 0009 (breeding and mutation are its main callers).
