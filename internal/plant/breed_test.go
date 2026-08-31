package plant

import "testing"

func homozygous(gene GeneID, v Allele) Genome {
	g := DefaultGenome()
	g[gene] = GenePair{A: v, B: v}
	return g
}

func TestMeiosisDrawsEveryAlleleFromAParent(t *testing.T) {
	a := RandomGenome(11)
	b := RandomGenome(22)
	for seed := uint64(0); seed < 64; seed++ {
		child := Meiosis(a, b, seed)
		for i := 0; i < GeneCount; i++ {
			if child[i].A != a[i].A && child[i].A != a[i].B {
				t.Fatalf("gene %d: allele A %d came from neither of parent A's alleles", i, child[i].A)
			}
			if child[i].B != b[i].A && child[i].B != b[i].B {
				t.Fatalf("gene %d: allele B %d came from neither of parent B's alleles", i, child[i].B)
			}
		}
	}
}

func TestHomozygousCrossYieldsTheExpectedHeterozygote(t *testing.T) {
	const gene = GeneDensity
	a := homozygous(gene, 240)
	b := homozygous(gene, 40)
	for seed := uint64(0); seed < 32; seed++ {
		child := Meiosis(a, b, seed)
		if child[gene] != (GenePair{A: 240, B: 40}) {
			t.Fatalf("seed %d: child pair = %+v, want {240 40}", seed, child[gene])
		}
		// Co-dominant expression puts the visible value at the midpoint...
		if got := ExpressFull(child).Get(gene); got != 140 {
			t.Errorf("expressed density = %d, want 140", got)
		}
	}
}

// TestExtremeAllelesSurviveGenerations is the test that justifies the diploid
// model. Under a blending scheme, crossing a high and a low line collapses the
// range within a few generations and no descendant can ever beat its best
// ancestor. Under diploid meiosis the extreme alleles pass down intact and can
// be recovered as a homozygote.
func TestExtremeAllelesSurviveGenerations(t *testing.T) {
	const (
		gene        = GeneDensity
		high        = Allele(240)
		low         = Allele(40)
		population  = 24
		generations = 12
	)

	pop := make([]Genome, population)
	for i := range pop {
		if i%2 == 0 {
			pop[i] = homozygous(gene, high)
		} else {
			pop[i] = homozygous(gene, low)
		}
	}

	for gen := 0; gen < generations; gen++ {
		next := make([]Genome, population)
		for i := range next {
			a := pop[i%population]
			b := pop[(i*7+3)%population]
			next[i] = Meiosis(a, b, uint64(gen)*7919+uint64(i))
		}
		pop = next
	}

	var seenHighHomozygote, seenLowAllele bool
	bestExpressed := 0
	for _, g := range pop {
		if g[gene].A == high && g[gene].B == high {
			seenHighHomozygote = true
		}
		if g[gene].A == low || g[gene].B == low {
			seenLowAllele = true
		}
		if v := int(ExpressFull(g).Get(gene)); v > bestExpressed {
			bestExpressed = v
		}
		// With no mutation, no allele may be invented.
		if (g[gene].A != high && g[gene].A != low) || (g[gene].B != high && g[gene].B != low) {
			t.Fatalf("meiosis invented an allele: %+v", g[gene])
		}
	}

	if !seenHighHomozygote {
		t.Errorf("after %d generations no high homozygote reappeared; extremes are not recoverable", generations)
	}
	if !seenLowAllele {
		t.Errorf("the low allele was lost from the pool entirely")
	}
	if bestExpressed < int(high) {
		t.Errorf("best expressed density is %d, never reaching the parental extreme %d", bestExpressed, high)
	}

	// Contrast: a blending model averages the two lines to a single value in
	// one generation and can never recover either extreme.
	blended := (int(high) + int(low)) / 2
	if blended >= int(high) {
		t.Fatal("test is misconfigured: the blend should sit strictly below the high parent")
	}
}

func TestMutateIsIdentityAtZeroRate(t *testing.T) {
	g := RandomGenome(5)
	if got := Mutate(g, 99, 0); got != g {
		t.Error("Mutate at 0 bp changed the genome")
	}
	if got := Mutate(g, 99, -100); got != g {
		t.Error("Mutate at a negative rate changed the genome")
	}
}

func TestMutateMovesAllelesBySmallSteps(t *testing.T) {
	g := RandomGenome(5)
	// A high rate so most alleles move, making the bound meaningful.
	mutated := Mutate(g, 77, BasisPoints/2)

	changed := 0
	for i := 0; i < GeneCount; i++ {
		for _, pair := range [][2]Allele{{g[i].A, mutated[i].A}, {g[i].B, mutated[i].B}} {
			before, after := int(pair[0]), int(pair[1])
			if before != after {
				changed++
			}
			if d := after - before; d > MaxMutationStep || d < -MaxMutationStep {
				t.Fatalf("gene %d moved by %d, beyond MaxMutationStep %d", i, d, MaxMutationStep)
			}
			if after < 0 || after > 255 {
				t.Fatalf("gene %d escaped the allele range: %d", i, after)
			}
		}
	}
	if changed == 0 {
		t.Error("no allele mutated at a 50% rate")
	}
}

func TestMutateClampsAtTheAlleleBoundaries(t *testing.T) {
	var g Genome
	for i := 0; i < GeneCount; i++ {
		g[i] = GenePair{A: 0, B: 255}
	}
	mutated := Mutate(g, 3, BasisPoints) // every allele mutates
	for i := 0; i < GeneCount; i++ {
		if mutated[i].A > MaxMutationStep {
			t.Fatalf("gene %d: allele at 0 jumped to %d", i, mutated[i].A)
		}
		if int(mutated[i].B) < 255-MaxMutationStep {
			t.Fatalf("gene %d: allele at 255 dropped to %d", i, mutated[i].B)
		}
	}
}

func TestMutationRateBPTracksMutabilityAndStaysSmall(t *testing.T) {
	lowP := ExpressFull(homozygous(GeneMutability, 0))
	highP := ExpressFull(homozygous(GeneMutability, 255))

	low := MutationRateBP(lowP, lowP, 0)
	high := MutationRateBP(highP, highP, 0)

	if low != 0 {
		t.Errorf("minimum mutability gave %d bp, want 0", low)
	}
	if high <= low {
		t.Errorf("maximum mutability (%d bp) did not exceed minimum (%d bp)", high, low)
	}
	if high > 100 {
		t.Errorf("maximum genetic mutation rate is %d bp (>1%%); mutations should be rare", high)
	}
	if withBonus := MutationRateBP(lowP, lowP, 250); withBonus != 250 {
		t.Errorf("global bonus gave %d bp, want 250", withBonus)
	}
	if clamped := MutationRateBP(highP, highP, BasisPoints*2); clamped != BasisPoints {
		t.Errorf("rate was not clamped: %d", clamped)
	}
}

func TestBreedIsDeterministic(t *testing.T) {
	a, b := RandomGenome(1), RandomGenome(2)
	r := FullRange()
	first := Breed(a, b, r, 42, 0)
	for i := 0; i < 8; i++ {
		if got := Breed(a, b, r, 42, 0); got != first {
			t.Fatal("Breed is not deterministic for a fixed seed")
		}
	}
	if other := Breed(a, b, r, 43, 0); other == first {
		t.Error("Breed produced identical offspring for different seeds")
	}
}
