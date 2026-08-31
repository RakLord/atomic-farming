// Package render is the Ebitengine front end. It reads sim state and never
// mutates it outside the Tick call in Update, so gameplay stays reproducible
// from GameState alone.
package render

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"atomicfarming/internal/bignum"
	"atomicfarming/internal/input"
	"atomicfarming/internal/sim"
	"atomicfarming/internal/ui"
)

// SaveFn persists the current state. The caller owns how that happens so
// render does not depend on the save layer directly.
type SaveFn func() error

// autosaveIntervalTicks is how often state is written, in logical ticks.
const autosaveIntervalTicks = 300

type Game struct {
	state   *sim.GameState
	uiState *ui.UIState
	save    SaveFn
	lab     *ui.LabState
	sprites *SpriteCache

	sinceAutosave int
	// saveErr holds the most recent autosave failure so the player is told
	// their progress is not being written, rather than finding out on reload.
	saveErr error
}

func New(s *sim.GameState, u *ui.UIState, save SaveFn) *Game {
	return &Game{
		state:   s,
		uiState: u,
		save:    save,
		lab:     ui.NewLabState(),
		sprites: NewSpriteCache(DefaultSpriteCacheSize),
	}
}

// OpenLab shows the genetics lab immediately, without waiting for a keypress.
func (g *Game) OpenLab() { g.lab.Open = true }

func (g *Game) Update() error {
	g.handleInput()

	// All logical state advances here and nowhere else. See
	// docs/adr/0008-tick-model.md. The lab's animation is view state, so it
	// advances separately and never touches the simulation.
	g.state.Tick()
	g.lab.Tick()
	g.uiState.TickNotice()

	g.sinceAutosave++
	if g.sinceAutosave >= autosaveIntervalTicks {
		g.sinceAutosave = 0
		g.autosave()
	}
	return nil
}

func (g *Game) autosave() {
	if g.save == nil {
		return
	}
	if err := g.save(); err != nil {
		if g.saveErr == nil {
			log.Printf("autosave failed: %v", err)
		}
		g.saveErr = err
		return
	}
	g.saveErr = nil
}

func (g *Game) handleInput() {
	if g.lab.Open {
		g.handleLabInput()
		return
	}
	// The lab is a development tool for tuning the procedural generator, and
	// the seed of the eventual in-game genetics screen.
	if inpututil.IsKeyJustPressed(ebiten.KeyL) {
		g.lab.Open = true
		return
	}

	if g.uiState.SeedIndexOpen {
		g.handleSeedIndexInput()
		return
	}

	mx, my := ebiten.CursorPosition()
	pos, onFarm := cellAt(g.state.Grid, mx, my)
	input.Hover(g.state, g.uiState, pos, onFarm)

	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return
	}
	if mx >= panelX {
		g.handlePanelClick(mx, my)
		return
	}
	input.ClickPlot(g.state, g.uiState, pos, onFarm)
}

func (g *Game) buySeed(id sim.SeedOfferID) { input.BuySeed(g.state, g.uiState, id) }
func (g *Game) buyUnlock(id sim.UnlockID)  { input.BuyUnlock(g.state, g.uiState, id) }
func (g *Game) selectSeed(index int)       { input.SelectSeed(g.state, g.uiState, index) }
func (g *Game) discardSeed(index int)      { input.DiscardSeed(g.state, g.uiState, index) }

func (g *Game) clickSeedGroup(group sim.SeedGroup) {
	input.ClickSeedGroup(g.state, g.uiState, group)
}

// handleSeedIndexInput drives the seed picker. It runs instead of farm input
// while the index is open, so a click meant for one of its buttons cannot also
// sow the plot behind it.
func (g *Game) handleSeedIndexInput() {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.uiState.CloseSeedIndex()
		return
	}
	if _, wheel := ebiten.Wheel(); wheel != 0 {
		step := -1
		if wheel < 0 {
			step = 1
		}
		total := len(g.state.Inventory.StacksOfKind(g.uiState.SeedIndexKind))
		g.uiState.ScrollSeedIndex(step, total, indexRows)
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		g.handleSeedIndexClick(mx, my)
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(colorBG)
	g.drawHeader(screen)
	g.drawFarm(screen)
	g.drawPanel(screen)
	if g.uiState.SeedIndexOpen {
		g.drawSeedIndex(screen)
	}
	if g.lab.Open {
		g.drawLab(screen)
	}
}

// Layout pins the logical resolution; Ebitengine scales it to the window.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenW, screenH
}

func (g *Game) drawHeader(dst *ebiten.Image) {
	fillRect(dst, 0, 0, screenW, headerH, colorHeaderBG)
	fillRect(dst, 0, headerH-1, screenW, 1, colorDivider)

	drawText(dst, "ATOMIC FARMING", fontTitle, 20, 13, colorText)
	drawText(dst, "$"+g.state.Cash.Format(bignum.DisplayShort, 2), fontDisplay, 240, 8, colorCash)

	if g.saveErr != nil {
		drawTextRight(dst, "SAVE FAILING", fontBody, screenW-20, 17, colorWarning)
		return
	}
	drawTextRight(dst, plural(g.state.DiscoveredCount(), "strain", "strains")+" found",
		fontBody, screenW-20, 17, colorTextMuted)
}

func (g *Game) drawFarm(dst *ebiten.Image) {
	grid := g.state.Grid
	cell, _, _ := gridGeometry(grid)
	if cell <= 0 {
		return
	}
	for y := 0; y < grid.H; y++ {
		for x := 0; x < grid.W; x++ {
			g.drawPlot(dst, sim.Position{X: x, Y: y}, cell)
		}
	}
}

func (g *Game) drawPlot(dst *ebiten.Image, p sim.Position, cell int) {
	x, y, w, h := cellRect(g.state.Grid, p.X, p.Y)
	plot, _ := g.state.Grid.At(p)

	soil := colorSoil
	if g.uiState.IsHovered(p) {
		soil = colorPlotHover
	}
	fillRect(dst, x, y, w, h, soil)

	if plot != nil && plot.Crop == nil {
		// Furrows: two darker bands so bare soil reads as tilled rather than
		// as a flat swatch. They are hidden once something is growing.
		furrow := h / 3
		for i := 1; i <= 2; i++ {
			fillRect(dst, x+4, y+furrow*i-1, w-8, 2, colorSoilEdge)
		}
	} else if plot != nil {
		maturity := plot.Growth.Maturity(plot.Crop.Stages())
		g.sprites.DrawPlant(dst, plot.Phenotype, maturity, x, y, w, h)
	}

	strokeRect(dst, x, y, w, h, 1, colorSoilEdge)
	if plot != nil && plot.Growth.Ready {
		strokeRect(dst, x+1, y+1, w-2, h-2, 2, colorReady)
	}
	if g.uiState.IsSelected(p) {
		strokeRect(dst, x, y, w, h, 3, colorPlotPick)
	}
}
