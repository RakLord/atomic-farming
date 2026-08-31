package morph

import "atomicfarming/internal/plant"

// leafPath returns a leaf outline in local space: the stalk attaches at the
// origin and the tip sits at (1, 0), with width expressed as a fraction of
// length. Callers scale and rotate it into place.
//
// Every archetype is built from the same anchors so that a change of
// archetype moves the silhouette rather than replacing it wholesale.
func leafPath(a plant.LeafArchetype, w float64) []Segment {
	switch a {
	case plant.LeafLance:
		// Widest near the base, drawn out to a long point.
		return []Segment{
			moveTo(Point{0, 0}),
			cubicTo(Point{0.22, w}, Point{0.68, w * 0.55}, Point{1, 0}),
			cubicTo(Point{0.68, -w * 0.55}, Point{0.22, -w}, Point{0, 0}),
			closePath(),
		}
	case plant.LeafLobed:
		// Three shallow lobes per side.
		return []Segment{
			moveTo(Point{0, 0}),
			cubicTo(Point{0.14, w * 0.95}, Point{0.28, w * 0.40}, Point{0.42, w * 0.80}),
			cubicTo(Point{0.56, w * 1.10}, Point{0.64, w * 0.45}, Point{0.78, w * 0.70}),
			cubicTo(Point{0.90, w * 0.85}, Point{0.96, w * 0.30}, Point{1, 0}),
			cubicTo(Point{0.96, -w * 0.30}, Point{0.90, -w * 0.85}, Point{0.78, -w * 0.70}),
			cubicTo(Point{0.64, -w * 0.45}, Point{0.56, -w * 1.10}, Point{0.42, -w * 0.80}),
			cubicTo(Point{0.28, -w * 0.40}, Point{0.14, -w * 0.95}, Point{0, 0}),
			closePath(),
		}
	case plant.LeafNeedle:
		return []Segment{
			moveTo(Point{0, 0}),
			quadTo(Point{0.5, w * 0.30}, Point{1, 0}),
			quadTo(Point{0.5, -w * 0.30}, Point{0, 0}),
			closePath(),
		}
	case plant.LeafHeart:
		// A notched base: the outline dips back past the attachment point.
		return []Segment{
			moveTo(Point{0.14, 0}),
			cubicTo(Point{0.02, w * 1.25}, Point{0.58, w * 1.15}, Point{1, 0}),
			cubicTo(Point{0.58, -w * 1.15}, Point{0.02, -w * 1.25}, Point{0.14, 0}),
			closePath(),
		}
	case plant.LeafFan:
		// Narrow at the stalk, broad and blunt at the tip.
		return []Segment{
			moveTo(Point{0, 0}),
			cubicTo(Point{0.48, w * 0.30}, Point{0.84, w * 1.05}, Point{0.97, w * 0.85}),
			quadTo(Point{1.06, 0}, Point{0.97, -w * 0.85}),
			cubicTo(Point{0.84, -w * 1.05}, Point{0.48, -w * 0.30}, Point{0, 0}),
			closePath(),
		}
	default: // plant.LeafOval
		return []Segment{
			moveTo(Point{0, 0}),
			quadTo(Point{0.5, w}, Point{1, 0}),
			quadTo(Point{0.5, -w}, Point{0, 0}),
			closePath(),
		}
	}
}

// leafWidthFactor scales the archetype's nominal width so that shapes which
// are naturally slender do not read as broken when the width gene is high.
func leafWidthFactor(a plant.LeafArchetype) float64 {
	switch a {
	case plant.LeafNeedle:
		return 0.35
	case plant.LeafLance:
		return 0.70
	case plant.LeafFan:
		return 1.15
	default:
		return 1
	}
}
