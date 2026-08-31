package input

import (
	"testing"

	"atomicfarming/internal/sim"
	"atomicfarming/internal/ui"
)

func TestSelectPlotTogglesOnRepeatClick(t *testing.T) {
	s, u := sim.NewGameState(), ui.NewUIState()
	p := sim.Position{X: 1, Y: 1}

	SelectPlot(s, u, p, true)
	if !u.IsSelected(p) {
		t.Fatal("first click did not select the plot")
	}
	SelectPlot(s, u, p, true)
	if u.HasSelection {
		t.Error("second click on the same plot did not deselect it")
	}
}

func TestSelectPlotClearsWhenClickingOffTheFarm(t *testing.T) {
	s, u := sim.NewGameState(), ui.NewUIState()
	SelectPlot(s, u, sim.Position{X: 0, Y: 0}, true)

	SelectPlot(s, u, sim.Position{}, false)

	if u.HasSelection {
		t.Error("clicking off the farm did not clear the selection")
	}
}

func TestSelectPlotRejectsOutOfBoundsPositions(t *testing.T) {
	s, u := sim.NewGameState(), ui.NewUIState()
	// ok=true but the position is off the grid: the guard must still hold.
	SelectPlot(s, u, sim.Position{X: 99, Y: 99}, true)
	if u.HasSelection {
		t.Error("an out-of-bounds position was selected")
	}
}

func TestHoverTracksAndClears(t *testing.T) {
	s, u := sim.NewGameState(), ui.NewUIState()
	p := sim.Position{X: 2, Y: 0}

	Hover(s, u, p, true)
	if !u.IsHovered(p) {
		t.Fatal("hover was not recorded")
	}
	Hover(s, u, sim.Position{}, false)
	if u.HasHover {
		t.Error("hover was not cleared when the cursor left the farm")
	}
}
