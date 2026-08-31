package sim

// Tick advances the simulation by one logical step.
//
// Load-bearing invariant: logical state changes only ever happen here, and
// Tick never reads the wall clock. That is what keeps
//
//	for i := 0; i < n; i++ { s.Tick() }
//
// a valid fast-forward, which is how offline progress will be implemented.
// See docs/adr/0008-tick-model.md.
//
// The scaffold advances the clock and nothing else. Phase 1 grows crops here,
// walking the grid with baseGrowContext.
func (s *GameState) Tick() {
	if s == nil {
		return
	}
	s.Ticks++
}

// baseGrowContext builds the per-tick invariant portion of GrowContext.
// Callers fill in Pos per plot. Modifiers is Normalized once here rather than
// once per crop.
func (s *GameState) baseGrowContext() GrowContext {
	return GrowContext{
		Grid:      newGridView(s.Grid),
		Tick:      s.Ticks,
		Modifiers: s.Modifiers.Normalized(),
		Layer:     s.Layer,
	}
}
