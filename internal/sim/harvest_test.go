package sim

import (
	"testing"

	"atomicfarming/internal/bignum"
	"atomicfarming/internal/plant"
)

// readyPlot plants a crop and ticks until it can be harvested.
func readyPlot(t *testing.T, s *GameState, p Position, g plant.Genome) *Plot {
	t.Helper()
	plantTestCrop(s, p, g)
	for i := 0; i < 20000; i++ {
		s.Tick()
		plot, _ := s.Grid.At(p)
		if plot.Crop == nil {
			t.Fatal("the plant died before it was ready")
		}
		if plot.Growth.Ready {
			return plot
		}
	}
	t.Fatal("the plant never became ready")
	return nil
}

// reliableGenome always survives, always harvests, and never regrows, so a
// test measures the one thing it is about.
func reliableGenome(extra map[plant.GeneID]plant.Allele) plant.Genome {
	base := map[plant.GeneID]plant.Allele{
		plant.GeneVitality:      255,
		plant.GeneHarvestChance: 255,
		plant.GeneGrowthRate:    255,
		plant.GeneRegrowth:      0,
	}
	for k, v := range extra {
		base[k] = v
	}
	return genomeWith(base)
}

func TestHarvestRejectsPlotsThatAreNotReady(t *testing.T) {
	s := NewGameState()
	pos := Position{}

	if _, err := s.Harvest(pos); err != ErrNoPlant {
		t.Errorf("harvesting bare soil gave %v, want ErrNoPlant", err)
	}
	if _, err := s.Harvest(Position{X: 99}); err != ErrNoSuchPlot {
		t.Errorf("harvesting off the farm gave %v, want ErrNoSuchPlot", err)
	}

	plantTestCrop(s, pos, reliableGenome(nil))
	if _, err := s.Harvest(pos); err != ErrNotReady {
		t.Errorf("harvesting an immature plant gave %v, want ErrNotReady", err)
	}
	if plot, _ := s.Grid.At(pos); plot.Crop == nil {
		t.Error("a rejected harvest destroyed the plant")
	}
}

func TestHarvestPaysCashAndClearsAnAnnual(t *testing.T) {
	s := NewGameState()
	pos := Position{}
	readyPlot(t, s, pos, reliableGenome(nil))

	before := s.Cash
	res, err := s.Harvest(pos)
	if err != nil {
		t.Fatalf("Harvest: %v", err)
	}
	if !res.Success {
		t.Fatal("a plant with maximum harvest chance failed")
	}
	if res.Units < 1 {
		t.Errorf("Units = %d, want at least 1", res.Units)
	}
	if !s.Cash.GT(before) {
		t.Errorf("cash did not rise: %s then %s", before, s.Cash)
	}
	if !res.Cleared {
		t.Error("an annual did not clear its plot")
	}
	if plot, _ := s.Grid.At(pos); plot.Crop != nil {
		t.Error("the plot still holds a crop after an annual harvest")
	}
}

func TestPerennialRegrowsInsteadOfClearing(t *testing.T) {
	s := NewGameState()
	pos := Position{}
	readyPlot(t, s, pos, reliableGenome(map[plant.GeneID]plant.Allele{plant.GeneRegrowth: 255}))

	res, err := s.Harvest(pos)
	if err != nil {
		t.Fatalf("Harvest: %v", err)
	}
	if res.Cleared {
		t.Fatal("a perennial cleared its plot")
	}
	plot, _ := s.Grid.At(pos)
	if plot.Crop == nil {
		t.Fatal("the perennial was removed")
	}
	if plot.Growth.Ready {
		t.Error("the perennial is immediately ready again; regrowth costs nothing")
	}
	if plot.Growth.Stage == 0 {
		t.Error("the perennial dropped to bare soil; regrowing should beat replanting")
	}
}

func TestDensityDominatesValue(t *testing.T) {
	mods := GlobalModifiers{}.Normalized()
	value := func(gene plant.GeneID, v plant.Allele) bignum.Decimal {
		g := genomeWith(map[plant.GeneID]plant.Allele{
			plant.GeneDensity: 0, plant.GeneYieldQuality: 0, gene: v,
		})
		return UnitValue(plant.ExpressFull(g), mods)
	}

	base := value(plant.GeneDensity, 0)
	quality := value(plant.GeneYieldQuality, 255)
	density := value(plant.GeneDensity, 255)

	if !quality.GT(base) {
		t.Error("Quality does not raise unit value")
	}
	if !density.GT(quality) {
		t.Errorf("Density (%s) does not beat Quality (%s); the stat the game climbs toward should pay best", density, quality)
	}
}

func TestSellPriceModifierScalesTheHarvest(t *testing.T) {
	g := reliableGenome(nil)
	p := plant.ExpressFull(g)

	base := UnitValue(p, GlobalModifiers{}.Normalized())
	boosted := UnitValue(p, GlobalModifiers{SellPriceMul: bignum.MustParse("3")}.Normalized())

	if want := base.MulInt(3); !boosted.Eq(want) {
		t.Errorf("with a 3x sell modifier value is %s, want %s", boosted, want)
	}
}

// harvestRepeatedly plants, ripens and gathers the same genome n times,
// returning how many distinct lines ended up in the barn.
func harvestRepeatedly(t *testing.T, s *GameState, g plant.Genome, n int) int {
	t.Helper()
	pos := Position{}
	for i := 0; i < n; i++ {
		readyPlot(t, s, pos, g)
		if _, err := s.Harvest(pos); err != nil {
			t.Fatalf("Harvest: %v", err)
		}
	}
	return len(s.Inventory.Stacks)
}

// TestSelfSeedingIsCloning is the regression test for the bug this model
// exists to fix: a barn full of near-identical singleton lines after a handful
// of harvests. Ordinary play must leave the gene pool where it started.
func TestSelfSeedingIsCloning(t *testing.T) {
	s := NewGameStateWithSeed(5150)
	s.Inventory = Inventory{}
	parent := reliableGenome(map[plant.GeneID]plant.Allele{
		plant.GeneYieldAmount: 255, // maximise seed return, so the sample is large
	})

	lines := harvestRepeatedly(t, s, parent, 300)
	if lines > 3 {
		t.Errorf("300 harvests produced %d distinct lines; ordinary play should barely drift at all", lines)
	}

	seeds := s.Inventory.Total()
	if seeds < 100 {
		t.Fatalf("only %d seeds returned across 300 harvests; the sample is too small to mean anything", seeds)
	}
	// Whatever did come back must be within one step of its parent.
	for _, stack := range s.Inventory.Stacks {
		steps := 0
		for i := 0; i < plant.GeneCount; i++ {
			for _, d := range []int{
				int(stack.Genome[i].A) - int(parent[i].A),
				int(stack.Genome[i].B) - int(parent[i].B),
			} {
				if d < 0 {
					d = -d
				}
				steps += d
			}
		}
		if steps > 1 {
			t.Errorf("a returned seed is %d steps from its parent; self-seeding should move one at most", steps)
		}
	}
	t.Logf("300 harvests, %d seeds returned, %d distinct lines", seeds, lines)
}

// TestIrradiationRestoresDrift: with the base rate near-never, the bred
// strains would be unreachable without a way to buy into mutation.
func TestIrradiationRestoresDrift(t *testing.T) {
	s := NewGameStateWithSeed(5150)
	s.Inventory = Inventory{}
	s.Unlocks[UnlockIrradiationI] = true
	s.Unlocks[UnlockIrradiationII] = true
	rebuildModifiers(s)

	parent := reliableGenome(map[plant.GeneID]plant.Allele{
		plant.GeneYieldAmount: 255,
		plant.GeneMutability:  255,
	})

	lines := harvestRepeatedly(t, s, parent, 300)
	if lines < 8 {
		t.Errorf("300 irradiated harvests produced only %d distinct lines; drift is not buyable", lines)
	}
	t.Logf("300 irradiated harvests produced %d distinct lines", lines)
}

func TestMutabilityAndIrradiationScaleTheRate(t *testing.T) {
	plain := plant.ExpressFull(genomeWith(map[plant.GeneID]plant.Allele{plant.GeneMutability: 0}))
	mutable := plant.ExpressFull(genomeWith(map[plant.GeneID]plant.Allele{plant.GeneMutability: 255}))
	none := GlobalModifiers{}.Normalized()
	boosted := GlobalModifiers{MutationRateMul: bignum.MustParse("144")}.Normalized()

	base := SelfSeedMutationPPMFor(plain, none)
	byGene := SelfSeedMutationPPMFor(mutable, none)
	byUpgrade := SelfSeedMutationPPMFor(plain, boosted)
	both := SelfSeedMutationPPMFor(mutable, boosted)

	if base != SelfSeedMutationPPM {
		t.Errorf("baseline rate is %d ppm, want %d", base, SelfSeedMutationPPM)
	}
	if byGene <= base {
		t.Errorf("Mutability did not raise the rate: %d vs %d ppm", byGene, base)
	}
	if byUpgrade <= base {
		t.Errorf("irradiation did not raise the rate: %d vs %d ppm", byUpgrade, base)
	}
	if both <= byGene || both <= byUpgrade {
		t.Errorf("the two do not compose: gene %d, upgrade %d, both %d ppm", byGene, byUpgrade, both)
	}
	if both > MaxSelfSeedMutationPPM {
		t.Errorf("combined rate %d ppm exceeds the ceiling %d", both, MaxSelfSeedMutationPPM)
	}

	// Even absurd stacking must not make every seed a mutant.
	absurd := SelfSeedMutationPPMFor(mutable, GlobalModifiers{MutationRateMul: bignum.MustParse("1e9")}.Normalized())
	if absurd != MaxSelfSeedMutationPPM {
		t.Errorf("an extreme multiplier gave %d ppm, want the ceiling %d", absurd, MaxSelfSeedMutationPPM)
	}
}

func TestSeedReturnAveragesBelowOneForAnOrdinaryPlant(t *testing.T) {
	// Slightly lossy on purpose: a farm that never needs to buy a seed makes
	// the shop pointless.
	const trials = 200
	total := 0
	for seed := uint64(1); seed <= trials; seed++ {
		s := NewGameState()
		s.WorldSeed = seed
		pos := Position{}
		readyPlot(t, s, pos, reliableGenome(map[plant.GeneID]plant.Allele{plant.GeneYieldAmount: 128}))
		res, err := s.Harvest(pos)
		if err != nil {
			t.Fatalf("Harvest: %v", err)
		}
		total += res.Seeds
	}
	avg := float64(total) / trials
	if avg <= 0.2 || avg >= 1.4 {
		t.Errorf("average seed return is %.2f per harvest; want a little under 1 so seeds slowly run down", avg)
	}
}

func TestAFailedHarvestPaysNothingAndClears(t *testing.T) {
	s := NewGameState()
	pos := Position{}
	// Zero harvest chance: the roll cannot succeed.
	readyPlot(t, s, pos, genomeWith(map[plant.GeneID]plant.Allele{
		plant.GeneVitality:      255,
		plant.GeneGrowthRate:    255,
		plant.GeneHarvestChance: 0,
	}))

	before := s.Cash
	res, err := s.Harvest(pos)
	if err != nil {
		t.Fatalf("Harvest: %v", err)
	}
	if res.Success {
		t.Fatal("a plant with zero harvest chance succeeded")
	}
	if !s.Cash.Eq(before) {
		t.Error("a failed harvest still paid out")
	}
	if plot, _ := s.Grid.At(pos); plot.Crop != nil {
		t.Error("a failed harvest left the crop standing")
	}
}
