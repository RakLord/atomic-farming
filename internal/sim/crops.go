package sim

import (
	"fmt"

	"atomicfarming/internal/bignum"
	"atomicfarming/internal/plant"
)

// CropKind is a crop's stable on-disk identifier. It is a save-format
// constant: never rename one that has shipped.
type CropKind string

// Crop is the universal behaviour every plantable thing implements. Concrete
// crops live in the internal/sim/crops subpackage and self-register from
// init(). Adding a crop means dropping in one file — no edits to tick.go or
// save.go. See docs/adr/0005-crop-registry-and-plot-model.md.
//
// A Crop value is immutable configuration. All per-plot mutable state lives
// in the Growth passed to Grow and returned from it.
type Crop interface {
	Kind() CropKind
	// Stages is how many growth stages the crop passes through before it is
	// ready to harvest. Must be at least 1.
	Stages() int
	// Ranges bounds every gene for this species. It is what gives a species
	// its identity: a Stem's archetype genes have a narrow window, an exotic's
	// a wide one. See docs/adr/0009-plant-genome.md.
	Ranges() plant.SpeciesRanges
	// Grow advances the crop by one tick and returns the updated growth state.
	//
	// ctx is a read-only view of world state. Implementations must not mutate
	// anything reachable through it, nor retain it across ticks.
	Grow(ctx GrowContext, g Growth) Growth
}

// Yield is what a harvest produces, before global modifiers are applied by
// the caller.
type Yield struct {
	Kind   CropKind       `json:"kind"`
	Amount int            `json:"amount"`
	Value  bignum.Decimal `json:"value"`
}

// Harvestable is the optional capability for crops that can be harvested.
// The tick loop and the player's harvest action both type-assert for it.
//
// Regrowth is expressed through the return values rather than a separate
// interface: cleared reports whether the plot goes empty (an annual), and
// when it does not, next is the state the crop regrows from (a perennial).
type Harvestable interface {
	Crop
	Harvest(ctx GrowContext, g Growth) (y Yield, next Growth, cleared bool)
}

// GrowContext is the read-only view of world state handed to crops during a
// tick. Modifiers is guaranteed Normalized when the tick loop builds the
// context, so read sites can multiply without zero-guards.
type GrowContext struct {
	Grid      GridView
	Pos       Position
	Tick      uint64
	Modifiers GlobalModifiers
	Layer     Layer
	// Phenotype is this plant's expressed genes, already remapped into its
	// species ranges. It comes from the Plot's derived cache rather than being
	// re-expressed each tick.
	Phenotype plant.Phenotype
	// Seed is this plant's roll seed. Every random outcome for this plant
	// derives from it, so growth and death replay identically offline.
	// See docs/adr/0010-determinism.md.
	Seed uint64
}

// GridView is the read-only accessor for farm state. Plots are returned by
// value so crops cannot mutate the live grid through the view.
type GridView interface {
	PlotAt(p Position) (Plot, bool)
	InBounds(p Position) bool
	Size() (w, h int)
}

// NewTestGrowContext returns a GrowContext with Modifiers already Normalized,
// so crop tests do not have to remember that precondition. Grid is nil —
// tests that exercise neighbour reads must populate it themselves.
func NewTestGrowContext() GrowContext {
	return GrowContext{
		Modifiers: GlobalModifiers{}.Normalized(),
		Layer:     LayerField,
		Phenotype: plant.ExpressFull(plant.DefaultGenome()),
	}
}

// CropFactory produces a zero-valued instance of a Crop. The save layer uses
// it to reconstruct concrete types from a persisted kind string.
type CropFactory func() Crop

var cropRegistry = map[CropKind]CropFactory{}

// RegisterCrop wires a CropKind to a factory returning an empty instance.
// Concrete crops call this from init(). Duplicate registrations panic: a
// CropKind is a save-format identifier and must be unique.
func RegisterCrop(kind CropKind, f CropFactory) {
	if kind == "" {
		panic("sim: crop registered with empty kind")
	}
	if f == nil {
		panic(fmt.Sprintf("sim: nil factory registered for crop %q", kind))
	}
	if _, exists := cropRegistry[kind]; exists {
		panic(fmt.Sprintf("sim: duplicate crop registration for %q", kind))
	}
	cropRegistry[kind] = f
}

// RegisteredCropKinds returns the set of registered kinds. Intended for tests
// and diagnostics; production code should not iterate it.
func RegisteredCropKinds() []CropKind {
	out := make([]CropKind, 0, len(cropRegistry))
	for k := range cropRegistry {
		out = append(out, k)
	}
	return out
}

func newCropByKind(kind CropKind) (Crop, error) {
	f, ok := cropRegistry[kind]
	if !ok {
		return nil, fmt.Errorf("sim: unknown crop kind %q", kind)
	}
	return f(), nil
}
