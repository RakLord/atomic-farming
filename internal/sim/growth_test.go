package sim

import (
	"testing"

	"atomicfarming/internal/bignum"
	"atomicfarming/internal/plant"
)

// growUntilReady ticks until the plot at p is ready or dies, returning the
// tick count and whether it survived.
func growUntilReady(s *GameState, p Position, limit int) (ticks int, ready bool) {
	for i := 0; i < limit; i++ {
		s.Tick()
		plot, _ := s.Grid.At(p)
		if plot.Crop == nil {
			return i + 1, false
		}
		if plot.Growth.Ready {
			return i + 1, true
		}
	}
	return limit, false
}

func TestAPlantGrowsToReady(t *testing.T) {
	s := NewGameState()
	pos := Position{X: 1, Y: 1}
	plantTestCrop(s, pos, genomeWith(map[plant.GeneID]plant.Allele{
		plant.GeneGrowthRate: 128,
		plant.GeneVitality:   255, // never dies, so this measures growth only
	}))

	ticks, ready := growUntilReady(s, pos, 5000)
	if !ready {
		t.Fatalf("the plant was not ready after %d ticks", ticks)
	}

	plot, _ := s.Grid.At(pos)
	if plot.Growth.Stage != testCropStages-1 {
		t.Errorf("final stage = %d, want %d", plot.Growth.Stage, testCropStages-1)
	}
}

// TestGrowthRateGeneChangesHowLongAPlantTakes is the check that the gene is
// actually wired to the clock, rather than growth being uniform.
func TestGrowthRateGeneChangesHowLongAPlantTakes(t *testing.T) {
	measure := func(rate plant.Allele) int {
		s := NewGameState()
		pos := Position{}
		plantTestCrop(s, pos, genomeWith(map[plant.GeneID]plant.Allele{
			plant.GeneGrowthRate: rate,
			plant.GeneVitality:   255,
		}))
		ticks, ready := growUntilReady(s, pos, 20000)
		if !ready {
			t.Fatalf("rate %d never matured", rate)
		}
		return ticks
	}

	slow, fast := measure(0), measure(255)
	if fast >= slow {
		t.Errorf("a maximum-growth plant took %d ticks and a minimum-growth one %d; the gene is not reaching growth", fast, slow)
	}
	// The gene's range is 8..50 units per tick, so the spread should be real.
	if slow < fast*3 {
		t.Errorf("growth spread is only %dx (%d vs %d ticks); the gene barely matters", slow/fast, slow, fast)
	}
}

func TestGrowthRateModifierSpeedsEveryPlantUp(t *testing.T) {
	measure := func(mul string) int {
		s := NewGameState()
		s.Modifiers.GrowthRateMul = bignum.MustParse(mul)
		pos := Position{}
		plantTestCrop(s, pos, genomeWith(map[plant.GeneID]plant.Allele{
			plant.GeneGrowthRate: 100,
			plant.GeneVitality:   255,
		}))
		ticks, ready := growUntilReady(s, pos, 20000)
		if !ready {
			t.Fatalf("multiplier %s never matured", mul)
		}
		return ticks
	}

	base, boosted := measure("1"), measure("4")
	if boosted >= base {
		t.Errorf("a 4x growth modifier took %d ticks against a baseline %d", boosted, base)
	}
}

func TestMaturityRisesMonotonicallyToOne(t *testing.T) {
	s := NewGameState()
	pos := Position{}
	plantTestCrop(s, pos, genomeWith(map[plant.GeneID]plant.Allele{
		plant.GeneGrowthRate: 60,
		plant.GeneVitality:   255,
	}))

	prev := -1.0
	for i := 0; i < 20000; i++ {
		s.Tick()
		plot, _ := s.Grid.At(pos)
		if plot.Crop == nil {
			t.Fatal("the plant died despite maximum vitality")
		}
		m := plot.Growth.Maturity(testCropStages)
		if m < prev {
			t.Fatalf("maturity fell from %.4f to %.4f", prev, m)
		}
		if m < 0 || m > 1 {
			t.Fatalf("maturity %.4f is outside [0,1]", m)
		}
		prev = m
		if plot.Growth.Ready {
			break
		}
	}
	if prev != 1 {
		t.Errorf("a ready plant has maturity %.4f, want 1", prev)
	}
}

func TestHighVitalityPlantsSurvive(t *testing.T) {
	s := NewGameState()
	for i := 0; i < len(s.Grid.Plots); i++ {
		plantTestCrop(s, s.Grid.positionOf(i), genomeWith(map[plant.GeneID]plant.Allele{
			plant.GeneVitality:   255,
			plant.GeneGrowthRate: 200,
		}))
	}
	for i := 0; i < 3000; i++ {
		s.Tick()
	}
	for i, plot := range s.Grid.Plots {
		if plot.Crop == nil {
			t.Fatalf("plot %d died despite maximum vitality", i)
		}
	}
}

// TestDeathIsRolledPerStageNotPerTick is the guard against the compounding
// mistake: a per-tick roll would kill nearly everything long before maturity,
// at a rate that looks harmless when written down.
func TestDeathIsRolledPerStageNotPerTick(t *testing.T) {
	const trials = 200
	survived := 0
	for seed := 0; seed < trials; seed++ {
		s := NewGameState()
		s.WorldSeed = uint64(seed) + 1
		pos := Position{}
		plantTestCrop(s, pos, genomeWith(map[plant.GeneID]plant.Allele{
			plant.GeneVitality:   0, // worst case: the maximum per-stage rate
			plant.GeneGrowthRate: 20,
		}))
		if _, ready := growUntilReady(s, pos, 20000); ready {
			survived++
		}
	}
	// Three stage transitions at MaxStageDeathBP (3%) leaves about 91%
	// surviving. A per-tick roll over hundreds of ticks would leave almost none.
	if survived < trials*3/4 {
		t.Errorf("only %d/%d of the frailest plants survived; death is compounding per tick, not per stage", survived, trials)
	}
}

func TestGrowthReplaysIdenticallyFromTheSameSeed(t *testing.T) {
	run := func() Growth {
		s := NewGameState()
		s.WorldSeed = 4242
		pos := Position{X: 2, Y: 2}
		plantTestCrop(s, pos, plant.RandomGenome(8))
		for i := 0; i < 200; i++ {
			s.Tick()
		}
		plot, _ := s.Grid.At(pos)
		return plot.Growth
	}
	if a, b := run(), run(); a != b {
		t.Errorf("two runs from the same world seed diverged: %+v vs %+v", a, b)
	}
}

func TestEmptyPlotsAreLeftAlone(t *testing.T) {
	s := NewGameState()
	for i := 0; i < 50; i++ {
		s.Tick()
	}
	for i, plot := range s.Grid.Plots {
		if !plot.IsEmpty() {
			t.Fatalf("plot %d sprouted something on its own", i)
		}
	}
	if s.Ticks != 50 {
		t.Errorf("Ticks = %d, want 50", s.Ticks)
	}
}
