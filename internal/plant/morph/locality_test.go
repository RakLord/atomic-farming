package morph

import (
	"math"
	"testing"

	"atomicfarming/internal/plant"
)

// discreteGenes are the genes whose effect is inherently stepwise: the four
// archetype selectors, and the counts of nodes, leaves, petals and fruit. A
// one-step change to any of them can legitimately cross a bucket boundary and
// change the plant wholesale, so they are excluded from the locality check.
// Everything else must vary continuously.
var discreteGenes = map[plant.GeneID]bool{
	plant.GeneStemArchetype:   true,
	plant.GeneLeafArchetype:   true,
	plant.GeneFlowerArchetype: true,
	plant.GeneFruitArchetype:  true,
	plant.GeneNodeCount:       true,
	plant.GeneLeafPerNode:     true,
	plant.GenePetalCount:      true,
	plant.GeneFruitCount:      true,
}

// maxLocalDisplacement is how far a single point may move when one allele
// shifts by one step, in normalised plant units. The canvas is 1.55 tall, so
// this is well under one percent of the plant.
const maxLocalDisplacement = 0.012

// TestOneStepMutationBarelyMovesGeometry is the requirement — small genome
// change, small visual change — stated as an assertion.
//
// It is the property most easily broken by accident: hashing the genome to
// seed jitter, or deriving any layout from a bucketed value, would satisfy
// every other test in this package while making a mutated plant look nothing
// like its parent. See the comment on wobble.
func TestOneStepMutationBarelyMovesGeometry(t *testing.T) {
	worst, worstGene := 0.0, plant.GeneID(0)

	for seed := uint64(0); seed < 120; seed++ {
		base := plant.RandomGenome(seed)
		for gene := plant.GeneID(0); int(gene) < plant.GeneCount; gene++ {
			if discreteGenes[gene] {
				continue
			}
			step := base
			if step[gene].A < 255 {
				step[gene].A++
			} else {
				step[gene].A--
			}

			for _, mt := range []float64{0.5, 0.8, 1.0} {
				before := Build(plant.ExpressFull(base), mt)
				after := Build(plant.ExpressFull(step), mt)

				if len(before.Shapes) != len(after.Shapes) {
					t.Fatalf("seed %d gene %s: shape count changed from %d to %d for a one-step mutation",
						seed, plant.GeneCatalog[gene].Name, len(before.Shapes), len(after.Shapes))
				}

				a, b := before.Points(), after.Points()
				if len(a) != len(b) {
					t.Fatalf("seed %d gene %s: point count changed for a one-step mutation",
						seed, plant.GeneCatalog[gene].Name)
				}
				for i := range a {
					d := math.Hypot(a[i].X-b[i].X, a[i].Y-b[i].Y)
					if d > worst {
						worst, worstGene = d, gene
					}
					if d > maxLocalDisplacement {
						t.Fatalf("seed %d gene %s at t=%v: a one-step mutation moved a point by %.4f (limit %.4f)",
							seed, plant.GeneCatalog[gene].Name, mt, d, maxLocalDisplacement)
					}
				}
			}
		}
	}
	t.Logf("worst one-step displacement: %.5f (%s)", worst, plant.GeneCatalog[worstGene].Name)
}

// TestJitterVariesByElementNotByHash is the other half of the locality story:
// the noise must still produce genuine variety between elements, or every
// leaf on a plant would sit at an identical offset.
func TestJitterVariesByElementNotByHash(t *testing.T) {
	p := plant.ExpressFull(plant.RandomGenome(5))

	seen := map[float64]bool{}
	for i := 0; i < 12; i++ {
		seen[math.Round(wobble(p, i, 1)*1000)] = true
	}
	if len(seen) < 10 {
		t.Errorf("wobble produced only %d distinct values across 12 indices; foliage would look uniform", len(seen))
	}

	// And the same index under a different channel must differ, or leaves and
	// branches at one node would share an offset.
	if wobble(p, 3, 1) == wobble(p, 3, 2) {
		t.Error("wobble ignores its channel; different features at one node would move together")
	}
}
