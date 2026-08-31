package morph

import (
	"math"

	"atomicfarming/internal/plant"
)

// petalPath returns one petal in local space, attached at the origin and
// reaching (1, 0). width is the half-width as a fraction of length, and curl
// runs -1..1 for a petal that sweeps back or forward.
func petalPath(a plant.FlowerArchetype, w, curl float64) []Segment {
	// curl biases the control points along the petal, turning a symmetric
	// shape into one that hooks inward or flares outward.
	bias := curl * 0.22

	switch a {
	case plant.FlowerStar:
		// Straight-sided and sharply pointed.
		return []Segment{
			moveTo(Point{0, 0}),
			lineTo(Point{0.35 + bias, w}),
			lineTo(Point{1, 0}),
			lineTo(Point{0.35 + bias, -w}),
			closePath(),
		}
	case plant.FlowerBell:
		// Broad at the mouth, pinched at the throat.
		return []Segment{
			moveTo(Point{0, 0}),
			cubicTo(Point{0.30 + bias, w * 0.35}, Point{0.72, w * 1.20}, Point{1, w * 0.55}),
			quadTo(Point{1.10, 0}, Point{1, -w * 0.55}),
			cubicTo(Point{0.72, -w * 1.20}, Point{0.30 + bias, -w * 0.35}, Point{0, 0}),
			closePath(),
		}
	case plant.FlowerSpike:
		// A long narrow floret; many of these stacked read as a spike.
		return []Segment{
			moveTo(Point{0, 0}),
			quadTo(Point{0.55 + bias, w * 0.45}, Point{1, 0}),
			quadTo(Point{0.55 + bias, -w * 0.45}, Point{0, 0}),
			closePath(),
		}
	case plant.FlowerTrumpet:
		// Narrow throat opening into a wide flared mouth.
		return []Segment{
			moveTo(Point{0, 0}),
			cubicTo(Point{0.42 + bias, w * 0.18}, Point{0.72, w * 1.35}, Point{1, w * 1.05}),
			quadTo(Point{1.12, 0}, Point{1, -w * 1.05}),
			cubicTo(Point{0.72, -w * 1.35}, Point{0.42 + bias, -w * 0.18}, Point{0, 0}),
			closePath(),
		}
	case plant.FlowerCluster:
		// A small rounded floret, repeated densely by the caller.
		return []Segment{
			moveTo(Point{0, 0}),
			cubicTo(Point{0.20, w * 0.95}, Point{0.80, w * 0.95}, Point{1, 0}),
			cubicTo(Point{0.80, -w * 0.95}, Point{0.20, -w * 0.95}, Point{0, 0}),
			closePath(),
		}
	default: // plant.FlowerDisc
		return []Segment{
			moveTo(Point{0, 0}),
			cubicTo(Point{0.28 + bias, w}, Point{0.74, w * 0.85}, Point{1, 0}),
			cubicTo(Point{0.74, -w * 0.85}, Point{0.28 + bias, -w}, Point{0, 0}),
			closePath(),
		}
	}
}

// flowerLayout describes how an archetype arranges its petals.
type flowerLayout struct {
	petals  int
	arc     float64 // radians the petals span; 2π for a full radial whorl
	rotBase float64 // rotation of the first petal
}

func layoutFor(a plant.FlowerArchetype, petals int) flowerLayout {
	switch a {
	case plant.FlowerBell, plant.FlowerTrumpet:
		// Nodding forward rather than facing the viewer: a narrow fan aimed
		// down and out, so the flower reads as hanging.
		return flowerLayout{petals: maxInt(petals/2, 3), arc: math.Pi * 0.9, rotBase: -math.Pi * 0.95}
	case plant.FlowerSpike:
		// Florets along the stem tip rather than around a centre.
		return flowerLayout{petals: petals, arc: math.Pi * 0.55, rotBase: math.Pi/2 - math.Pi*0.275}
	default:
		return flowerLayout{petals: petals, arc: 2 * math.Pi, rotBase: 0}
	}
}

// addFlower emits one flower head centred at origin, facing outward, scaled to
// size. open runs 0..1 and drives both the head's scale and how far the petals
// have swung away from the bud.
func (b *Blueprint) addFlower(p plant.Phenotype, origin Point, size, open float64, petalCol, centreCol HSL, z int) {
	arch := p.FlowerArchetype()
	if arch == plant.FlowerNone || open <= 0 || size <= 0 {
		return
	}

	petals := p.Scaled(plant.GenePetalCount, 3, 12)
	layout := layoutFor(arch, petals)
	if layout.petals < 1 {
		return
	}

	width := lerp(0.22, 0.62, p.Unit(plant.GenePetalWidth))
	length := lerp(0.55, 1.15, p.Unit(plant.GenePetalLength))
	curl := p.Signed(plant.GenePetalCurl)

	// A bud has its petals folded in; opening swings them out and lengthens
	// them, so the transition from bud to bloom is continuous.
	folded := lerp(0.35, 1, easeOut(open))
	petal := petalPath(arch, width, curl*folded)

	step := layout.arc / float64(layout.petals)
	if layout.arc >= 2*math.Pi-1e-9 {
		step = 2 * math.Pi / float64(layout.petals)
	}

	for i := 0; i < layout.petals; i++ {
		rot := layout.rotBase + step*float64(i)
		// Nudge each petal by a smooth, phenotype-driven amount so a whorl
		// never looks machined.
		rot += wobble(p, i, 3) * p.Unit(plant.GeneJitter) * 0.18
		b.addOutlined(transform(petal, xform{Origin: origin, Scale: size * length * folded, Rot: rot}), petalCol, z)
	}

	// The eye of the flower, drawn last so it sits over the petal roots.
	centre := size * lerp(0.16, 0.34, p.Unit(plant.GeneFlowerSize)) * easeOut(open)
	if centre > 0 {
		b.addOutlined(circlePath(origin, centre), centreCol, z+1)
	}
}

// circlePath approximates a circle with four cubic segments.
func circlePath(c Point, r float64) []Segment {
	const k = 0.5522847498307936 // circle-to-bezier constant
	o := r * k
	return []Segment{
		moveTo(Point{c.X, c.Y + r}),
		cubicTo(Point{c.X + o, c.Y + r}, Point{c.X + r, c.Y + o}, Point{c.X + r, c.Y}),
		cubicTo(Point{c.X + r, c.Y - o}, Point{c.X + o, c.Y - r}, Point{c.X, c.Y - r}),
		cubicTo(Point{c.X - o, c.Y - r}, Point{c.X - r, c.Y - o}, Point{c.X - r, c.Y}),
		cubicTo(Point{c.X - r, c.Y + o}, Point{c.X - o, c.Y + r}, Point{c.X, c.Y + r}),
		closePath(),
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
