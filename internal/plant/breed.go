package plant

// MaxMutationStep is the largest change a single mutation makes to one
// allele. Keeping it small is what preserves the locality property: a mutated
// offspring must still visibly resemble its parent.
const MaxMutationStep = 3

// Salt offsets keep the several rolls a single allele needs from colliding
// with each other or with a neighbouring allele's rolls.
const (
	saltOccur = 0
	saltStep  = 1 << 20
	saltSign  = 1 << 21
)

// Meiosis produces a child genome from two parents, drawing one allele from
// each parent for every gene.
//
// A child's alleles are always literally its parents' — no value is invented
// or averaged. That is what stops a breeding line regressing to the mean: a
// rare extreme allele passes down intact even while it is not expressed.
func Meiosis(a, b Genome, seed uint64) Genome {
	var child Genome
	for i := 0; i < GeneCount; i++ {
		child[i].A = drawAllele(a[i], seed, uint64(i)*2)
		child[i].B = drawAllele(b[i], seed, uint64(i)*2+1)
	}
	return child
}

func drawAllele(p GenePair, seed, salt uint64) Allele {
	if Roll(seed, PurposeBreed, salt, 2) == 0 {
		return p.A
	}
	return p.B
}

// Mutate returns a copy of g in which each allele independently has a rateBP
// (basis points) chance of shifting by 1..MaxMutationStep in either
// direction, clamped to the allele range.
//
// This is the model for a cross, where shuffling several genes at once is the
// point. A plant seeding itself uses MutateOnce instead — see the note there.
//
// rateBP <= 0 returns the genome unchanged, so an unmutated breed costs
// nothing.
func Mutate(g Genome, seed uint64, rateBP int) Genome {
	if rateBP <= 0 {
		return g
	}
	out := g
	for i := 0; i < GeneCount; i++ {
		out[i].A = mutateAllele(out[i].A, seed, uint64(i)*2, rateBP)
		out[i].B = mutateAllele(out[i].B, seed, uint64(i)*2+1, rateBP)
	}
	return out
}

func mutateAllele(a Allele, seed, salt uint64, rateBP int) Allele {
	if !Chance(seed, PurposeMutation, salt+saltOccur, rateBP) {
		return a
	}
	step := int(Roll(seed, PurposeMutation, salt+saltStep, MaxMutationStep)) + 1
	if Roll(seed, PurposeMutation, salt+saltSign, 2) == 0 {
		step = -step
	}
	v := int(a) + step
	if v < 0 {
		v = 0
	}
	if v > 255 {
		v = 255
	}
	return Allele(v)
}

// MutateOnce gives a genome a single chance of a copy error. When it fires,
// exactly one allele moves exactly one step; otherwise the genome comes back
// untouched.
//
// This is the model for a plant seeding itself, and it is deliberately a
// different shape from Mutate. Rolling every allele independently — however
// low the rate — smears a little change across several genes at once, which
// leaves a barn full of near-identical lines nobody can tell apart or choose
// between. One roll, one gene, one step makes a mutation an event you can
// point at, and leaves every other seed a clean copy.
//
// See docs/adr/0014-self-seeding-is-cloning.md.
func MutateOnce(g Genome, seed uint64, chancePPM int) (Genome, bool) {
	if chancePPM <= 0 {
		return g, false
	}
	if !ChancePPM(seed, PurposeMutation, 0, chancePPM) {
		return g, false
	}

	gene := Roll(seed, PurposeMutation, 1, uint64(GeneCount))
	pair := g[gene]

	target := &pair.A
	if Roll(seed, PurposeMutation, 2, 2) == 1 {
		target = &pair.B
	}

	step := 1
	if Roll(seed, PurposeMutation, 3, 2) == 0 {
		step = -1
	}
	// Reflect at the boundaries rather than clamping. Clamping would let a
	// mutation fire and change nothing, breaking the one-step guarantee this
	// function exists to make.
	v := int(*target) + step
	switch {
	case v < 0:
		v = 1
	case v > 255:
		v = 254
	}
	*target = Allele(v)

	g[gene] = pair
	return g, true
}

// MutationRateBP is the per-allele mutation chance for an offspring of two
// parents, in basis points.
//
// It is the mean of the parents' expressed Mutability scaled into a
// deliberately small window, plus whatever global upgrades contribute. All
// integer arithmetic — a mutation is a gameplay outcome, so it must replay
// identically on every platform.
func MutationRateBP(a, b Phenotype, bonusBP int) int {
	const maxBaseBP = 40 // 0.4% per allele at maximum genetic mutability
	mut := (int(a.Get(GeneMutability)) + int(b.Get(GeneMutability))) / 2
	rate := (mut*maxBaseBP + 127) / 255
	rate += bonusBP
	if rate < 0 {
		rate = 0
	}
	if rate > BasisPoints {
		rate = BasisPoints
	}
	return rate
}

// Breed is the full cross: meiosis followed by mutation at the rate the
// parents and global upgrades imply.
func Breed(a, b Genome, ranges SpeciesRanges, seed uint64, mutationBonusBP int) Genome {
	rate := MutationRateBP(Express(a, ranges), Express(b, ranges), mutationBonusBP)
	return Mutate(Meiosis(a, b, seed), seed, rate)
}
