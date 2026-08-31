// Package input turns resolved plot coordinates into UI state changes. It
// imports neither Ebitengine nor any layout geometry: the render layer maps
// screen coordinates to a sim.Position and calls in here, which keeps input
// intent testable without a window.
package input

import (
	"atomicfarming/internal/sim"
	"atomicfarming/internal/ui"
)

// Hover points the UI at the plot under the cursor. ok is false when the
// cursor is not over the farm at all.
func Hover(s *sim.GameState, u *ui.UIState, p sim.Position, ok bool) {
	if u == nil {
		return
	}
	if !ok || s == nil || !s.Grid.InBounds(p) {
		u.ClearHover()
		return
	}
	u.SetHover(p)
}

// SelectPlot handles a click. Clicking off the farm clears the selection;
// clicking the selected plot again deselects it.
//
// Planting, harvesting, and tool placement land in Phase 1 — this is the seam
// they hang off.
func SelectPlot(s *sim.GameState, u *ui.UIState, p sim.Position, ok bool) {
	if u == nil {
		return
	}
	if !ok || s == nil || !s.Grid.InBounds(p) {
		u.ClearSelection()
		return
	}
	u.Select(p)
}
