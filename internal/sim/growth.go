package sim

import (
	"atomicfarming/internal/bignum"
	"atomicfarming/internal/plant"
)

// GrowthUnitsPerStage is the progress a plant must accumulate to advance one
// growth stage. Progress is an integer count, not a fraction of a stage, so
// growth replays exactly. See docs/adr/0010-determinism.md.
const GrowthUnitsPerStage = 1000

// Growth speed bounds, in units per tick. At the default 10 Hz tick rate the
// slowest plant takes 12.5 seconds per stage and the fastest 2 seconds.
const (
	MinGrowthUnitsPerTick = 8
	MaxGrowthUnitsPerTick = 50
)

// MaxStageDeathBP is the chance, in basis points, that a plant with the lowest
// possible Vitality dies when advancing a stage.
//
// The roll happens once per stage rather than once per tick. A per-tick roll
// compounds across the hundreds of ticks a plant lives, so even a single basis
// point would kill almost everything before maturity — a rate that reads as
// harmless and is not.
const MaxStageDeathBP = 300

// growthPerTick is how much progress a plant accumulates this tick.
func growthPerTick(ctx GrowContext) int {
	base := ctx.Phenotype.Scaled(plant.GeneGrowthRate, MinGrowthUnitsPerTick, MaxGrowthUnitsPerTick)
	units := base * modifierPermille(ctx.Modifiers.GrowthRateMul) / 1000
	if units < 1 {
		units = 1
	}
	return units
}

// stageDeathChanceBP is a plant's chance of dying as it advances a stage.
func stageDeathChanceBP(p plant.Phenotype) int {
	vitality := int(p.Get(plant.GeneVitality))
	return (255 - vitality) * MaxStageDeathBP / 255
}

// AdvanceGrowth is the standard growth step, shared by every crop so the
// accumulate-and-advance logic is written once. A crop calls it from Grow and
// may add behaviour of its own around it.
//
// Death is deliberately not handled here. It is a rule that applies to every
// plant, so the tick loop owns it — that way a new crop cannot forget to
// implement it, and Grow keeps a signature that only describes growth.
func AdvanceGrowth(ctx GrowContext, g Growth, stages int) Growth {
	if stages < 1 {
		stages = 1
	}
	if g.Ready {
		return g
	}

	g.Progress += growthPerTick(ctx)
	for g.Progress >= GrowthUnitsPerStage {
		g.Progress -= GrowthUnitsPerStage
		g.Stage++
		if g.Stage >= stages-1 {
			g.Stage = stages - 1
			g.Ready = true
			g.Progress = 0
			return g
		}
	}
	return g
}

// rollStageDeath reports whether a plant dies on entering the given stage.
// Salting by the stage makes it exactly one roll per stage, and the same roll
// on every replay.
func rollStageDeath(ctx GrowContext, stage int) bool {
	return plant.Chance(ctx.Seed, plant.PurposeDeath, uint64(stage), stageDeathChanceBP(ctx.Phenotype))
}

// Maturity is how far through its life a plant is, 0..1. The renderer draws
// the plant at this value; it is not gameplay state.
func (g Growth) Maturity(stages int) float64 {
	if stages < 1 {
		stages = 1
	}
	if g.Ready {
		return 1
	}
	if stages == 1 {
		return float64(g.Progress) / GrowthUnitsPerStage
	}
	per := 1 / float64(stages-1)
	return float64(g.Stage)*per + per*float64(g.Progress)/GrowthUnitsPerStage
}

// modifierPermille converts a multiplicative modifier into integer permille so
// that growth arithmetic stays integer-only.
//
// This is the one sanctioned bridge from Decimal into gameplay integer maths.
// It is safe because Decimal.Float64 is a math.Pow10 table lookup and a single
// IEEE-754 multiply — both bit-identical on every platform — and the scale and
// truncation that follow are exact. See docs/adr/0010-determinism.md.
func modifierPermille(d bignum.Decimal) int {
	if d.IsZero() {
		return 1000
	}
	v := d.Float64() * 1000
	if v < 0 {
		return 0
	}
	const maxPermille = 1 << 30
	if v > maxPermille {
		return maxPermille
	}
	return int(v)
}
