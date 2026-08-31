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
}

// NewGameState returns the state a brand-new save starts from.
func NewGameState() *GameState {
	s := &GameState{
		Layer:    LayerField,
		Grid:     NewGrid(DefaultGridW, DefaultGridH),
		Cash:     bignum.Zero(),
		Unlocks:  map[UnlockID]bool{},
		TickRate: DefaultTickRate,
		// Drawn from the clock so two players do not farm identical plants.
		// This is the only place the wall clock is read; Tick never does, and
		// every roll thereafter derives from this value.
		WorldSeed: plant.Hash64(uint64(time.Now().UnixNano())),
	}
	rebuildModifiers(s)
	return s
}

// StartingGridSize returns the farm dimensions a fresh run begins with: the
// base size plus whole extra rows and columns paid for out of the ExtraPlots
// budget granted by durable unlocks. Growth alternates column then row so the
// farm stays near-square, and stops at MaxGridW/MaxGridH.
func StartingGridSize(s *GameState) (w, h int) {
	w, h = DefaultGridW, DefaultGridH
	budget := 0
	if s != nil {
		budget = s.Modifiers.ExtraPlots
	}
	for {
		switch {
		case w <= h && w < MaxGridW && budget >= h:
			budget -= h
			w++
		case h < MaxGridH && budget >= w:
			budget -= w
			h++
		default:
			return w, h
		}
	}
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
