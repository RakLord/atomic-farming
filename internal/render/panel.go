package render

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"

	"atomicfarming/internal/sim"
)

// drawPanel renders the right-hand column. In the scaffold it reports what
// the simulation is doing so the pipeline is visibly working; the shop,
// upgrade tree, and codex land in later phases.
func (g *Game) drawPanel(dst *ebiten.Image) {
	fillRect(dst, panelX, panelY, panelW, panelH, colorPanelBG)
	fillRect(dst, panelX, panelY, 1, panelH, colorDivider)

	x := panelX + 24
	y := panelY + 24

	drawText(dst, "FARM", fontTitle, x, y, colorText)
	y += 40

	grid := g.state.Grid
	rows := [][2]string{
		{"Size", fmt.Sprintf("%d x %d", grid.W, grid.H)},
		{"Plots", fmt.Sprintf("%d", len(grid.Plots))},
		{"Planted", fmt.Sprintf("%d", plantedCount(grid))},
		{"Run", fmt.Sprintf("#%d", g.state.RunCount+1)},
		{"Layer", string(g.state.Layer)},
		{"Tick rate", fmt.Sprintf("%d Hz", g.state.TickRate)},
	}
	for _, row := range rows {
		drawText(dst, row[0], fontBody, x, y, colorTextMuted)
		drawTextRight(dst, row[1], fontBody, panelX+panelW-24, y, colorText)
		y += 24
	}

	y += 16
	fillRect(dst, x, y, panelW-48, 1, colorDivider)
	y += 20

	drawText(dst, "SELECTION", fontBody, x, y, colorTextMuted)
	y += 24
	if g.uiState.HasSelection {
		p := g.uiState.Selected
		drawText(dst, fmt.Sprintf("Plot (%d, %d)", p.X, p.Y), fontBody, x, y, colorText)
		y += 22
		if plot, ok := grid.At(p); ok && !plot.IsEmpty() {
			drawText(dst, string(plot.Crop.Kind()), fontBody, x, y, colorCash)
		} else {
			drawText(dst, "Empty — nothing to plant yet", fontSmall, x, y, colorTextMuted)
		}
	} else {
		drawText(dst, "Click a plot to select it", fontSmall, x, y, colorTextMuted)
	}

	// Footer: what this build is, so the empty panel is not mistaken for a bug.
	drawText(dst, "Phase 0 — scaffold", fontSmall, x, panelY+panelH-44, colorTextMuted)
	drawText(dst, "Seeds, growth and harvest land in Phase 1", fontSmall, x, panelY+panelH-28, colorTextMuted)
}

func plantedCount(g *sim.Grid) int {
	n := 0
	for _, p := range g.Plots {
		if !p.IsEmpty() {
			n++
		}
	}
	return n
}
