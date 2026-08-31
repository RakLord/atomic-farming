package morph

import "math"

// spine is the cubic bezier a stem or branch follows.
type spine struct{ P0, C1, C2, P3 Point }

// at evaluates the spine at u in [0,1].
func (s spine) at(u float64) Point {
	v := 1 - u
	a, b, c, d := v*v*v, 3*v*v*u, 3*v*u*u, u*u*u
	return Point{
		X: a*s.P0.X + b*s.C1.X + c*s.C2.X + d*s.P3.X,
		Y: a*s.P0.Y + b*s.C1.Y + c*s.C2.Y + d*s.P3.Y,
	}
}

// tangent is the unnormalised derivative at u.
func (s spine) tangent(u float64) Point {
	v := 1 - u
	a, b, c := 3*v*v, 6*v*u, 3*u*u
	return Point{
		X: a*(s.C1.X-s.P0.X) + b*(s.C2.X-s.C1.X) + c*(s.P3.X-s.C2.X),
		Y: a*(s.C1.Y-s.P0.Y) + b*(s.C2.Y-s.C1.Y) + c*(s.P3.Y-s.C2.Y),
	}
}

// scaledAboutBase scales the spine uniformly about its base, keeping x = 0.5
// fixed. Uniform scaling preserves angles, which is what lets a plant grow
// without its leaves swinging around — see buildStem.
func (s spine) scaledAboutBase(k float64) spine {
	f := func(p Point) Point { return Point{X: 0.5 + (p.X-0.5)*k, Y: p.Y * k} }
	return spine{P0: f(s.P0), C1: f(s.C1), C2: f(s.C2), P3: f(s.P3)}
}

// angleAt is the spine's heading at u, in radians.
func (s spine) angleAt(u float64) float64 {
	t := s.tangent(u)
	if t.X == 0 && t.Y == 0 {
		return math.Pi / 2
	}
	return math.Atan2(t.Y, t.X)
}

// ribbonSamples is how finely a tapered stem is sampled. High enough that the
// silhouette reads as smooth at any plot size we draw.
const ribbonSamples = 18

// ribbon builds a closed, filled outline that follows a spine with a width
// that can vary along its length. A tapered stem cannot be a stroke, because
// a stroke has one width for its whole run.
func ribbon(s spine, widthAt func(u float64) float64) []Segment {
	left := make([]Point, 0, ribbonSamples+1)
	right := make([]Point, 0, ribbonSamples+1)

	for i := 0; i <= ribbonSamples; i++ {
		u := float64(i) / ribbonSamples
		p := s.at(u)
		ang := s.angleAt(u)
		half := widthAt(u) / 2
		nx, ny := math.Cos(ang+math.Pi/2), math.Sin(ang+math.Pi/2)
		left = append(left, Point{X: p.X + nx*half, Y: p.Y + ny*half})
		right = append(right, Point{X: p.X - nx*half, Y: p.Y - ny*half})
	}

	path := make([]Segment, 0, len(left)+len(right)+2)
	path = append(path, moveTo(left[0]))
	for _, p := range left[1:] {
		path = append(path, lineTo(p))
	}
	for i := len(right) - 1; i >= 0; i-- {
		path = append(path, lineTo(right[i]))
	}
	path = append(path, closePath())
	return path
}

// xform is a scale-rotate-translate applied to a local-space path.
type xform struct {
	Origin Point
	Scale  float64
	Rot    float64 // radians
}

func (x xform) apply(p Point) Point {
	sx, sy := p.X*x.Scale, p.Y*x.Scale
	c, s := math.Cos(x.Rot), math.Sin(x.Rot)
	return Point{
		X: x.Origin.X + sx*c - sy*s,
		Y: x.Origin.Y + sx*s + sy*c,
	}
}

// transform maps a local-space path into plant space. The input is not
// modified, so archetype paths can be reused across nodes.
func transform(path []Segment, x xform) []Segment {
	out := make([]Segment, len(path))
	for i, seg := range path {
		out[i] = seg
		for j := 0; j < seg.pointCount(); j++ {
			out[i].P[j] = x.apply(seg.P[j])
		}
	}
	return out
}

// minLeafSin is how far below horizontal a leaf or branch may point, as the
// sine of its world-space angle. Foliage is clamped to it so that a strongly
// curved stem — a vine, especially — cannot swing leaves down through the
// soil line and out of the sprite.
const minLeafSin = -0.26

// clampDroop pulls an angle up until it points no further below horizontal
// than minLeafSin, preserving which side of the plant it points to.
func clampDroop(rot float64) float64 {
	if math.Sin(rot) >= minLeafSin {
		return rot
	}
	lift := math.Asin(minLeafSin) // negative, just below horizontal
	if math.Cos(rot) >= 0 {
		return lift
	}
	return math.Pi - lift
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

// easeOut is the growth curve: quick early progress that settles toward the
// final size, so a plant looks like it is filling out rather than scaling
// linearly.
func easeOut(t float64) float64 {
	t = clamp01(t)
	return 1 - (1-t)*(1-t)
}

// ramp maps t onto 0..1 across the window [lo, hi], flat outside it. It is how
// every growth stage is gated: a leaf, flower, or fruit fades in over its own
// window of the plant's maturity.
func ramp(t, lo, hi float64) float64 {
	if hi <= lo {
		if t >= hi {
			return 1
		}
		return 0
	}
	return clamp01((t - lo) / (hi - lo))
}

// fitScale is the factor a blueprint must be scaled by, about the base of its
// stem, to sit inside the canvas. It returns 1 when the plant already fits.
//
// Sizing the canvas to the largest plant the genome space can produce was
// measured at nearly three times the average plant, which left the typical
// plot looking half empty. Compressing the rare giant instead costs it a
// little size and keeps every ordinary plant filling its plot.
func fitScale(b Blueprint) float64 {
	lo, hi, ok := b.Bounds()
	if !ok {
		return 1
	}

	k := 1.0
	limit := func(extent, bound float64) {
		if extent == 0 {
			return
		}
		if r := bound / extent; r < k {
			k = r
		}
	}
	// Horizontal extents are measured from the stem base at x = 0.5.
	if hi.X > 0.5 {
		limit(hi.X-0.5, CanvasMaxX-0.5)
	}
	if lo.X < 0.5 {
		limit(lo.X-0.5, CanvasMinX-0.5)
	}
	if hi.Y > 0 {
		limit(hi.Y, CanvasMaxY)
	}
	if lo.Y < 0 {
		limit(lo.Y, CanvasMinY)
	}
	return k
}

// scaleBlueprint scales every shape about the base of the stem. Scaling is
// uniform, so a compressed plant keeps its silhouette and stays standing on
// the soil line.
func scaleBlueprint(b *Blueprint, k float64) {
	if k == 1 {
		return
	}
	for i := range b.Shapes {
		shape := &b.Shapes[i]
		shape.Width *= k
		for j := range shape.Path {
			seg := &shape.Path[j]
			for n := 0; n < seg.pointCount(); n++ {
				seg.P[n] = Point{X: 0.5 + (seg.P[n].X-0.5)*k, Y: seg.P[n].Y * k}
			}
		}
	}
}
