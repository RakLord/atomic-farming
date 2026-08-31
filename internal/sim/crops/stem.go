// Package crops holds every concrete Crop implementation.
package crops

import (
	"atomicfarming/internal/bignum"
	"atomicfarming/internal/plant"
	"atomicfarming/internal/sim"
)

// KindStem is the starter crop: a plain upright stalk with no flower and no
// fruit. Everyone begins with it, and every other crop is measured against it.
const KindStem sim.CropKind = "stem"

// StemStages is how many stages a Stem passes through. At the default tick
// rate that is roughly 8 to 50 seconds from sowing to ready, depending on the
// plant's Growth Rate.
const StemStages = 4

// stemRanges pins the Stem's identity.
//
// The archetype genes are clamped so hard that a Stem is always an upright
// stalk with no flower and no fruit — remapping means allele 255 still lands
// inside the window, so no genome can escape the species. The numeric genes
// stay wide, because those are what the player breeds; in particular Density
// runs the full range, since that is the stat the whole game climbs toward.
var stemRanges = plant.RangesFrom(map[plant.GeneID]plant.GeneRange{
	plant.GeneStemArchetype:   {Min: 0, Max: 40}, // always Upright
	plant.GeneFlowerArchetype: {Min: 0, Max: 30}, // always None
	plant.GeneFruitArchetype:  {Min: 0, Max: 35}, // always None
	plant.GeneLeafArchetype:   {Min: 0, Max: 90}, // Oval or Lance

	plant.GeneStemHeight:    {Min: 60, Max: 255},
	plant.GeneStemThickness: {Min: 40, Max: 255},
	plant.GeneStemTaper:     {Min: 80, Max: 220},
	plant.GeneNodeCount:     {Min: 40, Max: 200},
	plant.GeneStemHue:       {Min: 50, Max: 130}, // greens
	plant.GeneStemSat:       {Min: 20, Max: 230},
	plant.GeneStemLum:       {Min: 40, Max: 220},
	plant.GeneLeafSize:      {Min: 60, Max: 200},

	// Wide on purpose: 10s at the top of this window, 75s at the bottom, so
	// Growth Rate is worth breeding for rather than a rounding error.
	plant.GeneGrowthRate: {Min: 22, Max: 255},
	// A starter crop the player cannot keep alive is not a starter crop.
	plant.GeneVitality: {Min: 210, Max: 255},

	plant.GeneYieldAmount:   {Min: 30, Max: 200},
	plant.GeneYieldQuality:  {Min: 20, Max: 200},
	plant.GeneHarvestChance: {Min: 190, Max: 255},
	plant.GeneRegrowth:      {Min: 0, Max: 110}, // below the perennial threshold
	plant.GeneMutability:    {Min: 0, Max: 90},
})

// Stem carries no per-plant state: everything that varies lives in the plot's
// genome and growth. See docs/adr/0005-crop-registry-and-plot-model.md.
type Stem struct{}

func (s *Stem) Kind() sim.CropKind          { return KindStem }
func (s *Stem) Stages() int                 { return StemStages }
func (s *Stem) Ranges() plant.SpeciesRanges { return stemRanges }

func (s *Stem) Grow(ctx sim.GrowContext, g sim.Growth) sim.Growth {
	return sim.AdvanceGrowth(ctx, g, StemStages)
}

func (s *Stem) Harvest(ctx sim.GrowContext, g sim.Growth) (sim.Yield, sim.Growth, bool) {
	return sim.StandardHarvest(ctx, KindStem, g, StemStages)
}

// OfferStemSeed is the always-available shop entry every farm starts from.
const OfferStemSeed sim.SeedOfferID = "stem_seed"

// OfferVoidshoot is the rare seed, gated behind an unlock.
const (
	OfferVoidshoot  sim.SeedOfferID = "voidshoot_seed"
	UnlockVoidshoot sim.UnlockID    = "voidshoot_licence"
)

// voidshootGenome is a handcrafted strain: maximum density, near-black stem,
// sterile. Reaching it by breeding is effectively impossible, which is exactly
// why it is matched by exact signature and sold rather than bred.
var voidshootGenome = buildVoidshoot()

func buildVoidshoot() plant.Genome {
	g := plant.RandomGenome(0x501D_0F5E_ED17)
	for gene, v := range map[plant.GeneID]plant.Allele{
		plant.GeneDensity:       255,
		plant.GeneStemLum:       0,
		plant.GeneStemSat:       12,
		plant.GeneStemHeight:    250,
		plant.GeneStemThickness: 235,
		plant.GeneVitality:      255,
		plant.GeneYieldQuality:  240,
		plant.GeneMutability:    0,
	} {
		g[gene] = plant.GenePair{A: v, B: v}
	}
	return g
}

func init() {
	sim.RegisterCrop(KindStem, func() sim.Crop { return &Stem{} })
	sim.RegisterCropName(KindStem, "Stem")

	sim.RegisterSeedOffer(sim.SeedOffer{
		ID:          OfferStemSeed,
		Kind:        KindStem,
		Name:        "Stem Seed",
		Description: "A plain green stalk. Grows anywhere, dies of almost nothing.",
		// The catalog default expressed through stemRanges: every player's
		// first plant is the same, and variation arrives through harvest.
		Genome:   plant.DefaultGenome(),
		BaseCost: bignum.MustParse("5"),
	})
	sim.StarterOffer = OfferStemSeed

	sim.RegisterUnlock(sim.Unlock{
		ID:          UnlockVoidshoot,
		Name:        "Voidshoot Licence",
		Description: "Permits the sale of collapsed-matter seed stock.",
		Cost:        bignum.MustParse("2500"),
		// No Apply: this unlock exists only to open a shop entry, and
		// rebuildModifiers skips a nil apply.
	})
	sim.RegisterSeedOffer(sim.SeedOffer{
		ID:             OfferVoidshoot,
		Kind:           KindStem,
		Name:           "Voidshoot Seed",
		Description:    "Matter wound so tight the stalk drinks light.",
		Genome:         voidshootGenome,
		BaseCost:       bignum.MustParse("900"),
		RequiresUnlock: UnlockVoidshoot,
	})

	registerStemStrains()
}

// registerStemStrains defines what a Stem can be recognised as.
//
// Every predicate here is a condition on visible traits, so its Goal reads as
// something the player can deliberately breed toward rather than a lottery
// win. Each must also be reachable inside stemRanges — a strain nobody can
// ever grow is a silent content bug, which is what
// TestEveryPredicateStrainIsReachable guards against.
func registerStemStrains() {
	sim.RegisterStrain(sim.NamedStrain{
		ID:          "ironstem",
		Name:        "Ironstem",
		Description: "A stalk so dense it rings when struck.",
		Goal:        "Very high Density on a very thick stem",
		Rarity:      sim.RarityRare,
		Kind:        KindStem,
		Specificity: 20,
		Match: func(p plant.Phenotype) bool {
			return p.Get(plant.GeneDensity) >= 210 && p.Get(plant.GeneStemThickness) >= 200
		},
	})
	sim.RegisterStrain(sim.NamedStrain{
		ID:          "sunspire",
		Name:        "Sunspire",
		Description: "Races for the light and wins.",
		Goal:        "Near-maximum Stem Height and Growth Rate",
		Rarity:      sim.RarityUncommon,
		Kind:        KindStem,
		Specificity: 15,
		Match: func(p plant.Phenotype) bool {
			return p.Get(plant.GeneStemHeight) >= 235 && p.Get(plant.GeneGrowthRate) >= 210
		},
	})
	sim.RegisterStrain(sim.NamedStrain{
		ID:          "palewood",
		Name:        "Palewood",
		Description: "Bleached almost white, and oddly valuable for it.",
		Goal:        "Very low Stem Saturation with high Luminance",
		Rarity:      sim.RarityUncommon,
		Kind:        KindStem,
		Specificity: 15,
		Match: func(p plant.Phenotype) bool {
			return p.Get(plant.GeneStemSat) <= 40 && p.Get(plant.GeneStemLum) >= 205
		},
	})
	sim.RegisterStrain(sim.NamedStrain{
		ID:          "gnarlroot",
		Name:        "Gnarlroot",
		Description: "Short, crooked, and stubborn about it.",
		Goal:        "A hard-bent stem kept short",
		Rarity:      sim.RarityRare,
		Kind:        KindStem,
		Specificity: 20,
		Match: func(p plant.Phenotype) bool {
			curve := p.Get(plant.GeneStemCurve)
			return (curve <= 25 || curve >= 230) && p.Get(plant.GeneStemHeight) <= 90
		},
	})
	sim.RegisterStrain(sim.NamedStrain{
		ID:          "voidshoot",
		Name:        "Voidshoot",
		Description: "Collapsed matter that has not finished collapsing.",
		Goal:        "Cannot be bred — sold under licence",
		Rarity:      sim.RarityLegendary,
		Kind:        KindStem,
		Specificity: 100,
		Signature:   &voidshootGenome,
	})
}
