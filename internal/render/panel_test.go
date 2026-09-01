package render

import (
	"strings"
	"testing"

	"atomicfarming/internal/plant"
	"atomicfarming/internal/sim"
	"atomicfarming/internal/sim/crops"
	"atomicfarming/internal/ui"
)

func gameWithSeeds(t *testing.T, lines int) *Game {
	t.Helper()
	s := sim.NewGameStateWithSeed(1)
	s.Inventory = sim.Inventory{}
	for i := 0; i < lines; i++ {
		s.Inventory.Add(crops.KindStem, plant.RandomGenome(uint64(i)+1), i+1)
	}
	return New(s, ui.NewUIState(), func() error { return nil })
}

// TestSeedRowButtonsAppearOnlyWhenThereIsAChoice: a picker and a pick-for-me
// toggle both mean nothing when there is exactly one line.
func TestSeedRowButtonsAppearOnlyWhenThereIsAChoice(t *testing.T) {
	single := gameWithSeeds(t, 1).panelLayout()
	if len(single.seeds) != 1 {
		t.Fatalf("got %d seed rows, want 1", len(single.seeds))
	}
	if single.seeds[0].buttons {
		t.Error("a single-line row drew buttons that would do nothing")
	}

	many := gameWithSeeds(t, 4).panelLayout()
	if !many.seeds[0].buttons {
		t.Error("a multi-line row has no buttons")
	}
}

// TestSeedRowButtonsStayInsideTheirRow guards the layout: a button drawn
// outside its row, or overlapping the count, would be unclickable or ugly.
func TestSeedRowButtonsStayInsideTheirRow(t *testing.T) {
	l := gameWithSeeds(t, 4).panelLayout()
	row := l.seeds[0]

	for name, b := range map[string]rect{"index": row.index, "auto": row.auto} {
		if b.x < row.rect.x || b.x+b.w > row.rect.x+row.rect.w {
			t.Errorf("%s button escapes its row horizontally", name)
		}
		if b.y < row.rect.y || b.y+b.h > row.rect.y+row.rect.h {
			t.Errorf("%s button escapes its row vertically", name)
		}
	}
	if row.index.x+row.index.w > row.auto.x {
		t.Error("the two buttons overlap")
	}
	if row.rect.x+row.rect.w > panelX+panelW {
		t.Error("the seed row runs past the panel")
	}
}

// TestTooltipsNameTheButtonUnderTheCursor also proves the hit boxes match
// where the buttons are drawn, since both come from the same rects.
func TestTooltipsNameTheButtonUnderTheCursor(t *testing.T) {
	g := gameWithSeeds(t, 4)
	l := g.panelLayout()
	row := l.seeds[0]

	centre := func(r rect) (int, int) { return r.x + r.w/2, r.y + r.h/2 }

	mx, my := centre(row.index)
	label, _, ok := g.seedRowTooltip(l, mx, my)
	if !ok || !strings.Contains(label, "every line") {
		t.Errorf("index button tooltip = %q (ok=%v)", label, ok)
	}

	mx, my = centre(row.auto)
	label, _, ok = g.seedRowTooltip(l, mx, my)
	if !ok || !strings.Contains(label, "Auto-pick on") {
		t.Errorf("auto button tooltip with auto on = %q (ok=%v)", label, ok)
	}

	// The label must follow the state, or it would tell the player the opposite
	// of what clicking does.
	g.state.ToggleAutoSelectSeeds(crops.KindStem)
	label, _, ok = g.seedRowTooltip(l, mx, my)
	if !ok || !strings.Contains(label, "Auto-pick off") {
		t.Errorf("auto button tooltip with auto off = %q (ok=%v)", label, ok)
	}

	if _, _, ok := g.seedRowTooltip(l, row.rect.x+4, row.rect.y+4); ok {
		t.Error("the row body reported a tooltip")
	}
}

// TestInspectorClosesWhenItsPlantIsHarvested: the panel reads growth from the
// live plot, so an inspected crop that gets gathered would otherwise leave a
// ghost on screen.
func TestInspectorClosesWhenItsPlantIsHarvested(t *testing.T) {
	s := sim.NewGameStateWithSeed(3)
	pos := sim.Position{}
	if err := s.PlantSeed(pos, 0); err != nil {
		t.Fatalf("PlantSeed: %v", err)
	}
	plot, _ := s.Grid.At(pos)

	u := ui.NewUIState()
	u.InspectPlant(crops.KindStem, plot.Genome, pos)
	g := New(s, u, func() error { return nil })

	if in := g.inspectedPlant(); !in.ok || !in.hasGrowth {
		t.Fatal("the inspector cannot see the plant it is looking at")
	}

	if err := s.Uproot(pos); err != nil {
		t.Fatalf("Uproot: %v", err)
	}
	// The snapshot still renders, so the panel cannot crash mid-frame...
	if in := g.inspectedPlant(); !in.ok || in.hasGrowth {
		t.Error("a gone plant still reports growth")
	}
	// ...but the next input tick closes it rather than showing a ghost.
	g.handleInspectorInput()
	if u.InspectOpen {
		t.Error("the inspector stayed open over an empty plot")
	}
}

// TestInspectorUsesSpeciesRangesNotTheFullGenome is the reason this is not the
// lab: the lab expresses with ExpressFull, which would report a Stem's Growth
// Rate against a window it does not have.
func TestInspectorUsesSpeciesRangesNotTheFullGenome(t *testing.T) {
	s := sim.NewGameStateWithSeed(3)
	g := plant.DefaultGenome()
	g[plant.GeneGrowthRate] = plant.GenePair{A: 0, B: 0}

	u := ui.NewUIState()
	u.InspectSeedStack(sim.SeedStack{Kind: crops.KindStem, Genome: g, Count: 1})
	game := New(s, u, func() error { return nil })

	in := game.inspectedPlant()
	if !in.ok {
		t.Fatal("the inspector has nothing to show")
	}
	full := plant.ExpressFull(g).Get(plant.GeneGrowthRate)
	species := in.pheno.Get(plant.GeneGrowthRate)
	if species == full {
		t.Fatal("the inspector expressed with the full range; a Stem's window starts above zero")
	}
	if want := (&crops.Stem{}).Ranges()[plant.GeneGrowthRate].Min; species != uint8(want) {
		t.Errorf("Growth Rate reads %d, want the species floor %d", species, want)
	}
}

func TestCarrierCountFindsHiddenAlleles(t *testing.T) {
	g := plant.DefaultGenome()
	if got := carrierCount(g); got != 0 {
		t.Errorf("a homozygous genome reports %d carriers, want 0", got)
	}

	g[plant.GeneDensity] = plant.GenePair{A: 240, B: 40}
	g[plant.GeneStemHeight] = plant.GenePair{A: 100, B: 101} // below the noise floor
	if got := carrierCount(g); got != 1 {
		t.Errorf("carrierCount = %d, want 1 — a one-step difference is noise, not a hidden allele", got)
	}
}
