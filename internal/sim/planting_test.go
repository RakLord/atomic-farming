package sim

import (
	"testing"

	"atomicfarming/internal/plant"
)

func TestPlantingConsumesASeedAndStampsThePlot(t *testing.T) {
	s := NewGameState()
	g := plant.RandomGenome(7)
	s.Inventory = Inventory{}
	s.Inventory.Add(kindTestCrop, g, 2)

	pos := Position{X: 1, Y: 1}
	if err := s.PlantSeed(pos, 0); err != nil {
		t.Fatalf("PlantSeed: %v", err)
	}

	plot, _ := s.Grid.At(pos)
	if plot.Crop == nil {
		t.Fatal("nothing was planted")
	}
	if plot.Genome != g {
		t.Error("the plot did not take the seed's genome")
	}
	if plot.Seed == 0 {
		t.Error("the plot was not given a roll seed")
	}
	if (plot.Phenotype == plant.Phenotype{}) {
		t.Error("the plot's derived phenotype was not built")
	}
	if s.Inventory.Total() != 1 {
		t.Errorf("inventory holds %d seeds, want 1", s.Inventory.Total())
	}
}

func TestEveryPlantingGetsItsOwnSeed(t *testing.T) {
	s := NewGameState()
	g := plant.RandomGenome(8)
	s.Inventory = Inventory{}
	s.Inventory.Add(kindTestCrop, g, 4)

	seen := map[uint64]bool{}
	for i := 0; i < 4; i++ {
		pos := s.Grid.positionOf(i)
		if err := s.PlantSeed(pos, 0); err != nil {
			t.Fatalf("PlantSeed: %v", err)
		}
		plot, _ := s.Grid.At(pos)
		if seen[plot.Seed] {
			t.Fatalf("two plants share the roll seed %d; they would live identical lives", plot.Seed)
		}
		seen[plot.Seed] = true
	}
}

func TestPlantingRejectsBadRequests(t *testing.T) {
	s := NewGameState()
	s.Inventory = Inventory{}
	s.Inventory.Add(kindTestCrop, plant.DefaultGenome(), 1)

	if err := s.PlantSeed(Position{X: 99}, 0); err != ErrNoSuchPlot {
		t.Errorf("planting off the farm gave %v", err)
	}
	if err := s.PlantSeed(Position{}, 5); err != ErrNoSeed {
		t.Errorf("planting a non-existent stack gave %v", err)
	}
	if err := s.PlantSeed(Position{}, 0); err != nil {
		t.Fatalf("PlantSeed: %v", err)
	}
	if err := s.PlantSeed(Position{}, 0); err != ErrPlotOccupied {
		t.Errorf("planting into an occupied plot gave %v, want ErrPlotOccupied", err)
	}
}

func TestUprootClearsAPlot(t *testing.T) {
	s := NewGameState()
	pos := Position{}
	if err := s.Uproot(pos); err != ErrNoPlant {
		t.Errorf("uprooting bare soil gave %v", err)
	}
	plantTestCrop(s, pos, plant.DefaultGenome())
	if err := s.Uproot(pos); err != nil {
		t.Fatalf("Uproot: %v", err)
	}
	if plot, _ := s.Grid.At(pos); !plot.IsEmpty() {
		t.Error("the plot is still planted")
	}
}
