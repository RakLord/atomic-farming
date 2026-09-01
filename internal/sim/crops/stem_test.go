package crops

import (
	"testing"

	"atomicfarming/internal/bignum"

	"atomicfarming/internal/plant"
	"atomicfarming/internal/sim"
)

// homozygousGenome builds a genome whose alleles are all paired, so each gene
// expresses its own value directly.
//
// Reachability is a question about the species window, not about breeding
// luck. Random heterozygous genomes average their alleles, which crowds every
// expressed value toward the middle and would make an extreme-trait strain
// look unreachable when it is merely rare. A homozygote is also how a player
// would actually stabilise a bred line.
func homozygousGenome(seed uint64) plant.Genome {
	var g plant.Genome
	for i := 0; i < plant.GeneCount; i++ {
		v := plant.Allele(plant.Roll(seed, plant.PurposeGenome, uint64(i), 256))
		g[i] = plant.GenePair{A: v, B: v}
	}
	return g
}

// TestEveryPredicateStrainIsReachable is a content guard: a named strain whose
// conditions fall outside its species' gene ranges can never be grown, and
// nothing else would ever tell us.
func TestEveryPredicateStrainIsReachable(t *testing.T) {
	const samples = 20000
	stem := &Stem{}

	for _, id := range sim.StrainCatalogOrder {
		strain := sim.StrainCatalog[id]
		if !strain.Breedable() || strain.Kind != KindStem {
			continue
		}
		found := false
		for seed := uint64(0); seed < samples && !found; seed++ {
			g := homozygousGenome(seed)
			if strain.Matches(g, plant.Express(g, stem.Ranges())) {
				found = true
			}
		}
		if !found {
			t.Errorf("strain %q (%s) is unreachable within the Stem's gene ranges — no genome can satisfy %q",
				strain.ID, strain.Name, strain.Goal)
		}
	}
}

// TestStemRangesPinItsIdentity: a Stem is a stem whatever its genome says.
func TestStemRangesPinItsIdentity(t *testing.T) {
	stem := &Stem{}
	for seed := uint64(0); seed < 4000; seed++ {
		p := plant.Express(homozygousGenome(seed), stem.Ranges())
		if got := p.StemArchetype(); got != plant.StemUpright {
			t.Fatalf("seed %d grew a %v stem, want Upright", seed, got)
		}
		if got := p.FlowerArchetype(); got != plant.FlowerNone {
			t.Fatalf("seed %d grew a %v flower; a Stem has none", seed, got)
		}
		if got := p.FruitArchetype(); got != plant.FruitNone {
			t.Fatalf("seed %d grew %v fruit; a Stem has none", seed, got)
		}
	}
}

// TestStemIsHardyEnoughToLearnOn: the starter crop must not punish a new
// player with random deaths.
func TestStemIsHardyEnoughToLearnOn(t *testing.T) {
	stem := &Stem{}
	for seed := uint64(0); seed < 2000; seed++ {
		p := plant.Express(homozygousGenome(seed), stem.Ranges())
		if v := p.Get(plant.GeneVitality); v < 210 {
			t.Fatalf("a Stem can express Vitality %d; the starter crop should be reliable", v)
		}
		if c := p.Get(plant.GeneHarvestChance); c < 190 {
			t.Fatalf("a Stem can express Harvest Chance %d; the starter crop should be reliable", c)
		}
	}
}

func TestStemsAreAnnualsSoTheShopStaysRelevant(t *testing.T) {
	stem := &Stem{}
	for seed := uint64(0); seed < 2000; seed++ {
		p := plant.Express(homozygousGenome(seed), stem.Ranges())
		if sim.Regrows(p) {
			t.Fatalf("seed %d produced a perennial Stem; the starter crop is meant to be replanted", seed)
		}
	}
}

func TestVoidshootIsIdentifiedByItsSignature(t *testing.T) {
	offer, ok := sim.SeedShop[OfferVoidshoot]
	if !ok {
		t.Fatal("the Voidshoot offer is not registered")
	}
	pheno := plant.Express(offer.Genome, (&Stem{}).Ranges())

	strain, matched := sim.IdentifyStrain(KindStem, offer.Genome, pheno)
	if !matched {
		t.Fatal("the Voidshoot seed does not identify as a strain")
	}
	if strain.ID != "voidshoot" {
		t.Errorf("the Voidshoot seed identified as %q", strain.ID)
	}
	// A legendary must outrank any predicate strain it happens to satisfy.
	if strain.Rarity != sim.RarityLegendary {
		t.Errorf("Voidshoot resolved to rarity %v", strain.Rarity)
	}
	// It is still a stem, despite its genome being handcrafted.
	if pheno.StemArchetype() != plant.StemUpright || pheno.FlowerArchetype() != plant.FlowerNone {
		t.Error("the Voidshoot genome escapes the Stem's species ranges")
	}
}

func TestVoidshootCannotBeReachedByOrdinaryGenomes(t *testing.T) {
	offer := sim.SeedShop[OfferVoidshoot]
	for seed := uint64(0); seed < 20000; seed++ {
		if homozygousGenome(seed) == offer.Genome {
			t.Fatal("a sampled genome hit the Voidshoot signature exactly")
		}
	}
}

// TestStarterSetupIsPlayable checks the opening of the game across many
// worlds rather than one.
//
// A single run of this would be rolling the starter Stem's harvest chance —
// about 97%, so it fails roughly one run in thirty-six. That is a fine rate
// for a game and a terrible one for a test, and it is exactly the kind of flake
// that passes locally and fails in CI.
func TestStarterSetupIsPlayable(t *testing.T) {
	if sim.StarterOffer != OfferStemSeed {
		t.Errorf("StarterOffer = %q, want the Stem seed", sim.StarterOffer)
	}

	const worlds = 200
	paid, died, ticksTotal := 0, 0, 0

	for seed := uint64(1); seed <= worlds; seed++ {
		s := sim.NewGameStateWithSeed(seed)
		if s.Inventory.Total() != sim.StarterSeeds {
			t.Fatalf("a new farm starts with %d seeds, want %d", s.Inventory.Total(), sim.StarterSeeds)
		}
		if !s.Cash.IsZero() {
			t.Fatalf("a new farm starts with %s cash, want zero", s.Cash)
		}

		pos := sim.Position{}
		if err := s.PlantSeed(pos, 0); err != nil {
			t.Fatalf("planting a starter seed: %v", err)
		}
		ticks, lost := 0, false
		for i := 0; i < 20000; i++ {
			s.Tick()
			ticks++
			plot, _ := s.Grid.At(pos)
			if plot.Crop == nil {
				lost = true
				break
			}
			if plot.Growth.Ready {
				break
			}
		}
		if lost {
			died++
			continue
		}
		ticksTotal += ticks

		res, err := s.Harvest(pos)
		if err != nil {
			t.Fatalf("harvesting a starter plant: %v", err)
		}
		if !res.Success {
			continue
		}
		paid++
		// A first harvest must cover a meaningful share of the next seed, or
		// the loop never gets going.
		if s.Cash.LT(sim.SeedCost(s, OfferStemSeed).DivInt(4)) {
			t.Fatalf("world %d: a first harvest pays %s against a seed price of %s",
				seed, s.Cash, sim.SeedCost(s, OfferStemSeed))
		}
	}

	// The starter crop is meant to be forgiving, not infallible: losing a
	// first plant should be a rarity a player shrugs at, not a coin flip.
	if died > worlds/50 {
		t.Errorf("%d/%d starter plants died before ripening; the opening is too punishing", died, worlds)
	}
	if paid < worlds*90/100 {
		t.Errorf("only %d/%d first harvests paid out; the opening is too punishing", paid, worlds)
	}
	t.Logf("%d/%d first harvests paid, %d plants lost; a starter Stem matures in %d ticks on average",
		paid, worlds, died, ticksTotal/(worlds-died))
}

// maturationSeconds grows one Stem of the given genome and reports how long it
// took. Vitality is maxed so a death roll cannot skew the timing.
func maturationSeconds(t *testing.T, g plant.Genome) float64 {
	t.Helper()
	s := sim.NewGameStateWithSeed(7)
	s.Inventory = sim.Inventory{}
	s.Inventory.Add(KindStem, g, 1)

	pos := sim.Position{}
	if err := s.PlantSeed(pos, 0); err != nil {
		t.Fatalf("PlantSeed: %v", err)
	}
	for i := 0; i < 100000; i++ {
		s.Tick()
		plot, _ := s.Grid.At(pos)
		if plot.Crop == nil {
			t.Fatal("the plant died before ripening")
		}
		if plot.Growth.Ready {
			return float64(i+1) / float64(sim.DefaultTickRate)
		}
	}
	t.Fatal("the plant never ripened")
	return 0
}

func stemWithGrowthRate(v plant.Allele) plant.Genome {
	g := plant.DefaultGenome()
	g[plant.GeneGrowthRate] = plant.GenePair{A: v, B: v}
	return g
}

// TestTheStarterStemTakesTwentySeconds pins the opening pace.
//
// This test used to accept anything from 3 to 120 seconds, which is how the
// starter sat at 9.4s unnoticed. A pacing number nobody asserts is a pacing
// number that drifts.
func TestTheStarterStemTakesTwentySeconds(t *testing.T) {
	offer, ok := sim.SeedShop[OfferStemSeed]
	if !ok {
		t.Fatal("the starter offer is not registered")
	}
	got := maturationSeconds(t, offer.Genome)
	if got < 19 || got > 21 {
		t.Errorf("the starter Stem matures in %.1fs, want 20s", got)
	}
	t.Logf("starter Stem matures in %.1fs", got)
}

// TestGrowthRateSpreadIsWide: the gene has to be worth breeding for, not a
// rounding error next to Density.
func TestGrowthRateSpreadIsWide(t *testing.T) {
	stem := &Stem{}
	window := stem.Ranges()[plant.GeneGrowthRate]

	fastest := maturationSeconds(t, stemWithGrowthRate(255))
	slowest := maturationSeconds(t, stemWithGrowthRate(0))

	if fastest < 9 || fastest > 11 {
		t.Errorf("the fastest Stem matures in %.1fs, want about 10s", fastest)
	}
	if slowest < 70 || slowest > 80 {
		t.Errorf("the slowest Stem matures in %.1fs, want about 75s", slowest)
	}
	if spread := slowest / fastest; spread < 6 {
		t.Errorf("growth spread is only %.1fx; the gene barely matters", spread)
	}
	t.Logf("Stem matures in %.1fs to %.1fs (gene window %d..%d)",
		fastest, slowest, window.Min, window.Max)
}

func TestFieldExtensionWidensTheFarmImmediately(t *testing.T) {
	s := sim.NewGameStateWithSeed(1)
	s.Cash = bignum.MustParse("500")

	if s.Grid.W != 3 || s.Grid.H != 3 {
		t.Fatalf("a new farm is %dx%d, want 3x3", s.Grid.W, s.Grid.H)
	}
	if err := sim.PurchaseUnlock(s, sim.UnlockFieldExtension); err != nil {
		t.Fatalf("PurchaseUnlock: %v", err)
	}
	if s.Grid.W != 4 || s.Grid.H != 3 {
		t.Errorf("after buying Field Extension the farm is %dx%d, want 4x3", s.Grid.W, s.Grid.H)
	}
	if want := bignum.MustParse("400"); !s.Cash.Eq(want) {
		t.Errorf("cash = %s, want %s", s.Cash, want)
	}
}

// TestFieldExtensionSurvivesPrestige: you paid for the ground.
func TestFieldExtensionSurvivesPrestige(t *testing.T) {
	s := sim.NewGameStateWithSeed(1)
	s.Cash = bignum.MustParse("500")
	if err := sim.PurchaseUnlock(s, sim.UnlockFieldExtension); err != nil {
		t.Fatalf("PurchaseUnlock: %v", err)
	}

	sim.ApplyLayerReset(s, sim.LayerField)

	if s.Grid.W != 4 || s.Grid.H != 3 {
		t.Errorf("after a prestige the farm is %dx%d, want the 4x3 you bought", s.Grid.W, s.Grid.H)
	}
}

// TestFieldExtensionIsAReachableFirstGoal measures how long $100 actually takes
// from a standing start. Under a minute and it is not a goal; over several and
// it is a wall.
func TestFieldExtensionIsAReachableFirstGoal(t *testing.T) {
	s := sim.NewGameStateWithSeed(99)
	cost := sim.SeedCost(s, OfferStemSeed)
	target := sim.UnlockCatalog[sim.UnlockFieldExtension].Cost

	ticks := 0
	for ticks < 60*60*sim.DefaultTickRate && s.Cash.LT(target) {
		// Sow anything sowable, gather anything ready, buy a seed when the
		// barn is empty and there is cash for one — roughly how the opening
		// actually plays.
		for i := range s.Grid.Plots {
			pos := s.Grid.PositionAt(i)
			plot, _ := s.Grid.At(pos)
			switch {
			case plot.Crop == nil && s.Inventory.Total() > 0:
				_ = s.PlantSeed(pos, 0)
			case plot.Crop != nil && plot.Growth.Ready:
				if _, err := s.Harvest(pos); err != nil {
					t.Fatalf("Harvest: %v", err)
				}
			}
		}
		if s.Inventory.Total() == 0 && s.Cash.GTE(cost) {
			if err := sim.BuySeed(s, OfferStemSeed); err != nil {
				t.Fatalf("BuySeed: %v", err)
			}
		}
		s.Tick()
		ticks++
	}

	if s.Cash.LT(target) {
		t.Fatalf("an hour of play did not reach %s; the first upgrade is unreachable", target)
	}
	mins := float64(ticks) / float64(sim.DefaultTickRate) / 60
	if mins < 0.5 {
		t.Errorf("the first upgrade arrives after %.1f minutes; too cheap to be a goal", mins)
	}
	if mins > 8 {
		t.Errorf("the first upgrade takes %.1f minutes; that is a wall, not a goal", mins)
	}
	t.Logf("$100 reached after %.1f minutes of play", mins)
}
