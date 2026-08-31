package plant

// Archetypes are the categorical genome choices: which broad shape a plant's
// stem, leaves, flowers, and fruit take. They live here rather than in the
// morph package because they are genome semantics — morph decides how to draw
// one, not which ones exist.
//
// Archetype ordering is a save-format detail in the same way gene position
// is: an expressed archetype is derived from an allele value, so reordering
// would change what existing genomes grow into. Append only.

type StemArchetype uint8

const (
	StemUpright StemArchetype = iota
	StemBranching
	StemVining
	StemRosette
	StemBulb
	StemArchetypeCount
)

var stemArchetypeNames = [StemArchetypeCount]string{
	"Upright", "Branching", "Vining", "Rosette", "Bulb",
}

func (a StemArchetype) String() string {
	if a < StemArchetypeCount {
		return stemArchetypeNames[a]
	}
	return "Unknown"
}

type LeafArchetype uint8

const (
	LeafOval LeafArchetype = iota
	LeafLance
	LeafLobed
	LeafNeedle
	LeafHeart
	LeafFan
	LeafArchetypeCount
)

var leafArchetypeNames = [LeafArchetypeCount]string{
	"Oval", "Lance", "Lobed", "Needle", "Heart", "Fan",
}

func (a LeafArchetype) String() string {
	if a < LeafArchetypeCount {
		return leafArchetypeNames[a]
	}
	return "Unknown"
}

type FlowerArchetype uint8

const (
	FlowerNone FlowerArchetype = iota
	FlowerBell
	FlowerStar
	FlowerDisc
	FlowerCluster
	FlowerSpike
	FlowerTrumpet
	FlowerArchetypeCount
)

var flowerArchetypeNames = [FlowerArchetypeCount]string{
	"None", "Bell", "Star", "Disc", "Cluster", "Spike", "Trumpet",
}

func (a FlowerArchetype) String() string {
	if a < FlowerArchetypeCount {
		return flowerArchetypeNames[a]
	}
	return "Unknown"
}

type FruitArchetype uint8

const (
	FruitNone FruitArchetype = iota
	FruitBerry
	FruitPod
	FruitGrainHead
	FruitTuber
	FruitCapsule
	FruitArchetypeCount
)

var fruitArchetypeNames = [FruitArchetypeCount]string{
	"None", "Berry", "Pod", "Grain Head", "Tuber", "Capsule",
}

func (a FruitArchetype) String() string {
	if a < FruitArchetypeCount {
		return fruitArchetypeNames[a]
	}
	return "Unknown"
}
