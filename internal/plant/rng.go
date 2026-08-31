// Package plant holds the plant genome: its representation, how a diploid
// pair collapses into an expressed value, and how genomes breed and mutate.
//
// It is pure Go — no Ebitengine, and no dependency on internal/sim. sim
// depends on plant, not the reverse, so species gene ranges are passed in
// rather than looked up. That keeps the genome testable in isolation.
package plant

// Hash64 is splitmix64.
//
// It is fixed forever. Save-visible outcomes derive from this exact stream,
// so replacing it would silently change every existing plant's rolls. It is
// implemented here rather than taken from math/rand precisely so that no
// toolchain update can shift it. See docs/adr/0010-determinism.md.
func Hash64(x uint64) uint64 {
	x += 0x9e3779b97f4a7c15
	x = (x ^ (x >> 30)) * 0xbf58476d1ce4e5b9
	x = (x ^ (x >> 27)) * 0x94d049bb133111eb
	return x ^ (x >> 31)
}

// Purpose tags a roll so that independent decisions drawn from the same seed
// stay uncorrelated. Two rolls sharing a seed and salt but differing in
// Purpose are independent.
//
// Values are persisted indirectly through the outcomes they produce, so never
// renumber an existing Purpose — append new ones.
type Purpose uint64

const (
	PurposeDeath Purpose = iota + 1
	PurposeHarvest
	PurposeBreed
	PurposeMutation
	PurposeJitter
	PurposeSpawn
	PurposeGenome
)

// Mix folds a seed, a purpose, and a salt into one deterministic value.
func Mix(seed uint64, p Purpose, salt uint64) uint64 {
	return Hash64(Hash64(seed^uint64(p)*0x9e3779b97f4a7c15) ^ salt)
}

// Roll returns a deterministic value in [0, n). n == 0 yields 0.
//
// The modulo introduces a bias of order n/2^64, which for any n this game
// uses is far below one part in 10^15 — immaterial, and worth the simplicity.
func Roll(seed uint64, p Purpose, salt, n uint64) uint64 {
	if n == 0 {
		return 0
	}
	return Mix(seed, p, salt) % n
}

// BasisPoints is the denominator for integer probabilities: 10000 bp = 100%.
const BasisPoints = 10000

// Chance reports whether an event of probability bp basis points occurs.
//
// Gameplay probabilities are integers by rule, never floats: a float-derived
// outcome could differ between the desktop and WASM builds and desynchronise
// an offline tick replay. See docs/adr/0010-determinism.md.
func Chance(seed uint64, p Purpose, salt uint64, bp int) bool {
	if bp <= 0 {
		return false
	}
	if bp >= BasisPoints {
		return true
	}
	return Roll(seed, p, salt, BasisPoints) < uint64(bp)
}

// UnitFloat returns a deterministic float64 in [0, 1).
//
// Visual use only — jitter, wobble, anything that only reaches pixels. Never
// use it for a gameplay outcome; use Chance or Roll.
func UnitFloat(seed uint64, p Purpose, salt uint64) float64 {
	return float64(Mix(seed, p, salt)>>11) / float64(uint64(1)<<53)
}

// SignedUnit returns a deterministic float64 in [-1, 1). Visual use only.
func SignedUnit(seed uint64, p Purpose, salt uint64) float64 {
	return UnitFloat(seed, p, salt)*2 - 1
}
