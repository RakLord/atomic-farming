package plant

// Allele is one copy of one gene.
//
// 256 steps is fine enough that expressed values vary smoothly, and coarse
// enough that a one-step mutation is a genuinely small change — the property
// that makes bred offspring visibly resemble their parents.
type Allele uint8

// GenePair is the diploid pair for a single gene. Which allele came from
// which parent is not tracked; only the pair matters.
type GenePair struct{ A, B Allele }

// Homozygous reports whether both alleles are identical, so the gene breeds
// true.
func (p GenePair) Homozygous() bool { return p.A == p.B }

// GeneID indexes a gene within a Genome.
//
// A gene's position is a save-format identifier. The catalog is append-only:
// new genes go on the end, retired ones become reserved slots. Never reorder
// or delete. See docs/adr/0009-plant-genome.md.
type GeneID int

const (
	// Stem — the plant's frame.
	GeneStemArchetype GeneID = iota
	GeneStemHeight
	GeneStemThickness
	GeneStemCurve
	GeneStemTaper
	GeneNodeCount
	GeneBranchAngle
	GeneBranchLength
	GeneStemHue
	GeneStemSat
	GeneStemLum

	// Foliage. Colour is a shift from the stem's, not its own triple, so the
	// plant always reads as one organism.
	GeneLeafArchetype
	GeneLeafSize
	GeneLeafDroop
	GeneLeafPerNode
	GeneFoliageHueShift
	GeneFoliageLumShift

	// Flower — the visual payoff, so its colour is fully independent.
	GeneFlowerArchetype
	GeneFlowerSize
	GenePetalCount
	GenePetalLength
	GenePetalWidth
	GenePetalCurl
	GeneFlowerHue
	GeneFlowerSat
	GeneFlowerLum

	// Fruit.
	GeneFruitArchetype
	GeneFruitSize
	GeneFruitCount
	GeneFruitHue

	// Noise — keeps plants from looking mechanically perfect.
	GeneJitter
	GeneSymmetry

	// Vigour. Declared now, read when Phase 1 grows crops.
	GeneGrowthRate
	GeneVitality
	GeneLifespan
	GeneWaterNeed
	GeneNutrientDrain

	// Yield. Density is the stat the whole game climbs toward.
	GeneYieldAmount
	GeneYieldQuality
	GeneHarvestChance
	GeneDensity
	GeneRegrowth

	// Meta — genes about genes.
	GeneMutability
	GeneFertility
	GeneAffinity

	numGenes
)

// GeneCount is the number of genes in a genome.
const GeneCount = int(numGenes)

// Genome is the complete diploid gene set for one plant.
//
// A fixed-length array rather than a slice, so a Genome stays a comparable
// value type that copies cheaply and cannot be aliased between plots.
type Genome [GeneCount]GenePair

// Valid reports whether id addresses a real gene.
func (id GeneID) Valid() bool { return id >= 0 && int(id) < GeneCount }

// IsZero reports whether every allele is zero, which is the signal that a
// genome was never stamped. A real genome is vanishingly unlikely to be all
// zeroes, and DefaultGenome never is.
func (g Genome) IsZero() bool {
	return g == Genome{}
}

// DefaultGenome returns a genome homozygous at every gene's catalog default.
// It is what an unspecified plant expresses, and what a save missing genome
// data is repaired to.
func DefaultGenome() Genome {
	var g Genome
	for i := 0; i < GeneCount; i++ {
		d := GeneCatalog[i].Default
		g[i] = GenePair{A: d, B: d}
	}
	return g
}

// RandomGenome returns a genome with every allele drawn deterministically
// from seed. Used for wild plants and for the lab's Randomise control.
func RandomGenome(seed uint64) Genome {
	var g Genome
	for i := 0; i < GeneCount; i++ {
		g[i] = GenePair{
			A: Allele(Roll(seed, PurposeGenome, uint64(i)*2, 256)),
			B: Allele(Roll(seed, PurposeGenome, uint64(i)*2+1, 256)),
		}
	}
	return g
}
