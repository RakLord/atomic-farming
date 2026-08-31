package sim

import (
	"testing"

	"atomicfarming/internal/bignum"
)

func TestTickAdvancesTheClock(t *testing.T) {
	s := NewGameState()
	for i := 0; i < 5; i++ {
		s.Tick()
	}
	if s.Ticks != 5 {
		t.Errorf("Ticks = %d, want 5", s.Ticks)
	}
}

func TestTickIsSafeOnNilState(t *testing.T) {
	var s *GameState
	s.Tick() // must not panic
}

func TestBaseGrowContextNormalizesModifiersOnce(t *testing.T) {
	s := NewGameState()
	ctx := s.baseGrowContext()
	if !ctx.Modifiers.GrowthRateMul.Eq(bignum.One()) {
		t.Errorf("GrowContext.Modifiers not normalized: GrowthRateMul = %s", ctx.Modifiers.GrowthRateMul)
	}
	if ctx.Grid == nil {
		t.Fatal("GrowContext.Grid is nil")
	}
	if w, h := ctx.Grid.Size(); w != DefaultGridW || h != DefaultGridH {
		t.Errorf("GrowContext grid is %dx%d, want %dx%d", w, h, DefaultGridW, DefaultGridH)
	}
	if ctx.Layer != LayerField {
		t.Errorf("GrowContext.Layer = %q, want %q", ctx.Layer, LayerField)
	}
}

func TestNewTestGrowContextIsNormalized(t *testing.T) {
	ctx := NewTestGrowContext()
	if !ctx.Modifiers.SellPriceMul.Eq(bignum.One()) {
		t.Errorf("SellPriceMul = %s, want 1", ctx.Modifiers.SellPriceMul)
	}
}
