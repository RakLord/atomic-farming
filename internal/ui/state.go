// Package ui holds view state that never belongs in a save — hover,
// selection, and (later) which panels are open. The sim package never reads
// it, so gameplay stays reproducible from GameState alone.
package ui

import (
	"atomicfarming/internal/plant"
	"atomicfarming/internal/sim"
)

// NoticeLifetimeTicks is how long an action message stays on screen.
const NoticeLifetimeTicks = 40

// UIState is the player's transient view state.
type UIState struct {
	Hovered      sim.Position
	HasHover     bool
	Selected     sim.Position
	HasSelection bool

	// The seed queued for planting, held by identity rather than by inventory
	// index: stacks shift as seeds are spent, and an index would silently come
	// to mean a different seed.
	SeedKind   sim.CropKind
	SeedGenome plant.Genome
	HasSeed    bool

	// The seed index: the picker for choosing between lines of one species.
	SeedIndexOpen   bool
	SeedIndexKind   sim.CropKind
	SeedIndexScroll int

	// The plant inspector: a read-only look at one genome. The genome and
	// species are snapshotted so the panel still renders if the plot is
	// harvested mid-look; InspectPos is kept so a planted crop's growth can be
	// read live and watched as it ripens.
	InspectOpen     bool
	InspectKind     sim.CropKind
	InspectGenome   plant.Genome
	InspectPos      sim.Position
	InspectFromPlot bool

	// Notice is the last action's result, shown briefly.
	Notice      string
	noticeTicks int
}

// OpenSeedIndex shows the picker for one species.
func (u *UIState) OpenSeedIndex(kind sim.CropKind) {
	u.SeedIndexOpen, u.SeedIndexKind, u.SeedIndexScroll = true, kind, 0
}

func (u *UIState) CloseSeedIndex() { u.SeedIndexOpen = false }

// ScrollSeedIndex moves the picker's viewport, clamped to the row count.
func (u *UIState) ScrollSeedIndex(delta, rows, visible int) {
	u.SeedIndexScroll += delta
	max := rows - visible
	if max < 0 {
		max = 0
	}
	if u.SeedIndexScroll > max {
		u.SeedIndexScroll = max
	}
	if u.SeedIndexScroll < 0 {
		u.SeedIndexScroll = 0
	}
}

func NewUIState() *UIState { return &UIState{} }

// SelectSeed queues a seed for planting.
func (u *UIState) SelectSeed(stack sim.SeedStack) {
	if u.HasSeed && u.SeedKind == stack.Kind && u.SeedGenome == stack.Genome {
		u.ClearSeed()
		return
	}
	u.SeedKind, u.SeedGenome, u.HasSeed = stack.Kind, stack.Genome, true
}

func (u *UIState) ClearSeed() {
	u.SeedKind, u.SeedGenome, u.HasSeed = "", plant.Genome{}, false
}

// IsSeedSelected reports whether stack is the queued seed.
func (u *UIState) IsSeedSelected(stack sim.SeedStack) bool {
	return u.HasSeed && u.SeedKind == stack.Kind && u.SeedGenome == stack.Genome
}

// SeedIndex resolves the queued seed to its current inventory position,
// returning -1 when it is no longer held.
func (u *UIState) SeedIndex(inv *sim.Inventory) int {
	if !u.HasSeed || inv == nil {
		return -1
	}
	for i, stack := range inv.Stacks {
		if stack.Kind == u.SeedKind && stack.Genome == u.SeedGenome {
			return i
		}
	}
	return -1
}

// InspectPlant opens the inspector on a crop growing at pos.
func (u *UIState) InspectPlant(kind sim.CropKind, g plant.Genome, pos sim.Position) {
	u.InspectOpen, u.InspectKind, u.InspectGenome = true, kind, g
	u.InspectPos, u.InspectFromPlot = pos, true
}

// InspectSeedStack opens the inspector on a seed that has not been sown.
func (u *UIState) InspectSeedStack(stack sim.SeedStack) {
	u.InspectOpen, u.InspectKind, u.InspectGenome = true, stack.Kind, stack.Genome
	u.InspectFromPlot = false
}

func (u *UIState) CloseInspector() { u.InspectOpen = false }

// Notify shows a message for a short while.
func (u *UIState) Notify(msg string) {
	u.Notice, u.noticeTicks = msg, NoticeLifetimeTicks
}

// TickNotice ages the current message out. View state only.
func (u *UIState) TickNotice() {
	if u.noticeTicks > 0 {
		u.noticeTicks--
		if u.noticeTicks == 0 {
			u.Notice = ""
		}
	}
}

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
