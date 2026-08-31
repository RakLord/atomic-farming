package ui

import "atomicfarming/internal/plant"

// LabState is the genetics lab's view state: which genome is on the bench,
// how mature it is drawn, and which two genomes are loaded as parents.
//
// The lab exists to make the procedural generator tunable by looking at it
// rather than by reading test output. All of its logic lives here, free of
// Ebitengine, so it can be exercised headless.
type LabState struct {
	Open bool

	// Genome is the specimen on the bench.
	Genome plant.Genome
	// Maturity is where in its life the specimen is drawn, 0..1.
	Maturity float64
	// AutoGrow advances Maturity every tick, looping, so growth can be
	// watched rather than scrubbed.
	AutoGrow bool

	// Selected is the gene the +/- controls act on.
	Selected plant.GeneID

	ParentA, ParentB plant.Genome
	HasParentA       bool
	HasParentB       bool

	// seq makes every roll — randomise, mutate, breed — differ from the last
	// without reaching for a clock.
	seq uint64
}

// NewLabState returns a lab holding a default specimen.
func NewLabState() *LabState {
	return &LabState{Genome: plant.DefaultGenome(), Maturity: 1}
}

// AutoGrowStep is how much maturity advances per tick when auto-growing. At
// the default 10 Hz tick rate a full life takes about eight seconds.
const AutoGrowStep = 1.0 / 80

// Tick advances the auto-grow animation. It is view state only and never
// touches the simulation.
func (l *LabState) Tick() {
	if !l.Open || !l.AutoGrow {
		return
	}
	l.Maturity += AutoGrowStep
	if l.Maturity > 1 {
		l.Maturity = 0
	}
}

func (l *LabState) next() uint64 {
	l.seq++
	return plant.Hash64(l.seq)
}

// Randomise puts a fresh random specimen on the bench.
func (l *LabState) Randomise() { l.Genome = plant.RandomGenome(l.next()) }

// Mutate applies one round of mutation at a rate high enough to see, which is
// far above the rate breeding uses.
func (l *LabState) Mutate() {
	l.Genome = plant.Mutate(l.Genome, l.next(), plant.BasisPoints/8)
}

// Reset restores the catalog-default specimen.
func (l *LabState) Reset() { l.Genome = plant.DefaultGenome() }

// SelectGene moves the gene cursor, wrapping at both ends.
func (l *LabState) SelectGene(delta int) {
	n := plant.GeneCount
	next := (int(l.Selected) + delta) % n
	if next < 0 {
		next += n
	}
	l.Selected = plant.GeneID(next)
}

// AdjustGene shifts the selected gene's allele. secondAllele picks B over A.
func (l *LabState) AdjustGene(delta int, secondAllele bool) {
	if !l.Selected.Valid() {
		return
	}
	pair := l.Genome[l.Selected]
	target := &pair.A
	if secondAllele {
		target = &pair.B
	}
	v := int(*target) + delta
	if v < 0 {
		v = 0
	}
	if v > 255 {
		v = 255
	}
	*target = plant.Allele(v)
	l.Genome[l.Selected] = pair
}

// ScrubMaturity moves the maturity slider, clamped to its ends.
func (l *LabState) ScrubMaturity(delta float64) {
	l.AutoGrow = false
	l.Maturity += delta
	if l.Maturity < 0 {
		l.Maturity = 0
	}
	if l.Maturity > 1 {
		l.Maturity = 1
	}
}

// StoreParentA and StoreParentB load the bench specimen into a parent slot.
func (l *LabState) StoreParentA() { l.ParentA, l.HasParentA = l.Genome, true }
func (l *LabState) StoreParentB() { l.ParentB, l.HasParentB = l.Genome, true }

// CanBreed reports whether both parent slots are filled.
func (l *LabState) CanBreed() bool { return l.HasParentA && l.HasParentB }

// Breed crosses the two stored parents and puts the offspring on the bench.
// It reports false when a parent slot is empty.
func (l *LabState) Breed() bool {
	if !l.CanBreed() {
		return false
	}
	l.Genome = plant.Breed(l.ParentA, l.ParentB, plant.FullRange(), l.next(), 0)
	return true
}

// MutationPreviewCount is how many one-step neighbours the lab renders beside
// the specimen. Seeing them together is how the locality property is checked
// by eye, alongside the assertion in the morph tests.
const MutationPreviewCount = 9

// MutationPreviews returns MutationPreviewCount neighbours of the bench
// specimen, each one mutation away. They are derived from a fixed seed so the
// grid holds still while a gene is being adjusted.
func (l *LabState) MutationPreviews() []plant.Genome {
	out := make([]plant.Genome, MutationPreviewCount)
	for i := range out {
		out[i] = plant.Mutate(l.Genome, plant.Hash64(uint64(i)+1), plant.BasisPoints/12)
	}
	return out
}
