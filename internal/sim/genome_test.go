package sim

import (
	"testing"

	"atomicfarming/internal/plant"
)

func TestNextPlantSeedIsDistinctAndReplayable(t *testing.T) {
	s := NewGameState()
	s.WorldSeed = 12345

	seen := map[uint64]bool{}
	var first []uint64
	for i := 0; i < 500; i++ {
		seed := s.NextPlantSeed()
		if seen[seed] {
			t.Fatalf("plant seed %d repeated after %d plantings", seed, i)
		}
		seen[seed] = true
		first = append(first, seed)
	}

	// Replaying the same run from the same world seed must reproduce the same
	// sequence, or offline tick replay would diverge from live play.
	replay := NewGameState()
	replay.WorldSeed = 12345
	for i, want := range first {
		if got := replay.NextPlantSeed(); got != want {
			t.Fatalf("planting %d: replay produced %d, want %d", i, got, want)
		}
	}
}

func TestDifferentWorldsGrowDifferentPlants(t *testing.T) {
	a, b := NewGameState(), NewGameState()
	a.WorldSeed, b.WorldSeed = 1, 2
	if a.NextPlantSeed() == b.NextPlantSeed() {
		t.Error("two worlds produced the same plant seed")
	}
}

func TestNewGameStateDrawsAWorldSeed(t *testing.T) {
	if NewGameState().WorldSeed == 0 {
		t.Error("a new save has no world seed; every player would farm identical plants")
	}
}

func TestLayerResetRerollsTheWorldWithoutTouchingTheClock(t *testing.T) {
	s := NewGameState()
	s.WorldSeed = 99
	s.PlantCounter = 40

	ApplyLayerReset(s, LayerField)

	if s.WorldSeed == 99 {
		t.Error("the world seed survived the reset; the next run would grow the same plants")
	}
	if s.WorldSeed == 0 {
		t.Error("the reset cleared the world seed entirely")
	}
	if s.PlantCounter != 0 {
		t.Errorf("PlantCounter = %d after reset, want 0", s.PlantCounter)
	}

	// The reset itself must be deterministic, so a replayed prestige lands on
	// the same world.
	again := NewGameState()
	again.WorldSeed = 99
	ApplyLayerReset(again, LayerField)
	if again.WorldSeed != s.WorldSeed {
		t.Error("the reset's new world seed is not reproducible")
	}
}

func TestPlantedGenomeAndSeedSurviveSaving(t *testing.T) {
	isolateSave(t)

	s := NewGameState()
	genome := plant.RandomGenome(77)
	pos := Position{X: 1, Y: 2}
	p, _ := s.Grid.At(pos)
	p.Crop = &testCrop{}
	p.Genome = genome
	p.Seed = s.NextPlantSeed()
	wantSeed := p.Seed

	if err := s.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, ok, err := Load()
	if err != nil || !ok {
		t.Fatalf("Load: ok=%v err=%v", ok, err)
	}

	loaded, _ := got.Grid.At(pos)
	if loaded.Genome != genome {
		t.Error("the plant's genome did not survive the round trip")
	}
	if loaded.Seed != wantSeed {
		t.Errorf("plant seed = %d, want %d — its future rolls would change", loaded.Seed, wantSeed)
	}
	if got.WorldSeed != s.WorldSeed || got.PlantCounter != s.PlantCounter {
		t.Error("world randomness did not survive the round trip")
	}
}

// TestLoadRepairsAGenomelessCrop covers saves written before the genome layer
// existed: the all-zero genome expresses as a plant with no stem at all, so it
// must be replaced rather than drawn.
func TestLoadRepairsAGenomelessCrop(t *testing.T) {
	isolateSave(t)
	writeRawSave(t, `{"version":1,"state":{"grid":{"w":1,"h":1,"plots":[{"kind":"test_crop","crop":{}}]}}}`)

	got, ok, err := Load()
	if err != nil || !ok {
		t.Fatalf("Load: ok=%v err=%v", ok, err)
	}
	p, _ := got.Grid.At(Position{})
	if p.Crop == nil {
		t.Fatal("the crop was lost on load")
	}
	if p.Genome.IsZero() {
		t.Error("the crop was left with an empty genome")
	}
	if p.Genome != plant.DefaultGenome() {
		t.Error("the crop was not repaired to the catalog default genome")
	}
}

func TestEmptyPlotCarriesNoGenome(t *testing.T) {
	isolateSave(t)
	s := NewGameState()
	if err := s.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, _, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for i, p := range got.Grid.Plots {
		if !p.IsEmpty() {
			t.Fatalf("plot %d is not empty", i)
		}
		if !p.Genome.IsZero() || p.Seed != 0 {
			t.Errorf("empty plot %d carries genome data", i)
		}
	}
}
