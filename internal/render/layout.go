package render

import (
	"bytes"
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/gobold"

	"atomicfarming/internal/sim"
)

// Logical resolution. Ebitengine scales this to whatever window or canvas the
// player has; assets are authored at 1x. See docs/overview.md.
const (
	screenW = 1280
	screenH = 720

	headerH = 48

	gridAreaX = 0
	gridAreaY = headerH
	gridAreaW = 800
	gridAreaH = screenH - headerH

	panelX = gridAreaW
	panelY = headerH
	panelW = screenW - gridAreaW
	panelH = screenH - headerH

	// Plot size is computed from the farm's current dimensions rather than
	// fixed, because the farm grows — see docs/adr/0004-dynamic-grid-dimensions.md.
	minCellSize = 32
	maxCellSize = 160
	gridInset   = 24
)

var (
	colorBG        = color.RGBA{0x0c, 0x12, 0x0c, 0xff}
	colorHeaderBG  = color.RGBA{0x16, 0x1f, 0x16, 0xff}
	colorPanelBG   = color.RGBA{0x11, 0x19, 0x12, 0xff}
	colorDivider   = color.RGBA{0x27, 0x35, 0x27, 0xff}
	colorSoil      = color.RGBA{0x4a, 0x33, 0x22, 0xff}
	colorSoilEdge  = color.RGBA{0x2c, 0x1e, 0x14, 0xff}
	colorPlotHover = color.RGBA{0x7a, 0x59, 0x36, 0xff}
	colorPlotPick  = color.RGBA{0xc8, 0xa2, 0x4e, 0xff}
	colorText      = color.RGBA{0xef, 0xf3, 0xe8, 0xff}
	colorTextMuted = color.RGBA{0x7d, 0x8d, 0x79, 0xff}
	colorCash      = color.RGBA{0x8e, 0xd9, 0x6a, 0xff}
	colorWarning   = color.RGBA{0xe0, 0x7a, 0x4a, 0xff}
	colorReady     = color.RGBA{0xf0, 0xd0, 0x5a, 0xff}
	colorOverlay   = color.RGBA{0x00, 0x00, 0x00, 0xc0}

	colorRowBG       = color.RGBA{0x18, 0x22, 0x1a, 0xff}
	colorRowBuyable  = color.RGBA{0x1f, 0x30, 0x22, 0xff}
	colorRowSelected = color.RGBA{0x2e, 0x44, 0x30, 0xff}
	colorToggleOn    = color.RGBA{0x27, 0x42, 0x2b, 0xff}
	// Deliberately dim: an off toggle that merely looks slightly different
	// from an on one is not a toggle anybody can read at a glance.
	colorIconOff = color.RGBA{0x4a, 0x55, 0x49, 0xff}
	colorTooltip = color.RGBA{0x08, 0x0d, 0x09, 0xf2}
)

// rarityColors tint a strain's name so a find reads as a find.
var rarityColors = [4]color.RGBA{
	{0xef, 0xf3, 0xe8, 0xff}, // Common
	{0x6f, 0xc9, 0xe8, 0xff}, // Uncommon
	{0xc9, 0x8a, 0xf0, 0xff}, // Rare
	{0xf0, 0xb0, 0x3a, 0xff}, // Legendary
}

func rarityColor(r sim.StrainRarity) color.RGBA {
	if int(r) < len(rarityColors) {
		return rarityColors[r]
	}
	return colorText
}

var (
	uiFontSource = mustLoadUIFontSource()
	fontSmall    = &text.GoTextFace{Source: uiFontSource, Size: 12}
	fontBody     = &text.GoTextFace{Source: uiFontSource, Size: 14}
	fontTitle    = &text.GoTextFace{Source: uiFontSource, Size: 20}
	fontDisplay  = &text.GoTextFace{Source: uiFontSource, Size: 28}
)

func mustLoadUIFontSource() *text.GoTextFaceSource {
	src, err := text.NewGoTextFaceSource(bytes.NewReader(gobold.TTF))
	if err != nil {
		panic(fmt.Sprintf("render: load UI font: %v", err))
	}
	return src
}

// gridGeometry returns the pixel size of one plot and the top-left corner the
// farm is drawn from, sizing plots to fit the grid area and centring them.
// A zero cell size means there is nothing to draw.
func gridGeometry(g *sim.Grid) (cell, originX, originY int) {
	if g == nil || g.W <= 0 || g.H <= 0 {
		return 0, gridAreaX, gridAreaY
	}
	cell = (gridAreaW - 2*gridInset) / g.W
	if byHeight := (gridAreaH - 2*gridInset) / g.H; byHeight < cell {
		cell = byHeight
	}
	if cell > maxCellSize {
		cell = maxCellSize
	}
	if cell < minCellSize {
		cell = minCellSize
	}
	originX = gridAreaX + (gridAreaW-cell*g.W)/2
	originY = gridAreaY + (gridAreaH-cell*g.H)/2
	return cell, originX, originY
}

// cellRect returns the logical-pixel bounds of the plot at (cx, cy).
func cellRect(g *sim.Grid, cx, cy int) (x, y, w, h int) {
	cell, ox, oy := gridGeometry(g)
	return ox + cx*cell, oy + cy*cell, cell, cell
}

// cellAt returns the plot at logical coordinates (x, y). It is the exact
// inverse of cellRect, and reports false for coordinates off the farm.
func cellAt(g *sim.Grid, x, y int) (sim.Position, bool) {
	cell, ox, oy := gridGeometry(g)
	if cell <= 0 {
		return sim.Position{}, false
	}
	lx, ly := x-ox, y-oy
	if lx < 0 || ly < 0 {
		return sim.Position{}, false
	}
	p := sim.Position{X: lx / cell, Y: ly / cell}
	if !g.InBounds(p) {
		return sim.Position{}, false
	}
	return p, true
}
