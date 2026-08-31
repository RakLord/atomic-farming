package sim

import "atomicfarming/internal/plant"

// kindTestCrop is a crop kind that exists only for tests. Registering it here
// exercises the same registry path concrete crops use.
const kindTestCrop CropKind = "test_crop"

// testCropStages is fixed so growth tests can count ticks exactly.
const testCropStages = 4

type testCrop struct {
	// Watered is arbitrary per-crop config, present to prove that a crop's
	// own fields survive a save round-trip.
	Watered   bool `json:"watered,omitempty"`
	NumStages int  `json:"num_stages,omitempty"`
}

func (c *testCrop) Kind() CropKind { return kindTestCrop }

func (c *testCrop) Stages() int {
	if c.NumStages <= 0 {
		return testCropStages
	}
	return c.NumStages
}

// Ranges leaves every gene free, so tests control the phenotype directly
// through the genome rather than fighting a species window.
func (c *testCrop) Ranges() plant.SpeciesRanges { return plant.FullRange() }

func (c *testCrop) Grow(ctx GrowContext, g Growth) Growth {
	return AdvanceGrowth(ctx, g, c.Stages())
}

func (c *testCrop) Harvest(ctx GrowContext, g Growth) (Yield, Growth, bool) {
	return StandardHarvest(ctx, kindTestCrop, g, c.Stages())
}

func init() {
	RegisterCrop(kindTestCrop, func() Crop { return &testCrop{} })
	RegisterCropName(kindTestCrop, "Test Crop")
}

// plantTestCrop sows a test crop at p with the given genome, bypassing the
// inventory so a test can set up a plot directly.
func plantTestCrop(s *GameState, p Position, g plant.Genome) *Plot {
	plot, _ := s.Grid.At(p)
	*plot = Plot{Crop: &testCrop{}, Genome: g, Seed: s.NextPlantSeed()}
	plot.Express()
	return plot
}

// genomeWith returns a homozygous genome with the named genes overridden.
func genomeWith(overrides map[plant.GeneID]plant.Allele) plant.Genome {
	g := plant.DefaultGenome()
	for id, v := range overrides {
		g[id] = plant.GenePair{A: v, B: v}
	}
	return g
}
