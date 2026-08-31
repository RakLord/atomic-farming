package sim

import "atomicfarming/internal/plant"

// Farm dimensions. The grid is sized at runtime rather than by a compile-time
// constant because expanding the farm is a core mechanic — see
// docs/adr/0004-dynamic-grid-dimensions.md.
const (
	DefaultGridW = 3
	DefaultGridH = 3
	MaxGridW     = 12
	MaxGridH     = 12
)

// Position is a plot coordinate. Origin is the top-left plot.
type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Growth is a plot's mutable growth state. It lives on the Plot, not on the
// Crop, so Crop values stay immutable configuration shared across plots.
type Growth struct {
	// Stage is the current growth stage, 0..Crop.Stages()-1.
	Stage int `json:"stage,omitempty"`
	// Progress is ticks accumulated toward the next stage.
	Progress int `json:"progress,omitempty"`
	// Ready marks a crop that has finished growing and can be harvested.
	Ready bool `json:"ready,omitempty"`
}

// Plot is one farmable cell of the farm. The zero value is an empty plot.
type Plot struct {
	// Crop is the planted crop, or nil when the plot is empty.
	Crop   Crop
	Growth Growth
	// Genome is the individual variation of what is planted here. The Crop
	// says which species; the Genome says which strain of it.
	Genome plant.Genome
	// Seed is stamped when the crop is planted and never changes. Every
	// random outcome for this plant — whether a harvest succeeds, whether it
	// dies, what a cross produces — derives from it, so replaying ticks from
	// a save reproduces the same outcomes.
	// See docs/adr/0010-determinism.md.
	Seed uint64
}

// IsEmpty reports whether the plot has nothing planted in it.
func (p Plot) IsEmpty() bool { return p.Crop == nil }

// Grid is the farm: a runtime-dimensioned, row-major field of Plots.
// Plots has exactly W*H entries; the plot at (x, y) is Plots[y*W+x].
type Grid struct {
	W     int    `json:"w"`
	H     int    `json:"h"`
	Plots []Plot `json:"plots"`
}

// NewGrid returns an empty w×h farm. Negative dimensions are clamped to zero.
func NewGrid(w, h int) *Grid {
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	return &Grid{W: w, H: h, Plots: make([]Plot, w*h)}
}

// InBounds reports whether p addresses a plot on this grid.
func (g *Grid) InBounds(p Position) bool {
	return g != nil && p.X >= 0 && p.Y >= 0 && p.X < g.W && p.Y < g.H
}

// At returns a pointer to the plot at p for mutation by sim code. Callers
// outside sim read plots through GridView instead, which hands out copies.
func (g *Grid) At(p Position) (*Plot, bool) {
	if !g.InBounds(p) {
		return nil, false
	}
	i := p.Y*g.W + p.X
	if i < 0 || i >= len(g.Plots) {
		return nil, false
	}
	return &g.Plots[i], true
}

// Resize regrows the farm to w×h, preserving every plot that still falls
// inside the new bounds at its existing coordinate. Plots outside the new
// bounds are discarded. This is what farm expansion calls.
func (g *Grid) Resize(w, h int) {
	if g == nil {
		return
	}
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	next := make([]Plot, w*h)
	for y := 0; y < h && y < g.H; y++ {
		for x := 0; x < w && x < g.W; x++ {
			if i := y*g.W + x; i >= 0 && i < len(g.Plots) {
				next[y*w+x] = g.Plots[i]
			}
		}
	}
	g.W, g.H, g.Plots = w, h, next
}

// normalize repairs a grid whose Plots length disagrees with its dimensions.
// Load calls it so a truncated or hand-edited save cannot panic the tick loop.
func (g *Grid) normalize() {
	if g == nil {
		return
	}
	if g.W < 0 {
		g.W = 0
	}
	if g.H < 0 {
		g.H = 0
	}
	want := g.W * g.H
	switch {
	case len(g.Plots) == want:
		return
	case len(g.Plots) > want:
		g.Plots = g.Plots[:want]
	default:
		grown := make([]Plot, want)
		copy(grown, g.Plots)
		g.Plots = grown
	}
}

// gridView is the read-only wrapper handed to crops through GrowContext.
type gridView struct{ g *Grid }

func newGridView(g *Grid) GridView { return gridView{g: g} }

// PlotAt returns the plot by value so callers cannot mutate live grid data
// through the view. The Crop it carries is immutable configuration.
func (v gridView) PlotAt(p Position) (Plot, bool) {
	plot, ok := v.g.At(p)
	if !ok {
		return Plot{}, false
	}
	return *plot, true
}

func (v gridView) InBounds(p Position) bool { return v.g.InBounds(p) }

func (v gridView) Size() (w, h int) {
	if v.g == nil {
		return 0, 0
	}
	return v.g.W, v.g.H
}
