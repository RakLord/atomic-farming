package morph

import (
	"math"

	"atomicfarming/internal/plant"
)

// Colour is derived from genes here rather than in the renderer, so the whole
// look of a plant is decided in one pure, testable place.
//
// Ranges are deliberately narrower than the full HSL cube: fully saturated or
// near-black plants look like bugs, not variation.

func stemColor(p plant.Phenotype) HSL {
	return HSL{
		H: p.Unit(plant.GeneStemHue) * 360,
		S: lerp(0.18, 0.85, p.Unit(plant.GeneStemSat)),
		L: lerp(0.14, 0.62, p.Unit(plant.GeneStemLum)),
	}
}

// foliageColor is derived from the stem's rather than being its own gene
// triple. One fewer gene, and it guarantees the plant reads as a single
// organism instead of a stem with unrelated leaves attached.
func foliageColor(p plant.Phenotype) HSL {
	c := stemColor(p)
	c.H = wrapHue(c.H + p.Signed(plant.GeneFoliageHueShift)*45)
	c.L = clamp01(c.L + p.Signed(plant.GeneFoliageLumShift)*0.26)
	return c
}

func flowerColor(p plant.Phenotype) HSL {
	return HSL{
		H: p.Unit(plant.GeneFlowerHue) * 360,
		S: lerp(0.25, 0.95, p.Unit(plant.GeneFlowerSat)),
		L: lerp(0.30, 0.82, p.Unit(plant.GeneFlowerLum)),
	}
}

// flowerCentreColor is a contrasting eye: the petal hue rotated away and
// darkened or lightened to whichever reads against the petals.
func flowerCentreColor(p plant.Phenotype) HSL {
	c := flowerColor(p)
	c.H = wrapHue(c.H + 40)
	c.S = clamp01(c.S * 0.85)
	if c.L > 0.5 {
		c.L = clamp01(c.L - 0.30)
	} else {
		c.L = clamp01(c.L + 0.32)
	}
	return c
}

func fruitColor(p plant.Phenotype) HSL {
	return HSL{
		H: p.Unit(plant.GeneFruitHue) * 360,
		S: lerp(0.35, 0.95, p.Unit(plant.GeneFlowerSat)),
		L: lerp(0.26, 0.66, p.Unit(plant.GeneFruitSize)),
	}
}

// shade darkens a colour, used to separate overlapping foliage layers.
func shade(c HSL, by float64) HSL {
	c.L = clamp01(c.L - by)
	return c
}

func wrapHue(h float64) float64 {
	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}
	return h
}
