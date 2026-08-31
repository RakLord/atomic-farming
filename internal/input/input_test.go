package input

import (
	"strings"
	"testing"

	"atomicfarming/internal/bignum"
	"atomicfarming/internal/plant"
	"atomicfarming/internal/sim"
	// The Stem registers itself here, giving these tests a real crop and a
	// real shop to act on rather than a stand-in.
	"atomicfarming/internal/sim/crops"
	"atomicfarming/internal/ui"
)

// newGame roots the world at a fixed seed. sim.NewGameState draws its seed
// from the clock, so any test that grows and harvests a plant would otherwise
// be rolling that plant's ~3% chance of dying or failing to set — a flake that
// passes locally and fails in CI.
func newGame() (*sim.GameState, *ui.UIState) {
	return sim.NewGameStateWithSeed(20260901), ui.NewUIState()
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

func TestClickingASingleLineGroupSowsItWithoutAPicker(t *testing.T) {
	s, u := newGame()
	groups := s.GroupSeeds()
	if len(groups) != 1 || len(groups[0].Stacks) != 1 {
		t.Fatalf("a new farm should hold one line, got %+v", groups)
	}

	ClickSeedGroup(s, u, groups[0])
	if u.SeedIndexOpen {
		t.Error("a group with one line opened the picker; the common case must stay one click")
	}
	if !u.HasSeed {
		t.Error("clicking a single-line group did not queue its seed")
	}
}

// TestAutoPickSowsWithoutOpeningThePicker covers the default: sowing is the
// action you take constantly, so a row click must never make you confirm which
// of your near-identical lines you meant.
func TestAutoPickSowsWithoutOpeningThePicker(t *testing.T) {
	s, u := newGame()
	s.Inventory.Add(crops.KindStem, plant.RandomGenome(41), 2)
	s.Inventory.Add(crops.KindStem, plant.RandomGenome(42), 1)

	groups := s.GroupSeeds()
	if len(groups) != 1 {
		t.Fatalf("unnamed stems should collapse to one group, got %d", len(groups))
	}
	ClickSeedGroup(s, u, groups[0])

	if u.SeedIndexOpen {
		t.Error("auto-pick is on by default, so a row click should not open the picker")
	}
	if !u.HasSeed {
		t.Fatal("the row click did not queue a seed")
	}
	// It queues the bulk line, not whichever happened to be first.
	if idx := u.SeedIndex(&s.Inventory); idx < 0 || s.Inventory.Stacks[idx].Count != 3 {
		t.Error("auto-pick did not queue the most numerous line")
	}
}

func TestTurningAutoPickOffRestoresThePicker(t *testing.T) {
	s, u := newGame()
	s.Inventory.Add(crops.KindStem, plant.RandomGenome(41), 2)
	s.Inventory.Add(crops.KindStem, plant.RandomGenome(42), 1)

	ToggleSeedAutoSelect(s, u, crops.KindStem)
	if s.AutoSelectSeeds(crops.KindStem) {
		t.Fatal("the toggle did not turn auto-pick off")
	}

	ClickSeedGroup(s, u, s.GroupSeeds()[0])
	if !u.SeedIndexOpen {
		t.Error("with auto-pick off, a row click should open the picker")
	}

	ToggleSeedAutoSelect(s, u, crops.KindStem)
	if !s.AutoSelectSeeds(crops.KindStem) {
		t.Error("the toggle did not turn auto-pick back on")
	}
}

// TestTheIndexButtonAlwaysOpensThePicker: auto-pick removes the picker from
// the click path, so there must still be a deliberate way to reach it.
func TestTheIndexButtonAlwaysOpensThePicker(t *testing.T) {
	s, u := newGame()
	s.Inventory.Add(crops.KindStem, plant.RandomGenome(41), 2)

	if !s.AutoSelectSeeds(crops.KindStem) {
		t.Fatal("auto-pick should default on")
	}
	OpenSeedIndex(s, u, crops.KindStem)
	if !u.SeedIndexOpen {
		t.Error("the index button did not open the picker")
	}

	u.CloseSeedIndex()
	OpenSeedIndex(s, u, "no_such_species")
	if u.SeedIndexOpen {
		t.Error("the picker opened on a species holding no seeds")
	}
}

func TestDiscardRemovesTheLineAndUnqueuesIt(t *testing.T) {
	s, u := newGame()
	s.Inventory = sim.Inventory{}
	doomed := plant.RandomGenome(43)
	s.Inventory.Add(crops.KindStem, doomed, 3)
	s.Inventory.Add(crops.KindStem, plant.RandomGenome(44), 1)

	SelectSeed(s, u, 0)
	if !u.HasSeed {
		t.Fatal("the seed was not queued")
	}

	DiscardSeed(s, u, 0)
	if u.HasSeed {
		t.Error("discarding the queued line left it queued; sowing would fail silently")
	}
	if s.Inventory.Total() != 1 {
		t.Errorf("%d seeds remain, want 1", s.Inventory.Total())
	}
	if !strings.Contains(u.Notice, "Discarded") {
		t.Errorf("notice was %q, want a discard confirmation", u.Notice)
	}
}

func TestDiscardingTheLastLineClosesTheIndex(t *testing.T) {
	s, u := newGame()
	s.Inventory = sim.Inventory{}
	s.Inventory.Add(crops.KindStem, plant.RandomGenome(45), 1)
	u.OpenSeedIndex(crops.KindStem)

	DiscardSeed(s, u, 0)
	if u.SeedIndexOpen {
		t.Error("the picker stayed open over an empty species")
	}
}

func TestDiscardIgnoresBadIndices(t *testing.T) {
	s, u := newGame()
	before := s.Inventory.Total()
	DiscardSeed(s, u, -1)
	DiscardSeed(s, u, 99)
	if s.Inventory.Total() != before {
		t.Error("an out-of-range discard changed the inventory")
	}
}
