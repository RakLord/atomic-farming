# ADR 0003 — Large-number core and canonical save strings

**Status:** accepted.
**Date:** 2026-08-31.

## Context

An idle economy outgrows `float64`, which tops out near `1e308`. Atomic Farming's stated goal is matter dense enough to collapse the universe, so the currency and mass scalars are explicitly expected to run away.

The game ships on desktop and WASM from one codebase, so a browser-only library such as `break_infinity.js` would leave the desktop path unsolved and put a JS boundary in a hot path.

## Decision

**1. Use `internal/bignum`, copied from the reference project.**

A normalized scientific-decimal representation — sign, mantissa in `[1, 10)`, integer base-10 exponent — optimized for fast compare, multiply, and divide, with good-enough addition. Not arbitrary precision; an incremental economy does not need it.

**2. Compare with methods, not operators.**

Go cannot overload `<` or `>`. Use `Cmp`, `Eq`, `LT`, `LTE`, `GT`, `GTE`, `IsZero`, `Sign`.

**3. Growth-oriented scalars are `bignum.Decimal`; structural ones stay integers.**

`Cash`, yield values, and every multiplicative modifier are Decimals. Grid coordinates, growth stages, tick counters, plot counts, and the tick rate stay native integers — they drive discrete simulation flow, not runaway progression.

**4. Persist as canonical scientific strings** (`"2.5e3"`), so the save format does not depend on Go's in-memory representation and stays parseable from JavaScript.

**5. Display formatting is a separate layer.** `String()` stays canonical for save/load; `Format(DisplayShort, places)` is player-facing, so a display-style setting can change without touching save data or sim math.

## Consequences

- The economy is not capped by `float64`'s exponent range.
- Addition and subtraction lose precision when magnitudes differ by more than ~18 orders of magnitude. Acceptable, and consistent with the performance goal.
- More call sites use methods instead of operators.
