package sim

import (
	"fmt"

	"atomicfarming/internal/bignum"
	"atomicfarming/internal/plant"
)

// ResetRuleID is a rule's stable identifier.
type ResetRuleID string

// ResetRule owns one slice of GameState across a layer reset.
//
// Declaring resets as data rather than writing them inline gives two things
// the ad-hoc approach cannot: the set of state a prestige touches is
// enumerable, and TestResetRulesCoverEveryField can prove no GameState field
// was forgotten. See docs/adr/0007-layer-model-and-reset-registry.md.
type ResetRule struct {
	ID    ResetRuleID
	Layer Layer
	// Fields names the GameState fields this rule owns. It is what the
	// coverage test checks against, so it must match the Go field names
	// exactly.
	Fields []string
	// Reset restores this rule's state for a new run. Implementations consult
	// s for durable unlocks and may retain part or all of the previous run's
	// value — persistency over prestige lives here, inside the rule that owns
	// the state, not scattered through a monolithic reset function.
	Reset func(s *GameState)
}

// resetExemptFields lists GameState fields that no layer reset owns: player
// settings, durable cross-prestige progression, derived caches, and
// persistence bookkeeping. Everything else must be claimed by a rule.
var resetExemptFields = map[string]string{
	"Layer":             "the rung itself; changed by ascension, not by a reset",
	"TickRate":          "player setting, not run state",
	"Unlocks":           "durable progression; surviving prestige is the point",
	"DiscoveredStrains": "a collection log; a collection you lose on prestige is not a collection",
	"Modifiers":         "derived cache, not persisted; rebuilt from Unlocks at the end of every reset",
	"LastSaveUnix":      "persistence bookkeeping, stamped by Save",
}

var (
	resetRules  []ResetRule
	resetRuleID = map[ResetRuleID]bool{}
)

// RegisterResetRule adds a rule to the registry. Duplicate IDs panic: a rule
// ID names a decision about persistence and must be unique.
func RegisterResetRule(r ResetRule) {
	if r.ID == "" {
		panic("sim: reset rule registered with empty ID")
	}
	if r.Reset == nil {
		panic(fmt.Sprintf("sim: reset rule %q has no Reset func", r.ID))
	}
	if !r.Layer.Valid() {
		panic(fmt.Sprintf("sim: reset rule %q names unknown layer %q", r.ID, r.Layer))
	}
	if resetRuleID[r.ID] {
		panic(fmt.Sprintf("sim: duplicate reset rule %q", r.ID))
	}
	resetRuleID[r.ID] = true
	resetRules = append(resetRules, r)
}

// ResetRulesFor returns the rules registered against layer l, in registration
// order.
func ResetRulesFor(l Layer) []ResetRule {
	out := make([]ResetRule, 0, len(resetRules))
	for _, r := range resetRules {
		if r.Layer == l {
			out = append(out, r)
		}
	}
	return out
}

// ApplyLayerReset runs every rule registered for layer l, then rebuilds the
// derived modifier cache. It is the only supported way to start a fresh run.
func ApplyLayerReset(s *GameState, l Layer) {
	if s == nil {
		return
	}
	for _, r := range ResetRulesFor(l) {
		r.Reset(s)
	}
	rebuildModifiers(s)
}

func init() {
	RegisterResetRule(ResetRule{
		ID:     "field_grid",
		Layer:  LayerField,
		Fields: []string{"Grid"},
		Reset: func(s *GameState) {
			// A fresh farm, sized by whatever plot budget durable unlocks
			// have earned. Planted crops never survive a reset.
			w, h := StartingGridSize(s)
			s.Grid = NewGrid(w, h)
		},
	})
	RegisterResetRule(ResetRule{
		ID:     "field_cash",
		Layer:  LayerField,
		Fields: []string{"Cash"},
		Reset:  func(s *GameState) { s.Cash = bignum.Zero() },
	})
	RegisterResetRule(ResetRule{
		ID:     "field_clock",
		Layer:  LayerField,
		Fields: []string{"Ticks"},
		Reset:  func(s *GameState) { s.Ticks = 0 },
	})
	RegisterResetRule(ResetRule{
		ID:     "field_randomness",
		Layer:  LayerField,
		Fields: []string{"WorldSeed", "PlantCounter"},
		Reset: func(s *GameState) {
			// Re-derived rather than kept, so a new run grows different
			// plants, and rather than redrawn from the clock, so the reset
			// itself stays deterministic and replayable.
			s.WorldSeed = plant.Hash64(s.WorldSeed)
			s.PlantCounter = 0
		},
	})
	RegisterResetRule(ResetRule{
		ID:     "field_inventory",
		Layer:  LayerField,
		Fields: []string{"Inventory"},
		Reset: func(s *GameState) {
			// Seeds are run state. A fresh run starts from the same handful
			// everyone begins with, whatever was in the barn before.
			s.Inventory = Inventory{}
			grantStarterSeeds(s)
		},
	})
	RegisterResetRule(ResetRule{
		ID:     "field_run_count",
		Layer:  LayerField,
		Fields: []string{"RunCount"},
		Reset:  func(s *GameState) { s.RunCount++ },
	})
}
