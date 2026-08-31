package sim

import (
	"errors"

	"atomicfarming/internal/bignum"
	"atomicfarming/internal/plant"
)

// Value tuning. Unit value is shaped by two genes: Quality raises it directly,
// Density raises it far harder — density is the stat the whole game climbs
// toward, so it must be the one worth chasing.
const (
	// QualityValueBonusPermille is the extra unit value at maximum Quality.
	QualityValueBonusPermille = 1000
	// DensityValueBonusPermille is the extra unit value at maximum Density.
	DensityValueBonusPermille = 4000

	MinYieldUnits = 1
	MaxYieldUnits = 8

	// Seed return is rolled twice, each at a chance set by Yield Amount. The
	// range is deliberately centred slightly below one seed per harvest for an
	// average plant, so a farm slowly runs down and the shop stays relevant.
	MinSeedReturnBP = 2500
	MaxSeedReturnBP = 6500
	SeedReturnRolls = 2

	// SelfSeedMutationPPM is the baseline chance, in parts per million, that a
	// self-seeded seed carries a copy error — about one in ten thousand.
	//
	// A plant seeding itself is very nearly cloning. Variation is meant to come
	// from crossing, not from the ordinary act of harvesting.
	SelfSeedMutationPPM = 100
	// MaxMutabilityBoostPermille is what the Mutability gene is worth at its
	// maximum: ten times the baseline rate.
	MaxMutabilityBoostPermille = 9000
	// MaxSelfSeedMutationPPM caps the combined rate, so no stacking of genes
	// and upgrades can make every seed a mutant.
	MaxSelfSeedMutationPPM = 250_000
)

var (
	ErrNoPlant      = errors.New("sim: nothing planted here")
	ErrNotReady     = errors.New("sim: the crop is not ready to harvest")
	ErrNotHarvest   = errors.New("sim: this crop cannot be harvested")
	ErrNoSuchPlot   = errors.New("sim: no such plot")
	ErrNoSeed       = errors.New("sim: no such seed")
	ErrPlotOccupied = errors.New("sim: that plot is already planted")
	ErrTooExpensive = errors.New("sim: not enough cash")
	ErrLocked       = errors.New("sim: not unlocked yet")
)

// HarvestResult describes what a harvest produced, for the UI to report.
type HarvestResult struct {
	// Success is false when the harvest-chance roll failed. The plot is still
	// cleared; the crop was lost.
	Success   bool
	Units     int
	Cash      bignum.Decimal
	Seeds     int
	Cleared   bool
	Strain    NamedStrain
	HasStrain bool
	// NewStrain is true when this harvest was the strain's first discovery.
	NewStrain bool
}

// UnitValue is what one unit of a plant's produce sells for.
func UnitValue(p plant.Phenotype, mods GlobalModifiers) bignum.Decimal {
	quality := 1000 + int(p.Get(plant.GeneYieldQuality))*QualityValueBonusPermille/255
	density := 1000 + int(p.Get(plant.GeneDensity))*DensityValueBonusPermille/255
	v := bignum.FromInt(quality).MulInt(density).DivInt(1000).DivInt(1000)
	return v.Mul(mods.SellPriceMul)
}

// YieldUnits is how many units a harvest produces.
func YieldUnits(p plant.Phenotype, mods GlobalModifiers) int {
	units := p.Scaled(plant.GeneYieldAmount, MinYieldUnits, MaxYieldUnits)
	units = units * modifierPermille(mods.YieldMul) / 1000
	if units < 1 {
		units = 1
	}
	return units
}

// Harvest gathers the ready crop at p.
//
// It is player-initiated and so lives outside Tick — but every roll it makes
// still derives from the plot's stamped seed, so a harvest is decided by the
// plant rather than by when the player happened to click.
func (s *GameState) Harvest(p Position) (HarvestResult, error) {
	plot, ok := s.Grid.At(p)
	if !ok {
		return HarvestResult{}, ErrNoSuchPlot
	}
	if plot.Crop == nil {
		return HarvestResult{}, ErrNoPlant
	}
	if !plot.Growth.Ready {
		return HarvestResult{}, ErrNotReady
	}
	crop, ok := plot.Crop.(Harvestable)
	if !ok {
		return HarvestResult{}, ErrNotHarvest
	}

	ctx := s.growContextFor(p, plot)
	kind, genome, pheno, seed := plot.Crop.Kind(), plot.Genome, plot.Phenotype, plot.Seed

	var result HarvestResult
	if strain, isNew := s.DiscoverPlant(kind, genome, pheno); strain.ID != "" {
		result.Strain, result.HasStrain, result.NewStrain = strain, true, isNew
	}

	// The harvest roll is salted by the run's tick count so a plot that
	// regrows is not condemned to repeat its first result forever.
	chance := pheno.Scaled(plant.GeneHarvestChance, 0, plant.BasisPoints)
	if !plant.Chance(seed, plant.PurposeHarvest, s.Ticks, chance) {
		*plot = Plot{}
		result.Cleared = true
		return result, nil
	}

	yield, next, cleared := crop.Harvest(ctx, plot.Growth)
	result.Success = true
	result.Units = yield.Amount
	result.Cash = yield.Value
	result.Cleared = cleared

	s.Cash = s.Cash.Add(yield.Value)
	result.Seeds = s.returnSeeds(kind, genome, pheno, seed)

	if cleared {
		*plot = Plot{}
	} else {
		plot.Growth = next
	}
	return result, nil
}

// SelfSeedMutationPPMFor is a plant's chance of a copy error when it seeds
// itself, in parts per million.
//
// The baseline is near-never. The Mutability gene is worth up to ten times
// that, and irradiation upgrades multiply it further, so drift is something a
// player deliberately buys into rather than something that happens to them.
func SelfSeedMutationPPMFor(p plant.Phenotype, mods GlobalModifiers) int {
	mutability := 1000 + int(p.Get(plant.GeneMutability))*MaxMutabilityBoostPermille/255
	ppm := SelfSeedMutationPPM * mutability / 1000
	ppm = ppm * modifierPermille(mods.MutationRateMul) / 1000
	if ppm < 0 {
		return 0
	}
	if ppm > MaxSelfSeedMutationPPM {
		return MaxSelfSeedMutationPPM
	}
	return ppm
}

// returnSeeds gives back seeds carrying the harvested plant's genome.
//
// A seed is very nearly a clone: each gets one chance of a single-step copy
// error, and otherwise comes back identical. Rolling every allele instead —
// at any rate high enough to ever fire — left the barn full of near-identical
// lines that were impossible to tell apart or choose between.
// See docs/adr/0014-self-seeding-is-cloning.md.
func (s *GameState) returnSeeds(kind CropKind, genome plant.Genome, pheno plant.Phenotype, seed uint64) int {
	chance := pheno.Scaled(plant.GeneYieldAmount, MinSeedReturnBP, MaxSeedReturnBP)
	rate := SelfSeedMutationPPMFor(pheno, s.Modifiers.Normalized())

	given := 0
	for i := 0; i < SeedReturnRolls; i++ {
		if !plant.Chance(seed, plant.PurposeSpawn, s.Ticks*uint64(SeedReturnRolls)+uint64(i), chance) {
			continue
		}
		child, _ := plant.MutateOnce(genome, plant.Mix(seed, plant.PurposeMutation, s.Ticks+uint64(i)), rate)
		s.Inventory.Add(kind, child, 1)
		given++
	}
	return given
}

// StandardHarvest is the shared harvest body every crop can use: units and
// value from the genes, and regrowth decided by the Regrowth gene.
func StandardHarvest(ctx GrowContext, kind CropKind, g Growth, stages int) (Yield, Growth, bool) {
	units := YieldUnits(ctx.Phenotype, ctx.Modifiers)
	yield := Yield{
		Kind:   kind,
		Amount: units,
		Value:  UnitValue(ctx.Phenotype, ctx.Modifiers).MulInt(units),
	}

	if !Regrows(ctx.Phenotype) {
		return yield, Growth{}, true
	}
	// A perennial drops back to a middle stage rather than to bare soil, so
	// regrowing is meaningfully faster than replanting.
	next := Growth{Stage: regrowStage(stages)}
	return yield, next, false
}

// RegrowThreshold is the expressed Regrowth above which a plant is a perennial.
const RegrowThreshold = 128

// Regrows reports whether a plant survives its own harvest.
func Regrows(p plant.Phenotype) bool {
	return int(p.Get(plant.GeneRegrowth)) >= RegrowThreshold
}

func regrowStage(stages int) int {
	if stages <= 2 {
		return 0
	}
	return stages / 2
}
