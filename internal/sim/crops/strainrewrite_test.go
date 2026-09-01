package crops

import (
	"testing"

	"atomicfarming/internal/plant"
	"atomicfarming/internal/sim"
)

// The four bred Stem strains were originally written as Go closures. They are
// now data, so the game can report how far a plant is from each one.
//
// Restating a shipped rule risks changing what it means without anyone
// noticing — a strain that quietly became easier or impossible would look
// exactly like normal play. The original closures are kept here verbatim as
// the reference, and the rewritten conditions are checked against them across
// the genome space.
var originalPredicates = map[sim.StrainID]func(p plant.Phenotype) bool{
	"ironstem": func(p plant.Phenotype) bool {
		return p.Get(plant.GeneDensity) >= 210 && p.Get(plant.GeneStemThickness) >= 200
	},
	"sunspire": func(p plant.Phenotype) bool {
		return p.Get(plant.GeneStemHeight) >= 235 && p.Get(plant.GeneGrowthRate) >= 210
	},
	"palewood": func(p plant.Phenotype) bool {
		return p.Get(plant.GeneStemSat) <= 40 && p.Get(plant.GeneStemLum) >= 205
	},
	"gnarlroot": func(p plant.Phenotype) bool {
		curve := p.Get(plant.GeneStemCurve)
		return (curve <= 25 || curve >= 230) && p.Get(plant.GeneStemHeight) <= 90
	},
}

func TestRewrittenStrainsMatchTheOriginalPredicates(t *testing.T) {
	stem := &Stem{}
	ranges := stem.Ranges()

	for id, original := range originalPredicates {
		strain, ok := sim.StrainCatalog[id]
		if !ok {
			t.Errorf("strain %q is no longer registered", id)
			continue
		}
		if len(strain.Conditions) == 0 {
			t.Errorf("strain %q has no conditions", id)
			continue
		}

		agreed, matched := 0, 0
		for seed := uint64(0); seed < 40000; seed++ {
			g := homozygousGenome(seed)
			p := plant.Express(g, ranges)

			want := original(p)
			got := strain.Matches(g, p)
			if got != want {
				t.Fatalf("strain %q disagrees with its original rule at seed %d: rewritten=%v original=%v",
					id, seed, got, want)
			}
			agreed++
			if want {
				matched++
			}
		}
		// A rule nothing ever satisfies would agree trivially.
		if matched == 0 {
			t.Errorf("strain %q never matched across %d samples; the comparison proves nothing", id, agreed)
		}
		t.Logf("%s: %d/%d samples matched, rewritten rule agrees throughout", id, matched, agreed)
	}
}

// TestGnarlrootStillNeedsABentStem pins the one rule that changed shape: an
// either-way test became a single inverted band, and an off-by-one at either
// edge would silently move the strain.
func TestGnarlrootStillNeedsABentStem(t *testing.T) {
	strain := sim.StrainCatalog["gnarlroot"]

	for _, tc := range []struct {
		curve uint8
		want  bool
	}{
		{0, true}, {25, true}, {26, false}, {128, false},
		{229, false}, {230, true}, {255, true},
	} {
		var p plant.Phenotype
		p[plant.GeneStemCurve] = tc.curve
		p[plant.GeneStemHeight] = 50 // satisfies the other condition

		if got := strain.Matches(plant.Genome{}, p); got != tc.want {
			t.Errorf("curve %d: matched=%v, want %v", tc.curve, got, tc.want)
		}
	}
}
