package sim

import (
	"testing"

	"atomicfarming/internal/plant"
)

// withStrains registers strains for one test and restores the catalog after.
// The catalog is package state, so a test that added to it permanently would
// leak into every other test's identification results.
func withStrains(t *testing.T, strains ...NamedStrain) {
	t.Helper()
	savedCatalog := make(map[StrainID]NamedStrain, len(StrainCatalog))
	for k, v := range StrainCatalog {
		savedCatalog[k] = v
	}
	savedOrder := append([]StrainID(nil), StrainCatalogOrder...)
	t.Cleanup(func() { StrainCatalog, StrainCatalogOrder = savedCatalog, savedOrder })

	for _, s := range strains {
		RegisterStrain(s)
	}
}

func denseStrain(id StrainID, threshold uint8, specificity int) NamedStrain {
	return NamedStrain{
		ID: id, Name: string(id), Rarity: RarityRare, Specificity: specificity,
		Conditions: []GeneCondition{{Gene: plant.GeneDensity, Min: threshold, Max: 255}},
	}
}

func TestPredicateStrainMatchesOnlyQualifyingPlants(t *testing.T) {
	withStrains(t, denseStrain("dense", 200, 10))

	heavy := genomeWith(map[plant.GeneID]plant.Allele{plant.GeneDensity: 240})
	light := genomeWith(map[plant.GeneID]plant.Allele{plant.GeneDensity: 10})

	if _, ok := IdentifyStrain(kindTestCrop, heavy, plant.ExpressFull(heavy)); !ok {
		t.Error("a qualifying plant was not identified")
	}
	if s, ok := IdentifyStrain(kindTestCrop, light, plant.ExpressFull(light)); ok {
		t.Errorf("a non-qualifying plant was identified as %q", s.Name)
	}
}

func TestSignatureStrainMatchesOnlyItsExactGenome(t *testing.T) {
	target := plant.RandomGenome(1234)
	withStrains(t, NamedStrain{
		ID: "exact", Name: "Exact", Rarity: RarityLegendary,
		Specificity: 50, Signature: &target,
	})

	if _, ok := IdentifyStrain(kindTestCrop, target, plant.ExpressFull(target)); !ok {
		t.Fatal("the signature genome was not identified")
	}

	// One allele off must not match: a signature strain is exactly one genome.
	near := target
	near[plant.GeneStemHeight].A++
	if _, ok := IdentifyStrain(kindTestCrop, near, plant.ExpressFull(near)); ok {
		t.Error("a genome one allele away matched the signature")
	}
}

func TestMostSpecificStrainWins(t *testing.T) {
	withStrains(t,
		denseStrain("broad", 150, 10),
		denseStrain("narrow", 200, 25),
	)
	g := genomeWith(map[plant.GeneID]plant.Allele{plant.GeneDensity: 250})

	got, ok := IdentifyStrain(kindTestCrop, g, plant.ExpressFull(g))
	if !ok {
		t.Fatal("no strain matched")
	}
	if got.ID != "narrow" {
		t.Errorf("matched %q, want the more specific \"narrow\"", got.ID)
	}
}

// TestIdentifyStrainIsStableAcrossCalls guards the ordered-iteration rule.
// Resolving through the catalog map would let a plant name itself differently
// between frames, which is a miserable bug to chase.
func TestIdentifyStrainIsStableAcrossCalls(t *testing.T) {
	withStrains(t,
		denseStrain("a", 100, 10),
		denseStrain("b", 100, 10),
		denseStrain("c", 100, 10),
	)
	g := genomeWith(map[plant.GeneID]plant.Allele{plant.GeneDensity: 200})
	p := plant.ExpressFull(g)

	first, ok := IdentifyStrain(kindTestCrop, g, p)
	if !ok {
		t.Fatal("no strain matched")
	}
	for i := 0; i < 200; i++ {
		got, _ := IdentifyStrain(kindTestCrop, g, p)
		if got.ID != first.ID {
			t.Fatalf("identification is unstable: %q then %q", first.ID, got.ID)
		}
	}
}

func TestStrainsAreScopedToTheirSpecies(t *testing.T) {
	withStrains(t, NamedStrain{
		ID: "otherspecies", Name: "Other", Kind: "not_the_test_crop", Specificity: 10,
		Conditions: []GeneCondition{{Gene: plant.GeneDensity, Min: 0, Max: 255}},
	})
	g := plant.DefaultGenome()
	if s, ok := IdentifyStrain(kindTestCrop, g, plant.ExpressFull(g)); ok {
		t.Errorf("a strain for another species matched: %q", s.Name)
	}
}

func TestStrainNameFallsBackToTheSpecies(t *testing.T) {
	withStrains(t, denseStrain("dense", 250, 10))
	g := genomeWith(map[plant.GeneID]plant.Allele{plant.GeneDensity: 0})
	if got := StrainName(kindTestCrop, g, plant.ExpressFull(g), "Test Crop"); got != "Test Crop" {
		t.Errorf("StrainName = %q, want the species fallback", got)
	}
}

func TestDiscoveryLogRecordsEachStrainOnce(t *testing.T) {
	withStrains(t, denseStrain("dense", 100, 10))
	s := NewGameState()
	g := genomeWith(map[plant.GeneID]plant.Allele{plant.GeneDensity: 200})
	p := plant.ExpressFull(g)

	if _, isNew := s.DiscoverPlant(kindTestCrop, g, p); !isNew {
		t.Error("the first sighting was not reported as new")
	}
	if _, isNew := s.DiscoverPlant(kindTestCrop, g, p); isNew {
		t.Error("a repeat sighting was reported as new")
	}
	if s.DiscoveredCount() != 1 {
		t.Errorf("DiscoveredCount = %d, want 1", s.DiscoveredCount())
	}
}

// TestDiscoveredCountIgnoresUnknownStrains covers a save written by a build
// that knew strains this one does not.
func TestDiscoveredCountIgnoresUnknownStrains(t *testing.T) {
	s := NewGameState()
	s.DiscoveredStrains = map[StrainID]bool{"from_a_future_build": true}
	if got := s.DiscoveredCount(); got != 0 {
		t.Errorf("DiscoveredCount = %d, want 0", got)
	}
}

func TestRegisterStrainRejectsMalformedEntries(t *testing.T) {
	always := []GeneCondition{{Gene: plant.GeneDensity, Min: 0, Max: 255}}
	g := plant.DefaultGenome()
	cases := map[string]NamedStrain{
		"no ID":           {Name: "x", Conditions: always},
		"no name":         {ID: "x", Conditions: always},
		"neither matcher": {ID: "x", Name: "x"},
		"both matchers":   {ID: "x", Name: "x", Conditions: always, Signature: &g},
		"unknown gene":    {ID: "x", Name: "x", Conditions: []GeneCondition{{Gene: plant.GeneID(-1), Max: 255}}},
		"inverted band":   {ID: "x", Name: "x", Conditions: []GeneCondition{{Gene: plant.GeneDensity, Min: 200, Max: 100}}},
	}
	for name, s := range cases {
		t.Run(name, func(t *testing.T) {
			withStrains(t)
			defer func() {
				if recover() == nil {
					t.Errorf("RegisterStrain accepted a strain with %s", name)
				}
			}()
			RegisterStrain(s)
		})
	}
}
