package sim

import (
	"strings"
	"testing"

	"atomicfarming/internal/plant"
)

func phenoWith(gene plant.GeneID, v uint8) plant.Phenotype {
	var p plant.Phenotype
	p[gene] = v
	return p
}

func TestGeneConditionBoundsAreInclusive(t *testing.T) {
	c := GeneCondition{Gene: plant.GeneDensity, Min: 100, Max: 200}
	for _, tc := range []struct {
		v    uint8
		want bool
	}{
		{99, false}, {100, true}, {150, true}, {200, true}, {201, false},
	} {
		if got := c.Holds(phenoWith(plant.GeneDensity, tc.v)); got != tc.want {
			t.Errorf("value %d: holds=%v, want %v", tc.v, got, tc.want)
		}
	}
}

func TestGeneConditionOutsideInvertsTheBand(t *testing.T) {
	c := GeneCondition{Gene: plant.GeneStemCurve, Min: 26, Max: 229, Outside: true}
	for _, tc := range []struct {
		v    uint8
		want bool
	}{
		{0, true}, {25, true}, {26, false}, {229, false}, {230, true}, {255, true},
	} {
		if got := c.Holds(phenoWith(plant.GeneStemCurve, tc.v)); got != tc.want {
			t.Errorf("value %d: holds=%v, want %v", tc.v, got, tc.want)
		}
	}
}

func TestGeneConditionDescribesEachShape(t *testing.T) {
	tests := []struct {
		cond GeneCondition
		want string
	}{
		{GeneCondition{Gene: plant.GeneDensity, Min: 210, Max: 255}, "Density 210+"},
		{GeneCondition{Gene: plant.GeneStemSat, Min: 0, Max: 40}, "Stem Saturation 40 or less"},
		{GeneCondition{Gene: plant.GeneDensity, Min: 100, Max: 200}, "Density 100-200"},
		{GeneCondition{Gene: plant.GeneStemCurve, Min: 26, Max: 229, Outside: true}, "Stem Curve outside 26-229"},
	}
	for _, tc := range tests {
		if got := tc.cond.Describe(); got != tc.want {
			t.Errorf("Describe() = %q, want %q", got, tc.want)
		}
	}
	// An invalid gene must render rather than panic in the middle of a frame.
	bad := GeneCondition{Gene: plant.GeneID(-1), Max: 255}
	if got := bad.GeneName(); got != "Unknown" {
		t.Errorf("an invalid gene named itself %q", got)
	}
}

func TestStrainProgressMeasuresEachCondition(t *testing.T) {
	withStrains(t, NamedStrain{
		ID: "heavy", Name: "Heavy", Kind: kindTestCrop, Specificity: 10,
		Conditions: []GeneCondition{
			{Gene: plant.GeneDensity, Min: 200, Max: 255},
			{Gene: plant.GeneStemThickness, Min: 200, Max: 255},
		},
	})

	g := genomeWith(map[plant.GeneID]plant.Allele{
		plant.GeneDensity:       120, // short of it
		plant.GeneStemThickness: 240, // met
	})
	progress := StrainProgressFor(kindTestCrop, g, plant.ExpressFull(g))

	if len(progress) != 1 {
		t.Fatalf("got %d strains, want 1", len(progress))
	}
	p := progress[0]
	if p.Met {
		t.Error("a strain with an unmet condition reported as met")
	}
	if !p.Breedable {
		t.Error("a condition strain reported as unbreedable")
	}
	if len(p.Conditions) != 2 {
		t.Fatalf("got %d conditions, want 2", len(p.Conditions))
	}
	if p.Conditions[0].Current != 120 || p.Conditions[0].Met {
		t.Errorf("first condition = %+v, want current 120 and unmet", p.Conditions[0])
	}
	if p.Conditions[1].Current != 240 || !p.Conditions[1].Met {
		t.Errorf("second condition = %+v, want current 240 and met", p.Conditions[1])
	}
}

func TestStrainProgressMarksASatisfiedStrain(t *testing.T) {
	withStrains(t, denseStrain("dense", 200, 10))
	g := genomeWith(map[plant.GeneID]plant.Allele{plant.GeneDensity: 250})

	progress := StrainProgressFor(kindTestCrop, g, plant.ExpressFull(g))
	if len(progress) != 1 || !progress[0].Met {
		t.Errorf("a qualifying plant did not report the strain as met: %+v", progress)
	}
}

// TestSignatureStrainsReportAsUnbreedable: a handcrafted genome has no
// conditions to work toward, and telling a player otherwise would send them
// chasing something unreachable.
func TestSignatureStrainsReportAsUnbreedable(t *testing.T) {
	target := plant.RandomGenome(4321)
	withStrains(t, NamedStrain{
		ID: "exact", Name: "Exact", Kind: kindTestCrop,
		Specificity: 50, Signature: &target,
	})

	progress := StrainProgressFor(kindTestCrop, plant.DefaultGenome(), plant.ExpressFull(plant.DefaultGenome()))
	if len(progress) != 1 {
		t.Fatalf("got %d strains, want 1", len(progress))
	}
	if progress[0].Breedable {
		t.Error("a signature strain reported as breedable")
	}
	if len(progress[0].Conditions) != 0 {
		t.Error("a signature strain listed conditions it does not have")
	}
}

func TestStrainProgressIsScopedAndStable(t *testing.T) {
	withStrains(t,
		denseStrain("mine", 100, 10),
		NamedStrain{ID: "theirs", Name: "Theirs", Kind: "another_species", Specificity: 10,
			Conditions: []GeneCondition{{Gene: plant.GeneDensity, Min: 0, Max: 255}}},
	)
	g := plant.DefaultGenome()
	p := plant.ExpressFull(g)

	first := StrainProgressFor(kindTestCrop, g, p)
	if len(first) != 1 || first[0].Strain.ID != "mine" {
		t.Fatalf("progress = %+v, want only this species' strain", first)
	}
	for i := 0; i < 50; i++ {
		got := StrainProgressFor(kindTestCrop, g, p)
		if len(got) != len(first) || got[0].Strain.ID != first[0].Strain.ID {
			t.Fatal("strain progress order is unstable")
		}
	}
	if !strings.Contains(first[0].Conditions[0].Condition.Describe(), "Density") {
		t.Error("the condition does not describe its gene")
	}
}
