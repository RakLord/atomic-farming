package plant

import "fmt"

// GeneGroup buckets genes for display and for deciding which subsystem reads
// them.
type GeneGroup uint8

const (
	GroupStem GeneGroup = iota
	GroupFoliage
	GroupFlower
	GroupFruit
	GroupNoise
	GroupVigour
	GroupYield
	GroupMeta
	GroupCount
)

var groupNames = [GroupCount]string{
	"Stem", "Foliage", "Flower", "Fruit", "Noise", "Vigour", "Yield", "Meta",
}

func (g GeneGroup) String() string {
	if g < GroupCount {
		return groupNames[g]
	}
	return "Unknown"
}

// GeneSpec describes one gene. It says how the diploid pair collapses to a
// single value and what an unspecified plant carries; it deliberately does
// not say what the value means. Interpretation belongs to whoever reads it —
// the morph package for shape, the sim package for stats.
type GeneSpec struct {
	ID      GeneID
	Name    string
	Group   GeneGroup
	Expr    Expression
	Default Allele
}

// GeneCatalog describes every gene, indexed by GeneID.
//
// Append-only: a gene's index is a save-format identifier. Adding a gene goes
// on the end; retiring one leaves a reserved slot behind. Reordering would
// silently reinterpret every saved genome.
var GeneCatalog = [GeneCount]GeneSpec{
	// Stem
	GeneStemArchetype: {GeneStemArchetype, "Stem Archetype", GroupStem, ExprDominant, 0},
	GeneStemHeight:    {GeneStemHeight, "Stem Height", GroupStem, ExprAverage, 160},
	GeneStemThickness: {GeneStemThickness, "Stem Thickness", GroupStem, ExprAverage, 90},
	GeneStemCurve:     {GeneStemCurve, "Stem Curve", GroupStem, ExprAverage, 128},
	GeneStemTaper:     {GeneStemTaper, "Stem Taper", GroupStem, ExprAverage, 120},
	GeneNodeCount:     {GeneNodeCount, "Node Count", GroupStem, ExprAverage, 110},
	GeneBranchAngle:   {GeneBranchAngle, "Branch Angle", GroupStem, ExprAverage, 128},
	GeneBranchLength:  {GeneBranchLength, "Branch Length", GroupStem, ExprAverage, 100},
	GeneStemHue:       {GeneStemHue, "Stem Hue", GroupStem, ExprAverage, 80},
	GeneStemSat:       {GeneStemSat, "Stem Saturation", GroupStem, ExprAverage, 150},
	GeneStemLum:       {GeneStemLum, "Stem Luminance", GroupStem, ExprAverage, 90},

	// Foliage
	GeneLeafArchetype:   {GeneLeafArchetype, "Leaf Archetype", GroupFoliage, ExprDominant, 0},
	GeneLeafSize:        {GeneLeafSize, "Leaf Size", GroupFoliage, ExprAverage, 130},
	GeneLeafDroop:       {GeneLeafDroop, "Leaf Droop", GroupFoliage, ExprAverage, 120},
	GeneLeafPerNode:     {GeneLeafPerNode, "Leaves Per Node", GroupFoliage, ExprAverage, 128},
	GeneFoliageHueShift: {GeneFoliageHueShift, "Foliage Hue Shift", GroupFoliage, ExprAverage, 128},
	GeneFoliageLumShift: {GeneFoliageLumShift, "Foliage Luminance Shift", GroupFoliage, ExprAverage, 140},

	// Flower
	GeneFlowerArchetype: {GeneFlowerArchetype, "Flower Archetype", GroupFlower, ExprDominant, 128},
	GeneFlowerSize:      {GeneFlowerSize, "Flower Size", GroupFlower, ExprAverage, 120},
	GenePetalCount:      {GenePetalCount, "Petal Count", GroupFlower, ExprAverage, 128},
	GenePetalLength:     {GenePetalLength, "Petal Length", GroupFlower, ExprAverage, 140},
	GenePetalWidth:      {GenePetalWidth, "Petal Width", GroupFlower, ExprAverage, 110},
	GenePetalCurl:       {GenePetalCurl, "Petal Curl", GroupFlower, ExprAverage, 128},
	GeneFlowerHue:       {GeneFlowerHue, "Flower Hue", GroupFlower, ExprAverage, 220},
	GeneFlowerSat:       {GeneFlowerSat, "Flower Saturation", GroupFlower, ExprAverage, 190},
	GeneFlowerLum:       {GeneFlowerLum, "Flower Luminance", GroupFlower, ExprAverage, 160},

	// Fruit
	GeneFruitArchetype: {GeneFruitArchetype, "Fruit Archetype", GroupFruit, ExprDominant, 60},
	GeneFruitSize:      {GeneFruitSize, "Fruit Size", GroupFruit, ExprAverage, 90},
	GeneFruitCount:     {GeneFruitCount, "Fruit Count", GroupFruit, ExprAverage, 100},
	GeneFruitHue:       {GeneFruitHue, "Fruit Hue", GroupFruit, ExprAverage, 250},

	// Noise
	GeneJitter:   {GeneJitter, "Jitter", GroupNoise, ExprAverage, 60},
	GeneSymmetry: {GeneSymmetry, "Symmetry", GroupNoise, ExprAverage, 128},

	// Vigour — declared now, read when Phase 1 grows crops.
	// 106 rather than a neutral 128: Express remaps into the species window,
	// and the Stem's starts at 22, which would push a 128 up to 139 and mature
	// the starter seed in 17.6s instead of the intended 20s.
	GeneGrowthRate:    {GeneGrowthRate, "Growth Rate", GroupVigour, ExprAverage, 106},
	GeneVitality:      {GeneVitality, "Vitality", GroupVigour, ExprDominant, 200},
	GeneLifespan:      {GeneLifespan, "Lifespan", GroupVigour, ExprAverage, 128},
	GeneWaterNeed:     {GeneWaterNeed, "Water Need", GroupVigour, ExprAverage, 128},
	GeneNutrientDrain: {GeneNutrientDrain, "Nutrient Drain", GroupVigour, ExprAverage, 128},

	// Yield
	GeneYieldAmount:   {GeneYieldAmount, "Yield Amount", GroupYield, ExprAverage, 128},
	GeneYieldQuality:  {GeneYieldQuality, "Yield Quality", GroupYield, ExprAverage, 128},
	GeneHarvestChance: {GeneHarvestChance, "Harvest Chance", GroupYield, ExprAverage, 230},
	GeneDensity:       {GeneDensity, "Density", GroupYield, ExprAverage, 40},
	GeneRegrowth:      {GeneRegrowth, "Regrowth", GroupYield, ExprAverage, 0},

	// Meta
	GeneMutability: {GeneMutability, "Mutability", GroupMeta, ExprRecessive, 20},
	GeneFertility:  {GeneFertility, "Fertility", GroupMeta, ExprAverage, 150},
	GeneAffinity:   {GeneAffinity, "Affinity", GroupMeta, ExprAverage, 128},
}

// GenesInGroup returns the genes belonging to g, in catalog order.
func GenesInGroup(g GeneGroup) []GeneID {
	var out []GeneID
	for i := 0; i < GeneCount; i++ {
		if GeneCatalog[i].Group == g {
			out = append(out, GeneID(i))
		}
	}
	return out
}

// init validates the catalog at startup. A gap or a mismatched ID would
// silently misattribute every gene downstream, so fail loudly instead.
func init() {
	for i := 0; i < GeneCount; i++ {
		spec := GeneCatalog[i]
		if spec.Name == "" {
			panic(fmt.Sprintf("plant: gene %d has no catalog entry", i))
		}
		if int(spec.ID) != i {
			panic(fmt.Sprintf("plant: gene %d (%s) has mismatched ID %d", i, spec.Name, spec.ID))
		}
		if spec.Group >= GroupCount {
			panic(fmt.Sprintf("plant: gene %s has unknown group %d", spec.Name, spec.Group))
		}
		if spec.Expr >= exprCount {
			panic(fmt.Sprintf("plant: gene %s has unknown expression %d", spec.Name, spec.Expr))
		}
	}
}
