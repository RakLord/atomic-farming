package sim

import "atomicfarming/internal/bignum"

// GlobalModifiers aggregates every active global effect into one struct that
// hot paths read directly. Decimal fields are multiplicative — their zero
// value means "no bonus" once Normalized. Integer fields are additive.
//
// This struct is never mutated from gameplay code; rebuildModifiers is the
// only writer, and it derives the whole struct from GameState.Unlocks.
// Adding a new global effect is one field here plus one catalog entry in
// unlocks.go. See docs/adr/0006-global-modifier-pipeline.md.
//
// The struct is not persisted (GameState.Modifiers is tagged json:"-"), so
// these tags document intent only. Note that omitempty cannot work on a
// bignum.Decimal: encoding/json never treats a struct as empty.
type GlobalModifiers struct {
	// GrowthRateMul scales growth progress accumulated per tick.
	GrowthRateMul bignum.Decimal `json:"growth_rate_mul"`
	// YieldMul scales the amount a harvest produces.
	YieldMul bignum.Decimal `json:"yield_mul"`
	// SellPriceMul scales the cash received per unit sold.
	SellPriceMul bignum.Decimal `json:"sell_price_mul"`
	// SeedCostMul scales seed purchase costs.
	SeedCostMul bignum.Decimal `json:"seed_cost_mul"`
	// MutationRateMul scales how often a self-seeded seed carries a copy
	// error. The base rate is deliberately near-never, so this is what makes
	// deliberate drift possible at all.
	MutationRateMul bignum.Decimal `json:"mutation_rate_mul"`
	// ExtraColumns and ExtraRows widen and deepen the farm beyond its base
	// size. They replaced a plot budget that could not express "one more
	// column": on a 3x3 farm the first column costs three plots and the second
	// costs four, so a fixed grant silently wasted the remainder.
	ExtraColumns int `json:"extra_columns,omitempty"`
	ExtraRows    int `json:"extra_rows,omitempty"`
}

// Normalized returns a copy with zero-valued Decimal fields promoted to 1, so
// downstream multiplication is safe without guards. Integer fields are left
// alone — zero is already the additive identity.
//
// The tick loop calls this once per tick when building GrowContext, not once
// per crop.
func (m GlobalModifiers) Normalized() GlobalModifiers {
	if m.GrowthRateMul.IsZero() {
		m.GrowthRateMul = bignum.One()
	}
	if m.YieldMul.IsZero() {
		m.YieldMul = bignum.One()
	}
	if m.SellPriceMul.IsZero() {
		m.SellPriceMul = bignum.One()
	}
	if m.SeedCostMul.IsZero() {
		m.SeedCostMul = bignum.One()
	}
	if m.MutationRateMul.IsZero() {
		m.MutationRateMul = bignum.One()
	}
	return m
}

// rebuildModifiers recomputes s.Modifiers from s.Unlocks, the source of
// truth. It must run after every unlock purchase and after every save-load;
// that is what makes retuning an unlock's effect apply retroactively to
// existing saves with no migration.
//
// Unknown unlock IDs are ignored so a retired unlock cannot break old saves.
func rebuildModifiers(s *GameState) {
	if s == nil {
		return
	}
	s.Modifiers = GlobalModifiers{}
	for id, owned := range s.Unlocks {
		if !owned {
			continue
		}
		unlock, ok := UnlockCatalog[id]
		if !ok || unlock.Apply == nil {
			continue
		}
		unlock.Apply(&s.Modifiers)
	}
	// Farm size is derived from unlocks too, so it is rebuilt here rather than
	// at each of the several call sites that would otherwise have to remember.
	syncFarmSize(s)
}

// mulModifier folds factor into a multiplicative modifier field, treating an
// untouched (zero) field as the identity. Unlock Apply funcs use it so they
// compose correctly regardless of evaluation order.
func mulModifier(current, factor bignum.Decimal) bignum.Decimal {
	if current.IsZero() {
		current = bignum.One()
	}
	return current.Mul(factor)
}
