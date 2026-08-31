package main

import (
	"flag"
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"atomicfarming/internal/render"
	"atomicfarming/internal/sim"
	// Blank import runs each crop's init(), registering it with sim. Without
	// this, sim.Load cannot reconstruct planted crops from a save.
	_ "atomicfarming/internal/sim/crops"
	"atomicfarming/internal/ui"
)

// openLab starts with the genetics lab already showing. The lab is reachable
// in-game with the L key; this exists so it can be launched straight into for
// screenshots and for iterating on the procedural generator.
var openLab = flag.Bool("lab", false, "open the genetics lab at startup")

func main() {
	flag.Parse()

	state := loadOrNew()
	g := render.New(state, ui.NewUIState(), state.Save)
	if *openLab {
		g.OpenLab()
	}

	ebiten.SetWindowSize(1280, 720)
	ebiten.SetWindowTitle("Atomic Farming")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetTPS(state.TickRate)

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

// loadOrNew restores the save, falling back to a fresh farm. A corrupt or
// future-versioned save is reported and skipped rather than allowed to block
// startup.
func loadOrNew() *sim.GameState {
	s, ok, err := sim.Load()
	if err != nil {
		log.Printf("save load failed, starting fresh: %v", err)
		return sim.NewGameState()
	}
	if !ok {
		return sim.NewGameState()
	}
	return s
}
