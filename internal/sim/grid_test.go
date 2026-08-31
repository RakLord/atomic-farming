package sim

import (
	"testing"

	"atomicfarming/internal/plant"
)

func TestNewGridAllocatesEveryPlot(t *testing.T) {
	g := NewGrid(4, 3)
	if len(g.Plots) != 12 {
		t.Fatalf("len(Plots) = %d, want 12", len(g.Plots))
	}
	for y := 0; y < 3; y++ {
		for x := 0; x < 4; x++ {
			if _, ok := g.At(Position{X: x, Y: y}); !ok {
				t.Errorf("At(%d,%d) not addressable", x, y)
			}
		}
	}
}

func TestGridAtRejectsOutOfBounds(t *testing.T) {
	g := NewGrid(2, 2)
	for _, p := range []Position{{X: -1}, {Y: -1}, {X: 2}, {Y: 2}, {X: 2, Y: 2}} {
		if _, ok := g.At(p); ok {
			t.Errorf("At(%+v) = ok, want out of bounds", p)
		}
	}
}

func TestGridResizePreservesPlotsByCoordinate(t *testing.T) {
	g := NewGrid(2, 2)
	// Mark each plot with a distinguishable crop configuration.
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			p, _ := g.At(Position{X: x, Y: y})
			p.Crop = &testCrop{NumStages: y*2 + x + 1}
		}
	}

	g.Resize(4, 4)

	if g.W != 4 || g.H != 4 || len(g.Plots) != 16 {
		t.Fatalf("after grow: W=%d H=%d len=%d, want 4/4/16", g.W, g.H, len(g.Plots))
	}
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			p, _ := g.At(Position{X: x, Y: y})
			crop, ok := p.Crop.(*testCrop)
			if !ok {
				t.Fatalf("plot (%d,%d) lost its crop on grow", x, y)
			}
			if want := y*2 + x + 1; crop.NumStages != want {
				t.Errorf("plot (%d,%d) NumStages = %d, want %d — plots shifted", x, y, crop.NumStages, want)
			}
		}
	}
	// Newly added plots are empty.
	if p, _ := g.At(Position{X: 3, Y: 3}); !p.IsEmpty() {
		t.Error("newly added plot is not empty")
	}
}

func TestGridResizeShrinkDiscardsOutOfBoundsPlots(t *testing.T) {
	g := NewGrid(3, 3)
	p, _ := g.At(Position{X: 0, Y: 0})
	p.Crop = &testCrop{NumStages: 7}
	edge, _ := g.At(Position{X: 2, Y: 2})
	edge.Crop = &testCrop{NumStages: 9}

	g.Resize(2, 2)

	if len(g.Plots) != 4 {
		t.Fatalf("len(Plots) = %d, want 4", len(g.Plots))
	}
	kept, _ := g.At(Position{X: 0, Y: 0})
	crop, ok := kept.Crop.(*testCrop)
	if !ok || crop.NumStages != 7 {
		t.Error("in-bounds plot was not preserved on shrink")
	}
	if _, ok := g.At(Position{X: 2, Y: 2}); ok {
		t.Error("discarded plot is still addressable")
	}
}

func TestGridNormalizeRepairsPlotSliceLength(t *testing.T) {
	short := &Grid{W: 3, H: 3, Plots: make([]Plot, 2)}
	short.normalize()
	if len(short.Plots) != 9 {
		t.Errorf("short grid: len(Plots) = %d, want 9", len(short.Plots))
	}

	long := &Grid{W: 2, H: 2, Plots: make([]Plot, 10)}
	long.normalize()
	if len(long.Plots) != 4 {
		t.Errorf("long grid: len(Plots) = %d, want 4", len(long.Plots))
	}
}

func TestGridViewHandsOutCopies(t *testing.T) {
	g := NewGrid(2, 2)
	p, _ := g.At(Position{X: 1, Y: 1})
	p.Growth = Growth{Stage: 2}

	v := newGridView(g)
	got, ok := v.PlotAt(Position{X: 1, Y: 1})
	if !ok {
		t.Fatal("PlotAt(1,1) not found")
	}
	got.Growth.Stage = 99

	if live, _ := g.At(Position{X: 1, Y: 1}); live.Growth.Stage != 2 {
		t.Errorf("mutating the view's copy changed live grid state: Stage = %d", live.Growth.Stage)
	}
	if w, h := v.Size(); w != 2 || h != 2 {
		t.Errorf("Size() = %d,%d, want 2,2", w, h)
	}
}

func TestStartingGridSizeAddsUnlockedColumnsAndRows(t *testing.T) {
	tests := []struct {
		cols, rows   int
		wantW, wantH int
	}{
		{0, 0, DefaultGridW, DefaultGridH},
		{1, 0, DefaultGridW + 1, DefaultGridH},
		{0, 2, DefaultGridW, DefaultGridH + 2},
		{3, 3, DefaultGridW + 3, DefaultGridH + 3},
		// Clamped to the biggest farm the renderer is designed for.
		{99, 99, MaxGridW, MaxGridH},
		// Nonsense values must not shrink the farm below its base.
		{-5, -5, DefaultGridW, DefaultGridH},
	}
	for _, tc := range tests {
		s := NewGameStateWithSeed(1)
		s.Modifiers.ExtraColumns, s.Modifiers.ExtraRows = tc.cols, tc.rows
		if w, h := StartingGridSize(s); w != tc.wantW || h != tc.wantH {
			t.Errorf("+%dc +%dr: got %dx%d, want %dx%d", tc.cols, tc.rows, w, h, tc.wantW, tc.wantH)
		}
	}
}

// TestSyncFarmSizeGrowsWithoutDisturbingCrops is the failure that would hurt
// most and stay invisible: widening the farm must not move or drop anything
// already in the ground.
func TestSyncFarmSizeGrowsWithoutDisturbingCrops(t *testing.T) {
	s := NewGameStateWithSeed(1)
	planted := map[Position]plant.Genome{}
	for i := range s.Grid.Plots {
		pos := s.Grid.PositionAt(i)
		g := plant.RandomGenome(uint64(i) + 1)
		plot := plantTestCrop(s, pos, g)
		plot.Growth = Growth{Stage: 2, Progress: 400}
		planted[pos] = g
	}

	s.Modifiers.ExtraColumns = 1
	syncFarmSize(s)

	if s.Grid.W != DefaultGridW+1 || s.Grid.H != DefaultGridH {
		t.Fatalf("farm is %dx%d, want %dx%d", s.Grid.W, s.Grid.H, DefaultGridW+1, DefaultGridH)
	}
	for pos, want := range planted {
		plot, ok := s.Grid.At(pos)
		if !ok || plot.Crop == nil {
			t.Fatalf("the crop at %+v was lost when the farm grew", pos)
		}
		if plot.Genome != want {
			t.Errorf("the crop at %+v changed genome; plots shifted", pos)
		}
		if plot.Growth != (Growth{Stage: 2, Progress: 400}) {
			t.Errorf("the crop at %+v lost its growth: %+v", pos, plot.Growth)
		}
	}
	// The new column is bare ground, not a copy of anything.
	for y := 0; y < s.Grid.H; y++ {
		if plot, _ := s.Grid.At(Position{X: DefaultGridW, Y: y}); !plot.IsEmpty() {
			t.Errorf("the new plot at column %d row %d is not empty", DefaultGridW, y)
		}
	}
}

func TestSyncFarmSizeNeverShrinksTheFarm(t *testing.T) {
	s := NewGameStateWithSeed(1)
	s.Grid.Resize(6, 5)
	plot, _ := s.Grid.At(Position{X: 5, Y: 4})
	plot.Crop = &testCrop{}

	// No expansion unlocks: entitlement is the base farm, well under 6x5.
	syncFarmSize(s)

	if s.Grid.W != 6 || s.Grid.H != 5 {
		t.Fatalf("farm shrank to %dx%d; that would destroy planted crops", s.Grid.W, s.Grid.H)
	}
	if got, _ := s.Grid.At(Position{X: 5, Y: 4}); got.Crop == nil {
		t.Error("the outermost crop was destroyed")
	}
}
