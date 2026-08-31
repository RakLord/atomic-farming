package morph

import (
	"math"

	"atomicfarming/internal/plant"
)

// wobble returns a smooth pseudo-random value in [-1, 1] that varies by index
// but changes only slightly when the phenotype changes.
//
// This is the whole trick behind the locality guarantee. A conventional hash
// of the genome would give excellent variety and terrible continuity — one
// allele step would reshuffle every offset, and a mutated plant would look
// nothing like its parent. Instead, variety comes from the discrete index
// (which node, which petal) while the genes only shift a low-frequency phase.
// A one-step gene change moves the phase by about 0.02 radians, which nudges
// the result rather than randomising it.
//
// Visual use only. Gameplay randomness goes through plant.Roll.
func wobble(p plant.Phenotype, index, channel int) float64 {
	phase := float64(index)*2.3999632 + float64(channel)*1.1701
	drift := p.Unit(plant.GeneJitter)*6 + p.Unit(plant.GeneSymmetry)*4
	return math.Sin(phase + drift)
}
