# ADR 0002 — Versioned save envelope

**Status:** accepted.
**Date:** 2026-08-31.

## Context

`GameState` will change shape in every phase: crops in Phase 1, catalogs in Phase 2, automation in Phase 3, prestige state in Phase 5. Retrofitting a version field after players have saves means either migrating blind or breaking those saves.

## Decision

All saves are written through a versioned envelope under a single key:

```json
{ "version": 1, "state": { ... } }
```

- `version` is a monotonically increasing integer, starting at 1.
- `Load` reads the version first. An unknown version is an **error**, not a best-effort parse — the caller (`cmd/game/main.go`) logs it and boots fresh rather than guessing.
- Bump the version for any non-additive change to `state`. Purely additive fields with acceptable zero values do not need a bump.
- One key, so the save is atomic on both platforms. Splitting fields across keys invites partial-write corruption.

`Load` finishes with a `repairState` pass that restores invariants a hand-edited or truncated save could violate: non-nil maps, a grid whose plot count matches its dimensions, a valid layer, a positive tick rate. The tick loop is entitled to assume those hold.

`Save` stamps `LastSaveUnix` before writing. Nothing reads it yet; it is the seam offline progress needs.

## Consequences

- Migrations are a solved problem the first time one is needed.
- Every future save-shape change forces an explicit decision: additive, or bump plus migrate.
- Save tests must round-trip through the envelope, not marshal `GameState` directly.
- Tests that touch the desktop path must isolate it with `t.Setenv("XDG_CONFIG_HOME", t.TempDir())`, or they write into the developer's real config directory.
