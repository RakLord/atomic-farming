// Package morph turns an expressed plant phenotype into geometry.
//
// It is pure Go with no Ebitengine dependency, deliberately: the geometry is
// the hard, tunable part of the generator, and keeping it renderer-free means
// it can be unit-tested and inspected without a GPU. The render package turns
// a Blueprint into pixels and knows nothing about genes.
//
// Coordinate space is normalised: x and y both run 0..1, x = 0.5 is the plant
// centre, y = 0 is the soil line and y = 1 is the top of the plot. Nothing
// here knows the pixel size, which is what makes the output resolution
// independent.
package morph

// The canvas is the region of normalised space a blueprint may occupy.
//
// Plants spread wider than their stem and their lowest leaves rest on or just
// under the soil, so the drawable box is wider than the unit square. Height is
// capped at exactly one unit: the renderer scales by CanvasMaxY, so anything
// taller would make every ordinary plant look stunted in its plot.
//
// Blueprints are fitted to this box by fitToCanvas rather than the box being
// sized to the largest plant the genome space can produce. TestBlueprintStays// WithinCanvas is what holds that guarantee.
const (
	CanvasMinX = -0.35
	CanvasMaxX = 1.35
	CanvasMinY = -0.40
	CanvasMaxY = 1.00

	CanvasWidth  = CanvasMaxX - CanvasMinX
	CanvasHeight = CanvasMaxY - CanvasMinY
)

// Point is a position in normalised plant space.
type Point struct{ X, Y float64 }

// SegKind identifies which path command a Segment carries.
type SegKind uint8

const (
	SegMove SegKind = iota
	SegLine
	SegQuad
	SegCubic
	SegClose
)

// Segment is one path command. Which entries of P are meaningful depends on
// Kind: Move and Line use P[0]; Quad uses P[0] as its control point and P[1]
// as its end; Cubic uses P[0] and P[1] as controls and P[2] as its end; Close
// uses none.
type Segment struct {
	Kind SegKind
	P    [3]Point
}

func moveTo(p Point) Segment    { return Segment{Kind: SegMove, P: [3]Point{p}} }
func lineTo(p Point) Segment    { return Segment{Kind: SegLine, P: [3]Point{p}} }
func quadTo(c, p Point) Segment { return Segment{Kind: SegQuad, P: [3]Point{c, p}} }
func cubicTo(c1, c2, p Point) Segment {
	return Segment{Kind: SegCubic, P: [3]Point{c1, c2, p}}
}
func closePath() Segment { return Segment{Kind: SegClose} }

// pointCount is how many entries of P the segment actually uses.
func (s Segment) pointCount() int {
	switch s.Kind {
	case SegMove, SegLine:
		return 1
	case SegQuad:
		return 2
	case SegCubic:
		return 3
	default:
		return 0
	}
}

// HSL is a colour in hue-saturation-luminance. Genes are expressed as hue and
// shifts of it, so carrying colour in HSL keeps "shift the hue" an addition
// rather than a conversion. The renderer converts at raster time.
type HSL struct {
	H float64 // degrees, 0..360
	S float64 // 0..1
	L float64 // 0..1
}

// ShapeKind says whether a shape's path is filled or stroked.
type ShapeKind uint8

const (
	ShapeFill ShapeKind = iota
	ShapeStroke
)

// Paint order. Leaves straddle the stem so the plant reads as three
// dimensional, and the flower and fruit sit in front of everything.
const (
	ZBackLeaf  = 10
	ZStem      = 20
	ZBranch    = 25
	ZFrontLeaf = 30
	ZFlower    = 40
	ZFruit     = 50
)

// Shape is one drawable primitive.
type Shape struct {
	Kind  ShapeKind
	Path  []Segment
	Color HSL
	// Width is the stroke width in normalised units. Ignored for fills.
	Width float64
	Z     int
}

// Blueprint is a complete plant as geometry, ready to rasterise. Shapes are
// emitted in ascending Z order.
type Blueprint struct {
	Shapes []Shape
}

func (b *Blueprint) add(s Shape) {
	if len(s.Path) == 0 {
		return
	}
	b.Shapes = append(b.Shapes, s)
}

// Points returns every point referenced by every shape, in a stable order.
// Tests use it to compare two blueprints geometrically.
func (b Blueprint) Points() []Point {
	var out []Point
	for _, s := range b.Shapes {
		for _, seg := range s.Path {
			for i := 0; i < seg.pointCount(); i++ {
				out = append(out, seg.P[i])
			}
		}
	}
	return out
}

// Bounds returns the axis-aligned extent of the blueprint. An empty blueprint
// reports ok=false.
func (b Blueprint) Bounds() (min, max Point, ok bool) {
	pts := b.Points()
	if len(pts) == 0 {
		return Point{}, Point{}, false
	}
	min, max = pts[0], pts[0]
	for _, p := range pts[1:] {
		if p.X < min.X {
			min.X = p.X
		}
		if p.Y < min.Y {
			min.Y = p.Y
		}
		if p.X > max.X {
			max.X = p.X
		}
		if p.Y > max.Y {
			max.Y = p.Y
		}
	}
	return min, max, true
}
