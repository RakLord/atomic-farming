package morph

import (
	"math"
	"testing"

	"atomicfarming/internal/plant"
)

var maturities = []float64{0.1, 0.3, 0.45, 0.6, 0.72, 0.85, 0.95, 1.0}

func TestBuildIsDeterministic(t *testing.T) {
	for seed := uint64(0); seed < 200; seed++ {
		p := plant.ExpressFull(plant.RandomGenome(seed))
		for _, mt := range maturities {
			first := Build(p, mt)
			again := Build(p, mt)
			if len(first.Shapes) != len(again.Shapes) {
				t.Fatalf("seed %d t=%v: shape count differs between builds", seed, mt)
			}
			a, b := first.Points(), again.Points()
			for i := range a {
				if a[i] != b[i] {
					t.Fatalf("seed %d t=%v: point %d differs between builds", seed, mt, i)
				}
			}
		}
	}
}

func TestBlueprintStaysWithinCanvas(t *testing.T) {
	for seed := uint64(0); seed < 3000; seed++ {
		p := plant.ExpressFull(plant.RandomGenome(seed))
		for _, mt := range maturities {
			for _, pt := range Build(p, mt).Points() {
				if pt.X < CanvasMinX || pt.X > CanvasMaxX || pt.Y < CanvasMinY || pt.Y > CanvasMaxY {
					t.Fatalf("seed %d t=%v (stem=%v leaf=%v): point (%.3f, %.3f) escapes the canvas",
						seed, mt, p.StemArchetype(), p.LeafArchetype(), pt.X, pt.Y)
				}
			}
		}
	}
}

func TestNothingGrowsBeforeItIsSown(t *testing.T) {
	p := plant.ExpressFull(plant.DefaultGenome())
	if got := Build(p, 0); len(got.Shapes) != 0 {
		t.Errorf("a plant at t=0 has %d shapes, want none", len(got.Shapes))
	}
	if got := Build(p, -1); len(got.Shapes) != 0 {
		t.Errorf("a plant at negative maturity has %d shapes, want none", len(got.Shapes))
	}
}

// TestGrowthIsMonotonic checks that a plant only ever gains structure as it
// matures. A plant that shrinks or loses leaves mid-growth reads as a glitch.
func TestGrowthIsMonotonic(t *testing.T) {
	for seed := uint64(0); seed < 300; seed++ {
		p := plant.ExpressFull(plant.RandomGenome(seed))
		prevShapes := 0
		prevTop := 0.0
		for _, mt := range maturities {
			bp := Build(p, mt)
			if len(bp.Shapes) < prevShapes {
				t.Fatalf("seed %d: shape count fell from %d to %d at t=%v",
					seed, prevShapes, len(bp.Shapes), mt)
			}
			// Strict: a plant must never lose height as it matures. Any
			// regression at all means some element is being re-laid-out
			// rather than added to, which reads as a glitch.
			if _, hi, ok := bp.Bounds(); ok {
				if hi.Y < prevTop {
					t.Fatalf("seed %d: plant top fell from %.4f to %.4f at t=%v",
						seed, prevTop, hi.Y, mt)
				}
				prevTop = math.Max(prevTop, hi.Y)
			}
			prevShapes = len(bp.Shapes)
		}
	}
}

func TestShapesArePaintedBackToFront(t *testing.T) {
	for seed := uint64(0); seed < 100; seed++ {
		p := plant.ExpressFull(plant.RandomGenome(seed))
		bp := Build(p, 1)
		for i := 1; i < len(bp.Shapes); i++ {
			if bp.Shapes[i].Z < bp.Shapes[i-1].Z {
				t.Fatalf("seed %d: shape %d has Z %d, behind its predecessor's %d",
					seed, i, bp.Shapes[i].Z, bp.Shapes[i-1].Z)
			}
		}
	}
}

func TestEveryArchetypeProducesGeometry(t *testing.T) {
	for stem := plant.StemArchetype(0); stem < plant.StemArchetypeCount; stem++ {
		for leaf := plant.LeafArchetype(0); leaf < plant.LeafArchetypeCount; leaf++ {
			p := plant.ExpressFull(plant.DefaultGenome())
			p[plant.GeneStemArchetype] = archetypeValue(int(stem), int(plant.StemArchetypeCount))
			p[plant.GeneLeafArchetype] = archetypeValue(int(leaf), int(plant.LeafArchetypeCount))
			if got := p.StemArchetype(); got != stem {
				t.Fatalf("archetypeValue did not select stem %v, got %v", stem, got)
			}
			bp := Build(p, 1)
			if len(bp.Shapes) < 2 {
				t.Errorf("stem=%v leaf=%v produced only %d shapes", stem, leaf, len(bp.Shapes))
			}
		}
	}

	for flower := plant.FlowerArchetype(0); flower < plant.FlowerArchetypeCount; flower++ {
		p := plant.ExpressFull(plant.DefaultGenome())
		p[plant.GeneFlowerArchetype] = archetypeValue(int(flower), int(plant.FlowerArchetypeCount))
		bp := Build(p, 1)
		if len(bp.Shapes) == 0 {
			t.Errorf("flower=%v produced no shapes at all", flower)
		}
	}

	for fruit := plant.FruitArchetype(0); fruit < plant.FruitArchetypeCount; fruit++ {
		p := plant.ExpressFull(plant.DefaultGenome())
		p[plant.GeneFruitArchetype] = archetypeValue(int(fruit), int(plant.FruitArchetypeCount))
		if got := Build(p, 1); len(got.Shapes) == 0 {
			t.Errorf("fruit=%v produced no shapes at all", fruit)
		}
	}
}

// archetypeValue returns the phenotype value at the centre of bucket i.
func archetypeValue(i, n int) uint8 {
	return uint8((i*256 + 128) / n)
}
