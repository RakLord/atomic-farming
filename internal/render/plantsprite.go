package render

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"atomicfarming/internal/plant"
	"atomicfarming/internal/plant/morph"
)

// StageBuckets is how finely maturity is quantised for caching.
//
// Without quantisation every tick would produce a new maturity and so a new
// texture. Sixteen steps is well past what the eye resolves at plot scale
// while keeping a plant's whole life to at most sixteen rasterisations.
const StageBuckets = 16

// bucketOf quantises maturity into a cache bucket.
func bucketOf(t float64) int {
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return StageBuckets - 1
	}
	b := int(t * StageBuckets)
	if b >= StageBuckets {
		b = StageBuckets - 1
	}
	return b
}

// bucketMaturity is the maturity a bucket is rendered at: the middle of the
// bucket, so quantisation error is symmetric.
func bucketMaturity(bucket int) float64 {
	if bucket >= StageBuckets-1 {
		return 1
	}
	return (float64(bucket) + 0.5) / StageBuckets
}

// spriteSize is the pixel size of a plant sprite at the given scale, where
// scale is the pixel length of one unit of plant space. The sprite covers the
// whole morph canvas, which is wider and taller than the unit square because
// foliage spreads past the stem.
func spriteSize(scale int) (w, h int) {
	return int(math.Ceil(morph.CanvasWidth * float64(scale))),
		int(math.Ceil(morph.CanvasHeight * float64(scale)))
}

// spriteOrigin is where the plant's own origin — the base of the stem on the
// soil line — falls inside its sprite, in pixels.
func spriteOrigin(scale int) (x, y float64) {
	return (0.5 - morph.CanvasMinX) * float64(scale), morph.CanvasMaxY * float64(scale)
}

// rasterise draws a blueprint into a new image at the given scale.
func rasterise(bp morph.Blueprint, scale int) *ebiten.Image {
	w, h := spriteSize(scale)
	if w <= 0 || h <= 0 {
		w, h = 1, 1
	}
	img := ebiten.NewImage(w, h)

	var path vector.Path
	for _, shape := range bp.Shapes {
		path.Reset()
		appendPath(&path, shape.Path, float64(scale))

		var cs ebiten.ColorScale
		cs.ScaleWithColor(hslToRGBA(shape.Color))
		draw := &vector.DrawPathOptions{AntiAlias: true, ColorScale: cs}

		switch shape.Kind {
		case morph.ShapeStroke:
			vector.StrokePath(img, &path, &vector.StrokeOptions{
				Width:    float32(shape.Width * float64(scale)),
				LineCap:  vector.LineCapRound,
				LineJoin: vector.LineJoinRound,
			}, draw)
		default:
			vector.FillPath(img, &path, &vector.FillOptions{FillRule: vector.FillRuleNonZero}, draw)
		}
	}
	return img
}

// appendPath converts normalised plant geometry into sprite pixels. Plant
// space has y increasing upward from the soil; images have y increasing
// downward, so the vertical axis is flipped here.
func appendPath(dst *vector.Path, segs []morph.Segment, scale float64) {
	toPx := func(p morph.Point) (float32, float32) {
		return float32((p.X - morph.CanvasMinX) * scale),
			float32((morph.CanvasMaxY - p.Y) * scale)
	}
	for _, s := range segs {
		switch s.Kind {
		case morph.SegMove:
			x, y := toPx(s.P[0])
			dst.MoveTo(x, y)
		case morph.SegLine:
			x, y := toPx(s.P[0])
			dst.LineTo(x, y)
		case morph.SegQuad:
			cx, cy := toPx(s.P[0])
			x, y := toPx(s.P[1])
			dst.QuadTo(cx, cy, x, y)
		case morph.SegCubic:
			c1x, c1y := toPx(s.P[0])
			c2x, c2y := toPx(s.P[1])
			x, y := toPx(s.P[2])
			dst.CubicTo(c1x, c1y, c2x, c2y, x, y)
		case morph.SegClose:
			dst.Close()
		}
	}
}

// hslToRGBA converts a genome-derived colour into a drawable one. Colour is
// carried through the blueprint as HSL because genes express hue and shifts
// of it; the conversion belongs here, at the last possible moment.
func hslToRGBA(c morph.HSL) color.RGBA {
	h := math.Mod(c.H, 360)
	if h < 0 {
		h += 360
	}
	h /= 360
	s, l := clampUnit(c.S), clampUnit(c.L)

	if s == 0 {
		v := to8(l)
		return color.RGBA{R: v, G: v, B: v, A: 0xff}
	}
	var q float64
	if l < 0.5 {
		q = l * (1 + s)
	} else {
		q = l + s - l*s
	}
	p := 2*l - q
	return color.RGBA{
		R: to8(hueToChannel(p, q, h+1.0/3)),
		G: to8(hueToChannel(p, q, h)),
		B: to8(hueToChannel(p, q, h-1.0/3)),
		A: 0xff,
	}
}

func hueToChannel(p, q, t float64) float64 {
	if t < 0 {
		t++
	}
	if t > 1 {
		t--
	}
	switch {
	case t < 1.0/6:
		return p + (q-p)*6*t
	case t < 1.0/2:
		return q
	case t < 2.0/3:
		return p + (q-p)*(2.0/3-t)*6
	default:
		return p
	}
}

func to8(v float64) uint8 { return uint8(clampUnit(v)*255 + 0.5) }

func clampUnit(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// PlantPhenotypeKey hashes a phenotype for cache keying.
//
// Keying on the phenotype rather than the genome is what keeps species out of
// the cache key: two species express the same genome differently, but the
// same phenotype always draws the same plant.
func PlantPhenotypeKey(p plant.Phenotype) uint64 {
	var h uint64 = 0xcbf29ce484222325
	for i := 0; i < plant.GeneCount; i++ {
		h = plant.Hash64(h ^ uint64(p[i]))
	}
	return h
}
