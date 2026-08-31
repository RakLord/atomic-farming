package sim

import (
	"testing"

	"atomicfarming/internal/bignum"
)

func TestNormalizedPromotesZeroDecimalsToIdentity(t *testing.T) {
	m := GlobalModifiers{}.Normalized()
	one := bignum.One()
	for name, got := range map[string]bignum.Decimal{
		"GrowthRateMul": m.GrowthRateMul,
		"YieldMul":      m.YieldMul,
		"SellPriceMul":  m.SellPriceMul,
		"SeedCostMul":   m.SeedCostMul,
	} {
		if !got.Eq(one) {
			t.Errorf("%s = %s, want 1 — read sites would multiply by zero", name, got)
		}
	}
	if m.ExtraPlots != 0 {
		t.Errorf("ExtraPlots = %d, want 0 — integer fields are additive", m.ExtraPlots)
	}
}

func TestNormalizedLeavesSetValuesAlone(t *testing.T) {
	want := bignum.MustParse("2.5")
	m := GlobalModifiers{YieldMul: want}.Normalized()
	if !m.YieldMul.Eq(want) {
		t.Errorf("YieldMul = %s, want %s", m.YieldMul, want)
	}
}

func TestRebuildModifiersDerivesFromUnlocksAndIsIdempotent(t *testing.T) {
	const id UnlockID = "test_growth"
	UnlockCatalog[id] = Unlock{
		ID:    id,
		Apply: func(m *GlobalModifiers) { m.GrowthRateMul = mulModifier(m.GrowthRateMul, bignum.MustParse("1.5")) },
	}
	t.Cleanup(func() { delete(UnlockCatalog, id) })

	s := NewGameState()
	s.Unlocks[id] = true

	rebuildModifiers(s)
	first := s.Modifiers
	rebuildModifiers(s)

	if !s.Modifiers.GrowthRateMul.Eq(first.GrowthRateMul) {
		t.Errorf("rebuild is not idempotent: %s then %s", first.GrowthRateMul, s.Modifiers.GrowthRateMul)
	}
	if want := bignum.MustParse("1.5"); !s.Modifiers.GrowthRateMul.Eq(want) {
		t.Errorf("GrowthRateMul = %s, want %s", s.Modifiers.GrowthRateMul, want)
	}
}

func TestRebuildModifiersIgnoresUnknownAndUnownedUnlocks(t *testing.T) {
	s := NewGameState()
	s.Unlocks["retired_unlock_from_an_old_save"] = true
	s.Unlocks["also_not_owned"] = false

	rebuildModifiers(s) // must not panic

	if !s.Modifiers.Normalized().GrowthRateMul.Eq(bignum.One()) {
		t.Error("unknown unlock affected the modifier cache")
	}
}
