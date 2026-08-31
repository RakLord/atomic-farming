package sim

import (
	"time"

	"atomicfarming/internal/bignum"
	"atomicfarming/internal/plant"
)

// DefaultTickRate is the logical simulation rate in Hz. Logical state only
// advances on ticks — see docs/adr/0008-tick-model.md.
const DefaultTickRate = 10

// GameState is the complete persisted state of a save.
//
// Every field here must be accounted for by the reset registry: either owned
// by a ResetRule for each layer, or listed in resetExemptFields as durable
// state. TestResetRulesCoverEveryField enforces this, so adding a field is a
// prompt to decide what a prestige does to it.
type GameState struct {
	// Layer is the prestige rung the player is currently on.
	Layer Layer `json:"layer"`
	// Grid is the farm.
	Grid *Grid `json:"grid"`
	// Cash is the primary currency, displayed with a leading $.
	Cash bignum.Decimal `json:"cash"`
	// Unlocks is the set of purchased global upgrades — the source of truth
	// that Modifiers is derived from.
	Unlocks map[UnlockID]bool `json:"unlocks,omitempty"`
	// Modifiers is a derived read cache, rebuilt from Unlocks on every
	// purchase and on every load. Deliberately not persisted: a stored copy
	// could only ever be stale or misleading, and Unlocks — the source of
	// truth — is in the save already.
	Modifiers GlobalModifiers `json:"-"`
	// TickRate is the logical tick rate in Hz. A player setting, not run state.
	TickRate int `json:"tick_rate"`
	// Ticks counts logical ticks elapsed in the current run.
	Ticks uint64 `json:"ticks,omitempty"`
	// RunCount is how many runs of the current layer have been completed.
	RunCount int `json:"run_count,omitempty"`
	// LastSaveUnix is wall-clock seconds at the last write, stamped by Save.
	// It is the seam offline progress will read; nothing consumes it yet.
	LastSaveUnix int64 `json:"last_save_unix,omitempty"`
	// WorldSeed roots every random outcome in this run. It is drawn once when
	// the save is created and re-derived on each layer reset, so two runs
	// differ while either one replays identically.
	WorldSeed uint64 `json:"world_seed,omitempty"`
	// PlantCounter makes each planting's seed distinct. Monotonic within a run.
	PlantCounter uint64 `json:"plant_counter,omitempty"`
	// Inventory is the player's seed stock.
	Inventory Inventory `json:"inventory,omitempty"`
	// SeedAutoSelect records, per species, whether clicking its seed row sows
	// the bulk line straight away instead of opening the picker.
	//
	// Absent means on. Needing the picker on every single sowing is the worse
	// default by far, so the picker is opt-in.
	SeedAutoSelect map[CropKind]bool `json:"seed_auto_select,omitempty"`
	// DiscoveredStrains is the collection log: every named strain the player
	// has grown or bought. A strain's name is derived from its genome and
	// never stored; this record of having met it is the only real state the
	// naming system has. See docs/adr/0012-named-strains.md.
	DiscoveredStrains map[StrainID]bool `json:"discovered_strains,omitempty"`
}

// StarterSeeds is what a new farm begins with. Cash starts at zero, so without
// these there would be no way to begin.
const StarterSeeds = 3

// NewGameState returns the state a brand-new save starts from, with its world
// seed drawn from the clock.
//
// That draw makes it the one constructor whose result is not reproducible, so
// any test whose outcome depends on a roll must use NewGameStateWithSeed
// instead — otherwise it is quietly rolling dice on every run.
func NewGameState() *GameState {
	return NewGameStateWithSeed(plant.Hash64(uint64(time.Now().UnixNano())))
}

// NewGameStateWithSeed returns a brand-new save rooted at a chosen world seed,
// so every roll it goes on to make is reproducible.
func NewGameStateWithSeed(worldSeed uint64) *GameState {
	s := &GameState{
		Layer:             LayerField,
		Grid:              NewGrid(DefaultGridW, DefaultGridH),
		Cash:              bignum.Zero(),
		Unlocks:           map[UnlockID]bool{},
		TickRate:          DefaultTickRate,
		WorldSeed:         worldSeed,
		DiscoveredStrains: map[StrainID]bool{},
	}
	grantStarterSeeds(s)
	rebuildModifiers(s)
	return s
}

// AutoSelectSeeds reports whether a species sows its bulk line on a row click
// rather than opening the seed picker.
func (s *GameState) AutoSelectSeeds(kind CropKind) bool {
	if s == nil {
		return true
	}
	on, set := s.SeedAutoSelect[kind]
	return !set || on
}

// ToggleAutoSelectSeeds flips the preference and returns its new value.
func (s *GameState) ToggleAutoSelectSeeds(kind CropKind) bool {
	if s == nil {
		return true
	}
	if s.SeedAutoSelect == nil {
		s.SeedAutoSelect = map[CropKind]bool{}
	}
	next := !s.AutoSelectSeeds(kind)
	s.SeedAutoSelect[kind] = next
	return next
}

// StarterOffer is the shop entry a new farm's starting seeds come from. A crop
// claims it from init(); until one does, a new farm simply starts empty.
var StarterOffer SeedOfferID

func grantStarterSeeds(s *GameState) {
	o, ok := SeedShop[StarterOffer]
	if !ok {
		return
	}
	s.Inventory.Add(o.Kind, o.Genome, StarterSeeds)
}

// StartingGridSize returns the farm dimensions a run is entitled to: the base
// size plus whatever durable unlocks have widened it by, clamped to the
// maximum farm.
func StartingGridSize(s *GameState) (w, h int) {
	w, h = DefaultGridW, DefaultGridH
	if s != nil {
		w += s.Modifiers.ExtraColumns
		h += s.Modifiers.ExtraRows
	}
	if w > MaxGridW {
		w = MaxGridW
	}
	if h > MaxGridH {
		h = MaxGridH
	}
	if w < DefaultGridW {
		w = DefaultGridW
	}
	if h < DefaultGridH {
		h = DefaultGridH
	}
	return w, h
}

// syncFarmSize grows the farm to the size its unlocks entitle it to.
//
// It only ever grows. Grid.Resize preserves plots by coordinate when growing,
// so a farm mid-crop keeps everything standing; shrinking would destroy
// planted crops, so a farm somehow larger than its entitlement is left alone.
func syncFarmSize(s *GameState) {
	if s == nil || s.Grid == nil {
		return
	}
	w, h := StartingGridSize(s)
	if w <= s.Grid.W && h <= s.Grid.H {
		return
	}
	if w < s.Grid.W {
		w = s.Grid.W
	}
	if h < s.Grid.H {
		h = s.Grid.H
	}
	s.Grid.Resize(w, h)
}

// NextPlantSeed returns a fresh seed for a newly planted crop and advances the
// counter. Stamp the result on the Plot: every roll that plant ever makes
// derives from it.
func (s *GameState) NextPlantSeed() uint64 {
	s.PlantCounter++
	return plant.Hash64(s.WorldSeed ^ plant.Hash64(s.PlantCounter))
}

// HardReset wipes state back to a brand-new save, prestige progression and
// all. The caller is responsible for persisting it, or a later Load will
// restore what was wiped.
func (s *GameState) HardReset() {
	if s == nil {
		return
	}
	*s = *NewGameState()
}
