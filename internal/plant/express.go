package plant

// Expression is the rule by which a diploid pair collapses into one expressed
// value. Splitting expression from inheritance is what makes diploid worth
// its cost: a pair of (240, 40) expresses 140 under ExprAverage but can still
// pass on the 240, so extremes survive in the gene pool instead of averaging
// away over generations.
type Expression uint8

const (
	// ExprAverage is co-dominance: the expressed value is the midpoint.
	// Used for numeric and visual traits, where it gives smooth phenotypes.
	ExprAverage Expression = iota
	// ExprDominant expresses the higher allele. Used for categorical
	// archetypes, so a recessive shape can skip generations and reappear.
	ExprDominant
	// ExprRecessive expresses the lower allele, making a high value a hidden
	// trait that only shows when both alleles carry it.
	ExprRecessive

	exprCount
)

var expressionNames = [exprCount]string{"Co-dominant", "Dominant", "Recessive"}

func (e Expression) String() string {
	if e < exprCount {
		return expressionNames[e]
	}
	return "Unknown"
}

// Collapse reduces a diploid pair to a single 0..255 value.
func (e Expression) Collapse(p GenePair) uint8 {
	a, b := int(p.A), int(p.B)
	switch e {
	case ExprDominant:
		if a >= b {
			return uint8(a)
		}
		return uint8(b)
	case ExprRecessive:
		if a <= b {
			return uint8(a)
		}
		return uint8(b)
	default:
		return uint8((a + b) / 2)
	}
}

// GeneRange bounds one gene for a species.
type GeneRange struct{ Min, Max Allele }

// SpeciesRanges bounds every gene for one species. It is what gives a species
// its identity: a tomato's archetype genes have a narrow window, a wildflower's
// a wide one.
type SpeciesRanges [GeneCount]GeneRange

// FullRange is the unconstrained range — every gene free across 0..255. Used
// for wild plants, for the lab, and as the base a species narrows from.
func FullRange() SpeciesRanges {
	var r SpeciesRanges
	for i := range r {
		r[i] = GeneRange{Min: 0, Max: 255}
	}
	return r
}

// Phenotype is the expressed genome: one value per gene, already collapsed
// from its allele pair and already remapped into the species range.
type Phenotype [GeneCount]uint8

// Express collapses every gene and remaps it into the species range.
//
// Remapping rather than clamping is deliberate. Clamping would waste every
// mutation at a boundary — a plant already at its species maximum could never
// vary again. Remapping keeps the full genetic range meaningful for every
// species, and means two species with identical genomes look different, which
// is correct.
func Express(g Genome, r SpeciesRanges) Phenotype {
	var out Phenotype
	for i := 0; i < GeneCount; i++ {
		out[i] = remap(GeneCatalog[i].Expr.Collapse(g[i]), r[i])
	}
	return out
}

// ExpressFull expresses a genome with no species constraint.
func ExpressFull(g Genome) Phenotype { return Express(g, FullRange()) }

// remap linearly maps raw 0..255 onto [r.Min, r.Max] inclusive at both ends.
func remap(raw uint8, r GeneRange) uint8 {
	lo, hi := int(r.Min), int(r.Max)
	if hi < lo {
		lo, hi = hi, lo
	}
	if lo == hi {
		return uint8(lo)
	}
	return uint8(lo + (int(raw)*(hi-lo)+127)/255)
}

// Get returns the raw expressed value for a gene.
func (p Phenotype) Get(id GeneID) uint8 {
	if !id.Valid() {
		return 0
	}
	return p[id]
}

// Unit returns a gene as a float64 in [0, 1]. Visual use.
func (p Phenotype) Unit(id GeneID) float64 { return float64(p.Get(id)) / 255 }

// Signed returns a gene as a float64 in [-1, 1], with 128 reading as roughly
// neutral. Use for genes whose midpoint means "no effect" — curve, hue shift.
func (p Phenotype) Signed(id GeneID) float64 { return (float64(p.Get(id)) - 127.5) / 127.5 }

// Lerp maps a gene onto [lo, hi]. Visual use.
func (p Phenotype) Lerp(id GeneID, lo, hi float64) float64 {
	return lo + (hi-lo)*p.Unit(id)
}

// Scaled maps a gene onto [lo, hi] using integer arithmetic only. This is the
// accessor gameplay code must use — see docs/adr/0010-determinism.md.
func (p Phenotype) Scaled(id GeneID, lo, hi int) int {
	if hi < lo {
		lo, hi = hi, lo
	}
	return lo + (int(p.Get(id))*(hi-lo)+127)/255
}

// Choice maps a gene onto one of n evenly sized buckets, 0..n-1. Used for
// archetype selection.
func (p Phenotype) Choice(id GeneID, n int) int {
	if n <= 1 {
		return 0
	}
	c := int(p.Get(id)) * n / 256
	if c >= n {
		c = n - 1
	}
	return c
}

func (p Phenotype) StemArchetype() StemArchetype {
	return StemArchetype(p.Choice(GeneStemArchetype, int(StemArchetypeCount)))
}

func (p Phenotype) LeafArchetype() LeafArchetype {
	return LeafArchetype(p.Choice(GeneLeafArchetype, int(LeafArchetypeCount)))
}

func (p Phenotype) FlowerArchetype() FlowerArchetype {
	return FlowerArchetype(p.Choice(GeneFlowerArchetype, int(FlowerArchetypeCount)))
}

func (p Phenotype) FruitArchetype() FruitArchetype {
	return FruitArchetype(p.Choice(GeneFruitArchetype, int(FruitArchetypeCount)))
}
