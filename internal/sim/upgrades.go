package sim

import "atomicfarming/internal/bignum"

// General global upgrades — ones that belong to no particular crop. Crop
// specific unlocks live alongside their crop in internal/sim/crops.

const (
	UnlockIrradiationI   UnlockID = "irradiation_1"
	UnlockIrradiationII  UnlockID = "irradiation_2"
	UnlockFieldExtension UnlockID = "field_extension"
)

// IrradiationStepMul is what each irradiation tier multiplies the self-seeding
// mutation rate by.
//
// The base rate is about one seed in ten thousand, which keeps an early farm
// clean but also puts the bred strains out of reach. These are what make drift
// a strategy: two tiers take a high-Mutability line to roughly one seed in
// seven, so a player who wants to breed can pay for the chaos.
const IrradiationStepMul = "12"

func init() {
	// The first thing worth saving for. Cheap enough to be a goal within the
	// opening few minutes rather than a wall.
	RegisterUnlock(Unlock{
		ID:          UnlockFieldExtension,
		Name:        "Field Extension",
		Description: "Clears another column of ground for planting.",
		Cost:        bignum.MustParse("100"),
		Apply: func(m *GlobalModifiers) {
			m.ExtraColumns++
		},
	})
	RegisterUnlock(Unlock{
		ID:          UnlockIrradiationI,
		Name:        "Seed Irradiator",
		Description: "Mutations become twelve times as likely.",
		Cost:        bignum.MustParse("400"),
		Apply: func(m *GlobalModifiers) {
			m.MutationRateMul = mulModifier(m.MutationRateMul, bignum.MustParse(IrradiationStepMul))
		},
	})
	RegisterUnlock(Unlock{
		ID:          UnlockIrradiationII,
		Name:        "Reactor Bed",
		Description: "Mutations become twelve times likelier again.",
		Cost:        bignum.MustParse("12000"),
		Apply: func(m *GlobalModifiers) {
			m.MutationRateMul = mulModifier(m.MutationRateMul, bignum.MustParse(IrradiationStepMul))
		},
	})
}
