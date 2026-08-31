// Package ui holds view state that never belongs in a save — hover,
// selection, and (later) which panels are open. The sim package never reads
// it, so gameplay stays reproducible from GameState alone.
package ui

import "atomicfarming/internal/sim"

// UIState is the player's transient view state.
type UIState struct {
	Hovered      sim.Position
	HasHover     bool
	Selected     sim.Position
	HasSelection bool
}

func NewUIState() *UIState { return &UIState{} }

// SetHover marks p as the plot under the cursor.
func (u *UIState) SetHover(p sim.Position) {
	u.Hovered, u.HasHover = p, true
}

func (u *UIState) ClearHover() { u.Hovered, u.HasHover = sim.Position{}, false }

// Select marks p as the plot the player has clicked. Selecting the already
// selected plot clears it, so a second click deselects.
func (u *UIState) Select(p sim.Position) {
	if u.HasSelection && u.Selected == p {
		u.ClearSelection()
		return
	}
	u.Selected, u.HasSelection = p, true
}

func (u *UIState) ClearSelection() { u.Selected, u.HasSelection = sim.Position{}, false }

// IsSelected reports whether p is the currently selected plot.
func (u *UIState) IsSelected(p sim.Position) bool { return u.HasSelection && u.Selected == p }

// IsHovered reports whether p is the plot under the cursor.
func (u *UIState) IsHovered(p sim.Position) bool { return u.HasHover && u.Hovered == p }
