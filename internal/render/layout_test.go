package render

import (
	"testing"

	"atomicfarming/internal/sim"
)

// farmSizes spans the design range: the starting farm, an in-between size,
// the maximum, and a non-square farm.
var farmSizes = [][2]int{{3, 3}, {5, 7}, {8, 8}, {sim.MaxGridW, sim.MaxGridH}}

func TestCellAtInvertsCellRect(t *testing.T) {
	for _, size := range farmSizes {
		g := sim.NewGrid(size[0], size[1])
		for y := 0; y < g.H; y++ {
			for x := 0; x < g.W; x++ {
				rx, ry, w, h := cellRect(g, x, y)
				// Every corner and the centre of a plot must map back to it.
				probes := [][2]int{
					{rx, ry},
					{rx + w - 1, ry},
					{rx, ry + h - 1},
					{rx + w - 1, ry + h - 1},
					{rx + w/2, ry + h/2},
				}
				for _, p := range probes {
					got, ok := cellAt(g, p[0], p[1])
					if !ok {
						t.Fatalf("%dx%d farm: cellAt(%d,%d) missed plot (%d,%d)", g.W, g.H, p[0], p[1], x, y)
					}
					if got.X != x || got.Y != y {
						t.Errorf("%dx%d farm: cellAt(%d,%d) = (%d,%d), want (%d,%d)",
							g.W, g.H, p[0], p[1], got.X, got.Y, x, y)
					}
				}
			}
		}
	}
}

func TestCellAtRejectsCoordinatesOffTheFarm(t *testing.T) {
	g := sim.NewGrid(sim.DefaultGridW, sim.DefaultGridH)
	cell, ox, oy := gridGeometry(g)

	offGrid := [][2]int{
		{ox - 1, oy},               // left of the farm
		{ox, oy - 1},               // above the farm
		{ox + cell*g.W, oy},        // right of the farm
		{ox, oy + cell*g.H},        // below the farm
		{10, 10},                   // in the header
		{panelX + 40, panelY + 40}, // in the side panel
	}
	for _, p := range offGrid {
		if got, ok := cellAt(g, p[0], p[1]); ok {
			t.Errorf("cellAt(%d,%d) = (%d,%d), want off-farm", p[0], p[1], got.X, got.Y)
		}
	}
}

func TestGridGeometryFitsInsideTheGridArea(t *testing.T) {
	for _, size := range farmSizes {
		g := sim.NewGrid(size[0], size[1])
		cell, ox, oy := gridGeometry(g)
		if cell < minCellSize || cell > maxCellSize {
			t.Errorf("%dx%d farm: cell = %d, outside [%d,%d]", g.W, g.H, cell, minCellSize, maxCellSize)
		}
		if ox < gridAreaX || oy < gridAreaY {
			t.Errorf("%dx%d farm: origin (%d,%d) starts before the grid area", g.W, g.H, ox, oy)
		}
		if right, bottom := ox+cell*g.W, oy+cell*g.H; right > gridAreaX+gridAreaW || bottom > gridAreaY+gridAreaH {
			t.Errorf("%dx%d farm: extends to (%d,%d), past the grid area", g.W, g.H, right, bottom)
		}
	}
}

func TestGridGeometryHandlesAnEmptyFarm(t *testing.T) {
	for _, g := range []*sim.Grid{nil, sim.NewGrid(0, 0)} {
		if cell, _, _ := gridGeometry(g); cell != 0 {
			t.Errorf("empty farm: cell = %d, want 0", cell)
		}
		if _, ok := cellAt(g, 100, 100); ok {
			t.Error("empty farm: cellAt found a plot")
		}
	}
}

// TestPlotsGrowIntoTheAvailableSpace guards the reason geometry is computed
// rather than constant: a small farm must use big plots and a large farm
// small ones.
func TestPlotsGrowIntoTheAvailableSpace(t *testing.T) {
	small, _, _ := gridGeometry(sim.NewGrid(3, 3))
	large, _, _ := gridGeometry(sim.NewGrid(sim.MaxGridW, sim.MaxGridH))
	if small <= large {
		t.Errorf("3x3 plot size %d is not larger than %dx%d plot size %d",
			small, sim.MaxGridW, sim.MaxGridH, large)
	}
}
