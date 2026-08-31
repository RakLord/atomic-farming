// Package input turns resolved plot coordinates and panel clicks into game
// actions. It imports neither Ebitengine nor any layout geometry: the render
// layer maps screen coordinates to a sim.Position or a catalog ID and calls in
// here, which keeps every action testable without a window.
package input

import (
	"fmt"

	"atomicfarming/internal/bignum"
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

// ClickPlot is the single gesture that works the farm: sow an empty plot with
// the queued seed, gather a ready one, and otherwise just select.
//
// One button doing the obvious thing for the plot under it beats a mode
// selector the player has to remember they are in.
func ClickPlot(s *sim.GameState, u *ui.UIState, p sim.Position, ok bool) {
	if u == nil {
		return
	}
	if !ok || s == nil || !s.Grid.InBounds(p) {
		u.ClearSelection()
		return
	}
	u.Select(p)

	plot, found := s.Grid.At(p)
	if !found {
		return
	}
	switch {
	case plot.Crop == nil:
		plantAt(s, u, p)
	case plot.Growth.Ready:
		harvestAt(s, u, p)
	}
}

func plantAt(s *sim.GameState, u *ui.UIState, p sim.Position) {
	idx := u.SeedIndex(&s.Inventory)
	if idx < 0 {
		if s.Inventory.Total() == 0 {
			u.Notify("No seeds — buy one from the shop")
		} else {
			u.Notify("Pick a seed first")
		}
		return
	}
	name := sim.SeedStrainName(s.Inventory.Stacks[idx])
	if err := s.PlantSeed(p, idx); err != nil {
		u.Notify(friendly(err))
		return
	}
	u.Notify("Sowed " + name)
	// Keep the seed queued if any remain, so a row of plots is one click each.
	if u.SeedIndex(&s.Inventory) < 0 {
		u.ClearSeed()
	}
}

func harvestAt(s *sim.GameState, u *ui.UIState, p sim.Position) {
	name := s.PlotStrainName(p)
	res, err := s.Harvest(p)
	if err != nil {
		u.Notify(friendly(err))
		return
	}
	switch {
	case !res.Success:
		u.Notify(name + " failed to set — crop lost")
	case res.NewStrain:
		u.Notify("New strain discovered: " + res.Strain.Name + "!")
	default:
		u.Notify(fmt.Sprintf("Harvested %s: %d units, +$%s, %d seeds",
			name, res.Units, res.Cash.Format(bignum.DisplayShort, 2), res.Seeds))
	}
}

// SelectSeed queues an inventory stack for planting.
func SelectSeed(s *sim.GameState, u *ui.UIState, index int) {
	if s == nil || u == nil || index < 0 || index >= len(s.Inventory.Stacks) {
		return
	}
	u.SelectSeed(s.Inventory.Stacks[index])
}

// BuySeed purchases a shop offer.
func BuySeed(s *sim.GameState, u *ui.UIState, id sim.SeedOfferID) {
	if s == nil || u == nil {
		return
	}
	offer, ok := sim.SeedShop[id]
	if !ok {
		return
	}
	if err := sim.BuySeed(s, id); err != nil {
		u.Notify(friendly(err))
		return
	}
	u.Notify("Bought " + offer.Name)
}

// BuyUnlock purchases a global upgrade.
func BuyUnlock(s *sim.GameState, u *ui.UIState, id sim.UnlockID) {
	if s == nil || u == nil {
		return
	}
	unlock, ok := sim.UnlockCatalog[id]
	if !ok {
		return
	}
	if err := sim.PurchaseUnlock(s, id); err != nil {
		u.Notify(friendly(err))
		return
	}
	u.Notify("Unlocked " + unlock.Name)
}

// friendly turns a sim error into something worth reading on screen.
func friendly(err error) string {
	switch err {
	case sim.ErrTooExpensive:
		return "Not enough cash"
	case sim.ErrPlotOccupied:
		return "Something is already growing there"
	case sim.ErrNotReady:
		return "Not ready yet"
	case sim.ErrNoSeed:
		return "No seed to sow"
	case sim.ErrLocked:
		return "Not unlocked yet"
	default:
		return err.Error()
	}
}
