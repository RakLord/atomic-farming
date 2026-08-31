package input

import (
	"strings"
	"testing"

	"atomicfarming/internal/bignum"
	"atomicfarming/internal/sim"
	// The Stem registers itself here, giving these tests a real crop and a
	// real shop to act on rather than a stand-in.
	"atomicfarming/internal/sim/crops"
	"atomicfarming/internal/ui"
)

func newGame() (*sim.GameState, *ui.UIState) {
	return sim.NewGameState(), ui.NewUIState()
}

func TestHoverTracksAndClears(t *testing.T) {
	s, u := newGame()
	p := sim.Position{X: 2}

	Hover(s, u, p, true)
	if !u.IsHovered(p) {
		t.Fatal("hover was not recorded")
	}
	Hover(s, u, sim.Position{}, false)
	if u.HasHover {
		t.Error("hover was not cleared when the cursor left the farm")
	}
}

func TestClickingOffTheFarmClearsTheSelection(t *testing.T) {
	s, u := newGame()
	ClickPlot(s, u, sim.Position{}, true)
	ClickPlot(s, u, sim.Position{}, false)
	if u.HasSelection {
		t.Error("clicking off the farm did not clear the selection")
	}
}

// TestClickSowsThenGathers is the whole gesture: one button that does the
// obvious thing for whatever is under it.
func TestClickSowsThenGathers(t *testing.T) {
	s, u := newGame()
	pos := sim.Position{X: 1, Y: 1}
	SelectSeed(s, u, 0)

	ClickPlot(s, u, pos, true)
	plot, _ := s.Grid.At(pos)
	if plot.Crop == nil {
		t.Fatalf("clicking an empty plot with a seed queued did not sow it (%q)", u.Notice)
	}

	// Clicking again while it grows must not disturb it.
	ClickPlot(s, u, pos, true)
	if plot, _ := s.Grid.At(pos); plot.Crop == nil {
		t.Fatal("clicking a growing plant destroyed it")
	}

	for i := 0; i < 20000; i++ {
		s.Tick()
		if plot, _ := s.Grid.At(pos); plot.Crop == nil || plot.Growth.Ready {
			break
		}
	}
	if plot, _ := s.Grid.At(pos); plot.Crop == nil {
		t.Fatal("the plant died before ripening")
	}

	before := s.Cash
	ClickPlot(s, u, pos, true)
	if !s.Cash.GT(before) {
		t.Errorf("clicking a ready plant paid nothing (%q)", u.Notice)
	}
}

func TestSowingWithNoSeedSelectedExplainsItself(t *testing.T) {
	s, u := newGame()
	ClickPlot(s, u, sim.Position{}, true)

	if plot, _ := s.Grid.At(sim.Position{}); plot.Crop != nil {
		t.Fatal("a plot was sown with no seed queued")
	}
	if !strings.Contains(u.Notice, "seed") {
		t.Errorf("notice was %q; it should say a seed is needed", u.Notice)
	}
}

func TestTheQueuedSeedSurvivesUntilItRunsOut(t *testing.T) {
	s, u := newGame()
	SelectSeed(s, u, 0)
	total := s.Inventory.Total()

	for i := 0; i < total; i++ {
		ClickPlot(s, u, s.Grid.PositionAt(i), true)
	}
	if s.Inventory.Total() != 0 {
		t.Fatalf("%d seeds left, want all sown", s.Inventory.Total())
	}
	if u.HasSeed {
		t.Error("a seed is still queued after the last one was sown")
	}
}

func TestBuyingReportsSuccessAndFailure(t *testing.T) {
	s, u := newGame()

	BuySeed(s, u, crops.OfferStemSeed)
	if !strings.Contains(strings.ToLower(u.Notice), "cash") {
		t.Errorf("buying with no money said %q; it should mention cash", u.Notice)
	}

	s.Cash = bignum.MustParse("500")
	before := s.Inventory.Total()
	BuySeed(s, u, crops.OfferStemSeed)
	if s.Inventory.Total() != before+1 {
		t.Errorf("inventory holds %d, want %d", s.Inventory.Total(), before+1)
	}
	if !strings.Contains(u.Notice, "Bought") {
		t.Errorf("notice was %q, want a purchase confirmation", u.Notice)
	}
}

func TestBuyingAnUnlockOpensItsShopEntry(t *testing.T) {
	s, u := newGame()
	if sim.SeedOfferAvailable(s, crops.OfferVoidshoot) {
		t.Fatal("the gated offer is available before its unlock")
	}

	s.Cash = bignum.MustParse("100000")
	BuyUnlock(s, u, crops.UnlockVoidshoot)
	if !sim.SeedOfferAvailable(s, crops.OfferVoidshoot) {
		t.Fatalf("the offer is still gated after buying the unlock (%q)", u.Notice)
	}

	BuySeed(s, u, crops.OfferVoidshoot)
	if !s.DiscoveredStrains["voidshoot"] {
		t.Error("buying a named seed did not log the discovery")
	}
}

func TestSelectSeedIgnoresBadIndices(t *testing.T) {
	s, u := newGame()
	SelectSeed(s, u, -1)
	SelectSeed(s, u, 999)
	if u.HasSeed {
		t.Error("an out-of-range index selected a seed")
	}
}
