package crops

import (
	"testing"

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
		if strain.Match == nil || strain.Kind != KindStem {
			continue
		}
		found := false
		for seed := uint64(0); seed < samples && !found; seed++ {
			g := homozygousGenome(seed)
			if strain.Match(plant.Express(g, stem.Ranges())) {
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

func TestStemGrowsInAReasonableTime(t *testing.T) {
	// Fixed seed: a clock-seeded world would occasionally kill the plant and
	// turn this timing check into a flake.
	s := sim.NewGameStateWithSeed(7)
	pos := sim.Position{}
	if err := s.PlantSeed(pos, 0); err != nil {
		t.Fatalf("PlantSeed: %v", err)
	}
	ticks := 0
	for i := 0; i < 20000; i++ {
		s.Tick()
		ticks++
		plot, _ := s.Grid.At(pos)
		if plot.Crop == nil {
			t.Fatal("the plant died")
		}
		if plot.Growth.Ready {
			break
		}
	}
	seconds := float64(ticks) / float64(sim.DefaultTickRate)
	if seconds < 3 || seconds > 120 {
		t.Errorf("a default Stem takes %.1fs to mature; that is outside a playable window", seconds)
	}
	t.Logf("a default Stem matures in %d ticks (%.1fs)", ticks, seconds)
}
