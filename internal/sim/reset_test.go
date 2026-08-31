package sim

import (
	"reflect"
	"testing"

	"atomicfarming/internal/bignum"
)

// TestResetRulesCoverEveryField is the safety net that makes the reset
// registry worth having: every persisted GameState field must be either
// claimed by exactly one rule per layer, or explicitly declared durable.
// Adding a field without deciding what a prestige does to it fails here.
func TestResetRulesCoverEveryField(t *testing.T) {
	typ := reflect.TypeOf(GameState{})
	for _, layer := range LayerOrder {
		owner := map[string]ResetRuleID{}
		for _, rule := range ResetRulesFor(layer) {
			for _, field := range rule.Fields {
				if _, ok := typ.FieldByName(field); !ok {
					t.Errorf("layer %q: rule %q claims GameState.%s, which does not exist", layer, rule.ID, field)
					continue
				}
				if prev, dup := owner[field]; dup {
					t.Errorf("layer %q: GameState.%s claimed by both %q and %q", layer, field, prev, rule.ID)
					continue
				}
				owner[field] = rule.ID
			}
		}
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			if !f.IsExported() {
				continue
			}
			_, exempt := resetExemptFields[f.Name]
			ruleID, owned := owner[f.Name]
			switch {
			case exempt && owned:
				t.Errorf("layer %q: GameState.%s is listed durable but rule %q also resets it", layer, f.Name, ruleID)
			case !exempt && !owned:
				t.Errorf("layer %q: GameState.%s has no reset rule and is not in resetExemptFields — "+
					"decide what a prestige does to it, then add a rule or a durability note", layer, f.Name)
			}
		}
	}
}

func TestResetExemptFieldsAreRealFields(t *testing.T) {
	typ := reflect.TypeOf(GameState{})
	for name := range resetExemptFields {
		if _, ok := typ.FieldByName(name); !ok {
			t.Errorf("resetExemptFields names GameState.%s, which does not exist", name)
		}
	}
}

func TestApplyLayerResetClearsRunStateAndCountsTheRun(t *testing.T) {
	s := NewGameState()
	s.Cash = bignum.MustParse("1.5e6")
	s.Ticks = 4321
	p, _ := s.Grid.At(Position{X: 1, Y: 1})
	p.Crop = &testCrop{}
	p.Growth = Growth{Stage: 2, Ready: true}

	ApplyLayerReset(s, LayerField)

	if !s.Cash.IsZero() {
		t.Errorf("Cash = %s, want zero", s.Cash)
	}
	if s.Ticks != 0 {
		t.Errorf("Ticks = %d, want 0", s.Ticks)
	}
	if s.RunCount != 1 {
		t.Errorf("RunCount = %d, want 1", s.RunCount)
	}
	if got, _ := s.Grid.At(Position{X: 1, Y: 1}); !got.IsEmpty() {
		t.Error("planted crop survived the reset")
	}
}

// TestApplyLayerResetRetainsDurableProgression covers the persistency-over-
// prestige requirement: an unlock survives the reset and keeps paying out
// through the derived modifier cache, growing the farm the next run starts on.
func TestApplyLayerResetRetainsDurableProgression(t *testing.T) {
	const id UnlockID = "test_extra_plots"
	UnlockCatalog[id] = Unlock{
		ID:   id,
		Name: "Test Homestead",
		Apply: func(m *GlobalModifiers) {
			m.ExtraPlots += 3
			m.YieldMul = mulModifier(m.YieldMul, bignum.MustParse("2"))
		},
	}
	t.Cleanup(func() { delete(UnlockCatalog, id) })

	s := NewGameState()
	s.Unlocks[id] = true
	rebuildModifiers(s)

	ApplyLayerReset(s, LayerField)

	if !s.Unlocks[id] {
		t.Fatal("unlock did not survive the reset")
	}
	if s.Modifiers.ExtraPlots != 3 {
		t.Errorf("ExtraPlots = %d, want 3 — modifiers were not rebuilt after reset", s.Modifiers.ExtraPlots)
	}
	if s.Grid.W != 4 || s.Grid.H != 3 {
		t.Errorf("post-reset farm is %dx%d, want 4x3 — the retained plot budget was not spent", s.Grid.W, s.Grid.H)
	}
	if want := bignum.MustParse("2"); !s.Modifiers.YieldMul.Eq(want) {
		t.Errorf("YieldMul = %s, want %s", s.Modifiers.YieldMul, want)
	}
}

func TestHardResetWipesDurableProgressionToo(t *testing.T) {
	s := NewGameState()
	s.Unlocks["anything"] = true
	s.RunCount = 9

	s.HardReset()

	if len(s.Unlocks) != 0 {
		t.Errorf("Unlocks = %v, want empty", s.Unlocks)
	}
	if s.RunCount != 0 {
		t.Errorf("RunCount = %d, want 0", s.RunCount)
	}
}
