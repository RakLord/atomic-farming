package sim

import "testing"

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

func TestStartingGridSizeSpendsExtraPlotBudget(t *testing.T) {
	tests := []struct {
		extra int
		wantW int
		wantH int
	}{
		{extra: 0, wantW: 3, wantH: 3},
		{extra: 2, wantW: 3, wantH: 3}, // not enough for a whole column
		{extra: 3, wantW: 4, wantH: 3}, // one column
		{extra: 7, wantW: 4, wantH: 4}, // column (3) then row (4)
		{extra: 10000, wantW: MaxGridW, wantH: MaxGridH},
		{extra: -5, wantW: 3, wantH: 3}, // nonsense budget is inert
	}
	for _, tc := range tests {
		s := NewGameState()
		s.Modifiers.ExtraPlots = tc.extra
		w, h := StartingGridSize(s)
		if w != tc.wantW || h != tc.wantH {
			t.Errorf("ExtraPlots=%d: got %dx%d, want %dx%d", tc.extra, w, h, tc.wantW, tc.wantH)
		}
	}
}
