package plant

import "testing"

func TestMutateOnceNeverFiresAtZero(t *testing.T) {
	g := RandomGenome(3)
	for seed := uint64(0); seed < 500; seed++ {
		if got, fired := MutateOnce(g, seed, 0); fired || got != g {
			t.Fatal("MutateOnce fired at zero chance")
		}
		if got, fired := MutateOnce(g, seed, -50); fired || got != g {
			t.Fatal("MutateOnce fired at a negative chance")
		}
	}
}

// TestMutateOnceChangesExactlyOneAlleleByOneStep is the whole point of the
// function: a mutation is a discrete event, not a smear across the genome.
func TestMutateOnceChangesExactlyOneAlleleByOneStep(t *testing.T) {
	base := RandomGenome(11)
	fired := 0

	for seed := uint64(0); seed < 3000; seed++ {
		got, hit := MutateOnce(base, seed, PartsPerMillion) // always fires
		if !hit {
			t.Fatal("MutateOnce did not fire at certainty")
		}
		fired++

		changes := 0
		for i := 0; i < GeneCount; i++ {
			for _, d := range []int{
				int(got[i].A) - int(base[i].A),
				int(got[i].B) - int(base[i].B),
			} {
				if d == 0 {
					continue
				}
				changes++
				if d != 1 && d != -1 {
					t.Fatalf("gene %d moved by %d, want a single step", i, d)
				}
			}
		}
		if changes != 1 {
			t.Fatalf("seed %d changed %d alleles, want exactly 1", seed, changes)
		}
	}
	if fired == 0 {
		t.Fatal("nothing fired")
	}
}

func TestMutateOnceSpreadsAcrossTheGenome(t *testing.T) {
	base := RandomGenome(12)
	touched := map[int]bool{}
	for seed := uint64(0); seed < 4000; seed++ {
		got, _ := MutateOnce(base, seed, PartsPerMillion)
		for i := 0; i < GeneCount; i++ {
			if got[i] != base[i] {
				touched[i] = true
			}
		}
	}
	if len(touched) != GeneCount {
		t.Errorf("only %d of %d genes were ever mutated; some are unreachable", len(touched), GeneCount)
	}
}

// TestMutateOnceReflectsAtTheBoundaries: clamping would let a mutation fire
// and change nothing, breaking the one-step guarantee.
func TestMutateOnceReflectsAtTheBoundaries(t *testing.T) {
	var floor, ceiling Genome
	for i := 0; i < GeneCount; i++ {
		floor[i] = GenePair{A: 0, B: 0}
		ceiling[i] = GenePair{A: 255, B: 255}
	}

	for seed := uint64(0); seed < 1000; seed++ {
		for name, g := range map[string]Genome{"floor": floor, "ceiling": ceiling} {
			got, fired := MutateOnce(g, seed, PartsPerMillion)
			if !fired {
				t.Fatal("did not fire at certainty")
			}
			if got == g {
				t.Fatalf("%s: a mutation fired but changed nothing", name)
			}
		}
	}
}

func TestMutateOnceHitsItsStatedRate(t *testing.T) {
	base := RandomGenome(13)
	for _, ppm := range []int{1000, 10000, 100000} {
		const trials = 40000
		hits := 0
		for seed := uint64(0); seed < trials; seed++ {
			if _, fired := MutateOnce(base, seed, ppm); fired {
				hits++
			}
		}
		want := trials * ppm / PartsPerMillion
		if diff := hits - want; diff < -want/3-20 || diff > want/3+20 {
			t.Errorf("ppm=%d: %d hits in %d, want about %d", ppm, hits, trials, want)
		}
	}
}

func TestMutateOnceIsDeterministic(t *testing.T) {
	base := RandomGenome(14)
	for seed := uint64(0); seed < 200; seed++ {
		a, fa := MutateOnce(base, seed, 500000)
		b, fb := MutateOnce(base, seed, 500000)
		if a != b || fa != fb {
			t.Fatalf("seed %d is not deterministic", seed)
		}
	}
}

func TestChancePPMBoundaries(t *testing.T) {
	if ChancePPM(1, PurposeMutation, 0, 0) || ChancePPM(1, PurposeMutation, 0, -1) {
		t.Error("ChancePPM fired at or below zero")
	}
	if !ChancePPM(1, PurposeMutation, 0, PartsPerMillion) {
		t.Error("ChancePPM did not fire at certainty")
	}
}
