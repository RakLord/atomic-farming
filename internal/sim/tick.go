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
func (s *GameState) Tick() {
	if s == nil {
		return
	}
	s.advancePlots()
	s.Ticks++
}

// advancePlots grows every planted crop by one tick and rolls for death on any
// plant that just entered a new stage.
//
// Death is handled here rather than inside Grow so that it applies to every
// crop uniformly and a new crop cannot forget to implement it.
func (s *GameState) advancePlots() {
	if s.Grid == nil {
		return
	}
	base := s.baseGrowContext()
	for i := range s.Grid.Plots {
		plot := &s.Grid.Plots[i]
		if plot.Crop == nil {
			continue
		}
		ctx := base
		ctx.Pos = s.Grid.positionOf(i)
		ctx.Phenotype = plot.Phenotype
		ctx.Seed = plot.Seed

		before := plot.Growth.Stage
		plot.Growth = plot.Crop.Grow(ctx, plot.Growth)
		if plot.Growth.Stage > before && rollStageDeath(ctx, plot.Growth.Stage) {
			*plot = Plot{}
		}
	}
}

// baseGrowContext builds the per-tick invariant portion of GrowContext.
// Callers fill in Pos, Phenotype and Seed per plot. Modifiers is Normalized
// once here rather than once per crop.
func (s *GameState) baseGrowContext() GrowContext {
	return GrowContext{
		Grid:      newGridView(s.Grid),
		Tick:      s.Ticks,
		Modifiers: s.Modifiers.Normalized(),
		Layer:     s.Layer,
	}
}

// growContextFor builds the context for a single plot, for player actions that
// happen outside the tick loop.
func (s *GameState) growContextFor(p Position, plot *Plot) GrowContext {
	ctx := s.baseGrowContext()
	ctx.Pos = p
	ctx.Phenotype = plot.Phenotype
	ctx.Seed = plot.Seed
	return ctx
}
