package ui

import (
	"testing"

	"atomicfarming/internal/plant"
)

func TestLabStartsOnTheDefaultSpecimen(t *testing.T) {
	l := NewLabState()
	if l.Genome != plant.DefaultGenome() {
		t.Error("the bench does not start with the default genome")
	}
	if l.Maturity != 1 {
		t.Errorf("Maturity = %v, want 1 so the specimen is visible immediately", l.Maturity)
	}
}

func TestRandomiseAndMutateChangeTheSpecimen(t *testing.T) {
	l := NewLabState()
	before := l.Genome

	l.Randomise()
	if l.Genome == before {
		t.Fatal("Randomise left the genome unchanged")
	}
	afterRandom := l.Genome

	// Successive rolls must differ, or the controls would appear stuck.
	l.Randomise()
	if l.Genome == afterRandom {
		t.Error("two Randomise calls produced the same genome")
	}

	l.Genome = afterRandom
	l.Mutate()
	if l.Genome == afterRandom {
		t.Error("Mutate left the genome unchanged")
	}

	l.Reset()
	if l.Genome != plant.DefaultGenome() {
		t.Error("Reset did not restore the default genome")
	}
}

func TestSelectGeneWrapsAtBothEnds(t *testing.T) {
	l := NewLabState()
	l.SelectGene(-1)
	if int(l.Selected) != plant.GeneCount-1 {
		t.Errorf("stepping back from the first gene gave %d, want %d", l.Selected, plant.GeneCount-1)
	}
	l.SelectGene(1)
	if l.Selected != 0 {
		t.Errorf("stepping forward from the last gene gave %d, want 0", l.Selected)
	}
}

func TestAdjustGeneClampsToTheAlleleRange(t *testing.T) {
	l := NewLabState()
	l.Selected = plant.GeneStemHeight

	l.Genome[plant.GeneStemHeight] = plant.GenePair{A: 254, B: 1}
	l.AdjustGene(10, false)
	if got := l.Genome[plant.GeneStemHeight].A; got != 255 {
		t.Errorf("allele A = %d, want it clamped to 255", got)
	}
	l.AdjustGene(-10, true)
	if got := l.Genome[plant.GeneStemHeight].B; got != 0 {
		t.Errorf("allele B = %d, want it clamped to 0", got)
	}

	// Each allele is adjusted independently.
	l.Genome[plant.GeneStemHeight] = plant.GenePair{A: 100, B: 100}
	l.AdjustGene(5, false)
	if l.Genome[plant.GeneStemHeight].B != 100 {
		t.Error("adjusting allele A also moved allele B")
	}
}

func TestScrubMaturityClampsAndStopsAutoGrow(t *testing.T) {
	l := NewLabState()
	l.AutoGrow = true

	l.ScrubMaturity(-5)
	if l.Maturity != 0 {
		t.Errorf("Maturity = %v, want 0", l.Maturity)
	}
	if l.AutoGrow {
		t.Error("scrubbing did not take manual control from auto-grow")
	}

	l.ScrubMaturity(5)
	if l.Maturity != 1 {
		t.Errorf("Maturity = %v, want 1", l.Maturity)
	}
}

func TestTickOnlyAnimatesAnOpenAutoGrowingLab(t *testing.T) {
	l := NewLabState()
	l.Maturity = 0.5

	l.Tick() // closed, not growing
	if l.Maturity != 0.5 {
		t.Error("a closed lab animated")
	}

	l.Open = true
	l.Tick()
	if l.Maturity != 0.5 {
		t.Error("an open lab animated without auto-grow")
	}

	l.AutoGrow = true
	l.Tick()
	if l.Maturity <= 0.5 {
		t.Error("auto-grow did not advance maturity")
	}

	l.Maturity = 1
	l.Tick()
	if l.Maturity != 0 {
		t.Errorf("auto-grow did not loop at the end of life: %v", l.Maturity)
	}
}

func TestBreedNeedsBothParents(t *testing.T) {
	l := NewLabState()
	if l.CanBreed() || l.Breed() {
		t.Fatal("bred with no parents stored")
	}

	l.Randomise()
	l.StoreParentA()
	if l.CanBreed() || l.Breed() {
		t.Fatal("bred with only one parent stored")
	}

	l.Randomise()
	l.StoreParentB()
	if !l.CanBreed() {
		t.Fatal("CanBreed is false with both parents stored")
	}
	if !l.Breed() {
		t.Fatal("Breed failed with both parents stored")
	}

	// Every allele of the offspring must trace to a parent, mutation aside.
	for i := 0; i < plant.GeneCount; i++ {
		child := l.Genome[i]
		a, b := l.ParentA[i], l.ParentB[i]
		if !near(child.A, a.A) && !near(child.A, a.B) {
			t.Fatalf("gene %d: allele A came from neither of parent A's", i)
		}
		if !near(child.B, b.A) && !near(child.B, b.B) {
			t.Fatalf("gene %d: allele B came from neither of parent B's", i)
		}
	}
}

// near allows for one mutation step, since Breed mutates.
func near(got, want plant.Allele) bool {
	d := int(got) - int(want)
	return d >= -plant.MaxMutationStep && d <= plant.MaxMutationStep
}

func TestMutationPreviewsAreStableNeighbours(t *testing.T) {
	l := NewLabState()
	l.Randomise()

	previews := l.MutationPreviews()
	if len(previews) != MutationPreviewCount {
		t.Fatalf("got %d previews, want %d", len(previews), MutationPreviewCount)
	}

	// The grid must hold still while a gene is being adjusted, or it would be
	// useless for judging what one change did.
	again := l.MutationPreviews()
	for i := range previews {
		if previews[i] != again[i] {
			t.Fatalf("preview %d changed between calls", i)
		}
	}

	distinct := map[string]bool{}
	for _, g := range previews {
		distinct[g.String()] = true
		for i := 0; i < plant.GeneCount; i++ {
			if !near(g[i].A, l.Genome[i].A) || !near(g[i].B, l.Genome[i].B) {
				t.Fatalf("gene %d of a preview is more than one mutation from the specimen", i)
			}
		}
	}
	if len(distinct) < MutationPreviewCount-1 {
		t.Errorf("only %d distinct previews; the grid would show the same plant repeatedly", len(distinct))
	}
}
