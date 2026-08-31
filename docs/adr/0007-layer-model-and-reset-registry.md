# ADR 0007 — Layer model and the declarative reset registry

**Status:** accepted.
**Date:** 2026-08-31.

## Context

The brief calls for nested prestige layers, and specifically for **persistency over prestige**: "some parts of the previous layers may be marked to not reset on resets when some upgrades are unlocked".

The reference project implements its reset as one function, `ResetGenesis`, that wipes about twenty fields inline and carries retention rules as conditionals in the middle of it:

```go
var researchSnapshot map[Element]int
if s.LaboratoryUpgrades[LabStableIsotope] > 0 {
    researchSnapshot = copyElementIntMap(s.Research)
}
// ... 40 more lines of field assignments ...
if researchSnapshot != nil {
    for e, v := range researchSnapshot { s.Research[e] = v * 30 / 100 }
}
```

This has two failure modes, both silent. Adding a field to `GameState` and forgetting to wipe it means it leaks across prestiges — a bug that only appears after a player prestiges, which is late. And as retention upgrades multiply, the reset function becomes the one place every persistence decision is tangled together.

Retrofitting structure onto a shipped reset is hard, because by then saves exist that depend on its exact behaviour. Do it before the first reset ships.

## Decision

**1. Reset behaviour is declared as data.**

```go
type ResetRule struct {
    ID     ResetRuleID
    Layer  Layer
    Fields []string          // GameState fields this rule owns
    Reset  func(s *GameState)
}
```

Rules register from `init()`. `ApplyLayerReset(s, layer)` runs every rule for that layer, then calls `rebuildModifiers`.

**2. Retention lives inside the rule that owns the state.**

A persistency-over-prestige upgrade becomes a change to one rule's `Reset` — which consults `s` for durable unlocks and decides how much of the previous run's value to keep — rather than another conditional in a monolith. The rule that owns `Grid` already does this: it sizes the new farm from the `ExtraPlots` budget that durable unlocks granted.

**3. Every `GameState` field must be accounted for, and a test proves it.**

`TestResetRulesCoverEveryField` reflects over `GameState` and fails unless each exported field is either claimed by exactly one rule per layer, or listed in `resetExemptFields` with a written reason. Adding a field without deciding what a prestige does to it is a test failure at the moment the field is added, not a bug report after players prestige.

The exempt list is a decision record, not a loophole — each entry says why the field is durable:

```go
var resetExemptFields = map[string]string{
    "Layer":        "the rung itself; changed by ascension, not by a reset",
    "TickRate":     "player setting, not run state",
    "Unlocks":      "durable progression; surviving prestige is the point",
    ...
}
```

The test also rejects a field that is both exempt and owned, a rule naming a field that does not exist, and two rules claiming the same field.

**4. `Layer` is a string with a registered order.**

`LayerField` is the base. `LayerOrder` is what the coverage test walks, so adding a layer constant without adding it there is caught, and adding it there without writing its rules fails the coverage check for every field.

## Consequences

**Wins**
- The set of state a prestige touches is enumerable and testable.
- Persistency upgrades are local changes.
- Forgetting a field is a compile-green, test-red event.

**Costs**
- `Fields` is a string list checked at test time, not compile time. A field rename that misses the rule fails the test rather than the build — acceptable, since the test runs in CI on every push.
- Slightly more ceremony than a single reset function for the four fields the scaffold has. The cost is flat; the benefit grows with every field added.

## Alternatives considered

- **Struct tags (`reset:"wipe"`) instead of a registry.** Rejected: a tag cannot express "keep 30% if this unlock is owned", which is the actual requirement.
- **One reset function per layer, no registry.** Rejected: that is the reference's design, and it is what this ADR exists to avoid.
