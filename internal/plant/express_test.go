package plant

import "testing"

func TestExpressionCollapseRules(t *testing.T) {
	pair := GenePair{A: 200, B: 40}
	tests := []struct {
		expr Expression
		want uint8
	}{
		{ExprAverage, 120},
		{ExprDominant, 200},
		{ExprRecessive, 40},
	}
	for _, tc := range tests {
		if got := tc.expr.Collapse(pair); got != tc.want {
			t.Errorf("%v.Collapse(%+v) = %d, want %d", tc.expr, pair, got, tc.want)
		}
		// Order of alleles within the pair must not matter.
		flipped := GenePair{A: pair.B, B: pair.A}
		if got := tc.expr.Collapse(flipped); got != tc.want {
			t.Errorf("%v is order-dependent: flipped gave %d, want %d", tc.expr, got, tc.want)
		}
	}
}

func TestRemapHitsBothEndsOfTheSpeciesRange(t *testing.T) {
	r := GeneRange{Min: 60, Max: 200}
	if got := remap(0, r); got != 60 {
		t.Errorf("remap(0) = %d, want 60 — the low end is unreachable", got)
	}
	if got := remap(255, r); got != 200 {
		t.Errorf("remap(255) = %d, want 200 — the high end is unreachable", got)
	}
	if got := remap(128, r); got < 125 || got > 135 {
		t.Errorf("remap(128) = %d, want near the midpoint 130", got)
	}
	if got := remap(200, GeneRange{Min: 77, Max: 77}); got != 77 {
		t.Errorf("degenerate range gave %d, want 77", got)
	}
	// An inverted range is tolerated rather than producing nonsense.
	if got := remap(255, GeneRange{Min: 200, Max: 60}); got != 200 {
		t.Errorf("inverted range gave %d, want 200", got)
	}
}

func TestExpressAppliesSpeciesRanges(t *testing.T) {
	g := DefaultGenome()
	g[GeneStemHeight] = GenePair{A: 255, B: 255}

	narrow := FullRange()
	narrow[GeneStemHeight] = GeneRange{Min: 100, Max: 120}

	if got := Express(g, narrow).Get(GeneStemHeight); got != 120 {
		t.Errorf("constrained height = %d, want 120", got)
	}
	if got := ExpressFull(g).Get(GeneStemHeight); got != 255 {
		t.Errorf("unconstrained height = %d, want 255", got)
	}
}

func TestChoiceCoversEveryBucketEvenly(t *testing.T) {
	const buckets = 6
	counts := make([]int, buckets)
	for v := 0; v < 256; v++ {
		var p Phenotype
		p[GeneLeafArchetype] = uint8(v)
		c := p.Choice(GeneLeafArchetype, buckets)
		if c < 0 || c >= buckets {
			t.Fatalf("Choice(%d) = %d, out of range", v, c)
		}
		counts[c]++
	}
	for i, n := range counts {
		if n == 0 {
			t.Errorf("bucket %d is unreachable", i)
		}
		if n < 256/buckets-2 || n > 256/buckets+2 {
			t.Errorf("bucket %d has %d values, want about %d", i, n, 256/buckets)
		}
	}
}

func TestPhenotypeAccessors(t *testing.T) {
	var p Phenotype
	p[GeneStemHeight] = 255
	if got := p.Unit(GeneStemHeight); got != 1 {
		t.Errorf("Unit = %v, want 1", got)
	}
	if got := p.Scaled(GeneStemHeight, 10, 20); got != 20 {
		t.Errorf("Scaled = %d, want 20", got)
	}
	p[GeneStemHeight] = 0
	if got := p.Scaled(GeneStemHeight, 10, 20); got != 10 {
		t.Errorf("Scaled = %d, want 10", got)
	}
	if got := p.Signed(GeneStemHeight); got != -1 {
		t.Errorf("Signed(0) = %v, want -1", got)
	}
	p[GeneStemHeight] = 255
	if got := p.Signed(GeneStemHeight); got != 1 {
		t.Errorf("Signed(255) = %v, want 1", got)
	}
	// An invalid gene reads as zero rather than panicking.
	if got := p.Get(GeneID(-1)); got != 0 {
		t.Errorf("Get(invalid) = %d, want 0", got)
	}
}

func TestDefaultGenomeIsAViablePlant(t *testing.T) {
	g := DefaultGenome()
	if g.IsZero() {
		t.Fatal("DefaultGenome is all zeroes, which is the sentinel for an unstamped genome")
	}
	p := ExpressFull(g)
	if p.FlowerArchetype() == FlowerNone {
		t.Error("the default plant has no flower; it should be recognisably a plant")
	}
	if p.Get(GeneStemHeight) == 0 {
		t.Error("the default plant has no stem height")
	}
	for i := 0; i < GeneCount; i++ {
		if !g[i].Homozygous() {
			t.Errorf("gene %s is not homozygous in the default genome", GeneCatalog[i].Name)
		}
	}
}

func TestGeneCatalogCoversEveryGroup(t *testing.T) {
	for grp := GeneGroup(0); grp < GroupCount; grp++ {
		if len(GenesInGroup(grp)) == 0 {
			t.Errorf("group %s has no genes", grp)
		}
	}
}
