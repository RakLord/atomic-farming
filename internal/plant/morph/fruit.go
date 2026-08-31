package morph

import "atomicfarming/internal/plant"

// fruitPath returns one fruit in local space, hanging from the origin with its
// body below and its long axis along +x. Callers rotate it into place.
func fruitPath(a plant.FruitArchetype, size float64) []Segment {
	switch a {
	case plant.FruitPod:
		// A long curved pod.
		return []Segment{
			moveTo(Point{0, 0}),
			cubicTo(Point{size * 0.5, size * 0.42}, Point{size * 1.5, size * 0.34}, Point{size * 2.1, 0}),
			cubicTo(Point{size * 1.5, -size * 0.16}, Point{size * 0.5, -size * 0.20}, Point{0, 0}),
			closePath(),
		}
	case plant.FruitGrainHead:
		// A tapering ear, wider at the base than the tip.
		return []Segment{
			moveTo(Point{0, 0}),
			cubicTo(Point{size * 0.4, size * 0.62}, Point{size * 1.3, size * 0.44}, Point{size * 1.9, 0}),
			cubicTo(Point{size * 1.3, -size * 0.44}, Point{size * 0.4, -size * 0.62}, Point{0, 0}),
			closePath(),
		}
	case plant.FruitTuber:
		// Squat and lumpy.
		return []Segment{
			moveTo(Point{0, 0}),
			cubicTo(Point{size * 0.28, size * 0.95}, Point{size * 1.25, size * 0.80}, Point{size * 1.45, 0}),
			cubicTo(Point{size * 1.25, -size * 0.88}, Point{size * 0.28, -size * 0.95}, Point{0, 0}),
			closePath(),
		}
	case plant.FruitCapsule:
		// An angular seed case.
		return []Segment{
			moveTo(Point{0, 0}),
			lineTo(Point{size * 0.45, size * 0.62}),
			lineTo(Point{size * 1.30, size * 0.30}),
			lineTo(Point{size * 1.30, -size * 0.30}),
			lineTo(Point{size * 0.45, -size * 0.62}),
			closePath(),
		}
	default: // plant.FruitBerry
		return circlePath(Point{X: size * 0.85}, size*0.85)
	}
}

// fruitsPerCluster is how many bodies one fruiting site carries.
func fruitsPerCluster(a plant.FruitArchetype) int {
	switch a {
	case plant.FruitBerry:
		return 3
	case plant.FruitGrainHead, plant.FruitTuber:
		return 1
	default:
		return 2
	}
}
