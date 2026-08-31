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
