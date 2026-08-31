package render

import (
	"image/color"
	"testing"

	"atomicfarming/internal/plant"
	"atomicfarming/internal/plant/morph"
)

func TestBucketOfCoversMaturityWithoutEscaping(t *testing.T) {
	if got := bucketOf(0); got != 0 {
		t.Errorf("bucketOf(0) = %d, want 0", got)
	}
	if got := bucketOf(1); got != StageBuckets-1 {
		t.Errorf("bucketOf(1) = %d, want %d", got, StageBuckets-1)
	}
	if got := bucketOf(-3); got != 0 {
		t.Errorf("bucketOf(-3) = %d, want 0", got)
	}
	if got := bucketOf(9); got != StageBuckets-1 {
		t.Errorf("bucketOf(9) = %d, want %d", got, StageBuckets-1)
	}

	prev := -1
	seen := map[int]bool{}
	for i := 0; i <= 1000; i++ {
		b := bucketOf(float64(i) / 1000)
		if b < 0 || b >= StageBuckets {
			t.Fatalf("bucketOf produced out-of-range bucket %d", b)
		}
		if b < prev {
			t.Fatalf("bucketOf is not monotonic: %d then %d", prev, b)
		}
		prev = b
		seen[b] = true
	}
	if len(seen) != StageBuckets {
		t.Errorf("only %d of %d buckets are reachable", len(seen), StageBuckets)
	}
}

func TestBucketMaturityStaysInsideItsBucket(t *testing.T) {
	for b := 0; b < StageBuckets; b++ {
		m := bucketMaturity(b)
		if m < 0 || m > 1 {
			t.Fatalf("bucket %d maturity %v is outside [0,1]", b, m)
		}
		if got := bucketOf(m); got != b {
			t.Errorf("bucket %d renders at maturity %v, which buckets back to %d", b, m, got)
		}
	}
}

func TestSpriteGeometryMatchesTheCanvas(t *testing.T) {
	const scale = 100
	w, h := spriteSize(scale)
	if w < scale || h < scale {
		t.Errorf("sprite %dx%d is smaller than one plant unit at scale %d", w, h, scale)
	}
	ox, oy := spriteOrigin(scale)
	if ox < 0 || ox > float64(w) {
		t.Errorf("plant origin x %.1f falls outside the sprite width %d", ox, w)
	}
	if oy < 0 || oy > float64(h) {
		t.Errorf("plant origin y %.1f falls outside the sprite height %d", oy, h)
	}

	// Every point a blueprint can produce must land inside the sprite, or
	// plants would be clipped at the texture edge.
	corners := []morph.Point{
		{X: morph.CanvasMinX, Y: morph.CanvasMinY},
		{X: morph.CanvasMaxX, Y: morph.CanvasMaxY},
	}
	for _, c := range corners {
		px := (c.X - morph.CanvasMinX) * scale
		py := (morph.CanvasMaxY - c.Y) * scale
		if px < -0.001 || px > float64(w)+0.001 || py < -0.001 || py > float64(h)+0.001 {
			t.Errorf("canvas corner %+v maps to (%.2f, %.2f), outside the %dx%d sprite", c, px, py, w, h)
		}
	}
}

func TestHSLToRGBAKnownColours(t *testing.T) {
	tests := []struct {
		name string
		in   morph.HSL
		want color.RGBA
	}{
		{"black", morph.HSL{H: 0, S: 0, L: 0}, color.RGBA{0, 0, 0, 255}},
		{"white", morph.HSL{H: 0, S: 0, L: 1}, color.RGBA{255, 255, 255, 255}},
		{"red", morph.HSL{H: 0, S: 1, L: 0.5}, color.RGBA{255, 0, 0, 255}},
		{"green", morph.HSL{H: 120, S: 1, L: 0.5}, color.RGBA{0, 255, 0, 255}},
		{"blue", morph.HSL{H: 240, S: 1, L: 0.5}, color.RGBA{0, 0, 255, 255}},
		{"mid grey", morph.HSL{H: 200, S: 0, L: 0.5}, color.RGBA{128, 128, 128, 255}},
	}
	for _, tc := range tests {
		if got := hslToRGBA(tc.in); got != tc.want {
			t.Errorf("%s: hslToRGBA(%+v) = %v, want %v", tc.name, tc.in, got, tc.want)
		}
	}
}

func TestHSLToRGBAHandlesOutOfRangeInput(t *testing.T) {
	// Hue wraps and saturation/luminance clamp, so a gene combination can
	// never produce an invalid colour.
	if got, want := hslToRGBA(morph.HSL{H: 480, S: 1, L: 0.5}), hslToRGBA(morph.HSL{H: 120, S: 1, L: 0.5}); got != want {
		t.Errorf("hue 480 gave %v, want the same as hue 120 (%v)", got, want)
	}
	if got, want := hslToRGBA(morph.HSL{H: -120, S: 1, L: 0.5}), hslToRGBA(morph.HSL{H: 240, S: 1, L: 0.5}); got != want {
		t.Errorf("hue -120 gave %v, want the same as hue 240 (%v)", got, want)
	}
	if got := hslToRGBA(morph.HSL{H: 0, S: 5, L: 9}); got != (color.RGBA{255, 255, 255, 255}) {
		t.Errorf("out-of-range saturation/luminance gave %v, want white", got)
	}
}

func TestPhenotypeKeyIsStableAndDiscriminating(t *testing.T) {
	p := plant.ExpressFull(plant.RandomGenome(31))
	first := PlantPhenotypeKey(p)
	if again := PlantPhenotypeKey(p); again != first {
		t.Error("PlantPhenotypeKey is not stable")
	}

	q := p
	q[plant.GeneStemHeight]++
	if PlantPhenotypeKey(q) == first {
		t.Error("a one-step phenotype change did not change the key; the cache would serve a stale sprite")
	}

	// Two genomes that express identically must share a key, so a field of
	// one strain rasterises once.
	seen := map[uint64]bool{}
	for seed := uint64(0); seed < 400; seed++ {
		seen[PlantPhenotypeKey(plant.ExpressFull(plant.RandomGenome(seed)))] = true
	}
	if len(seen) < 395 {
		t.Errorf("only %d distinct keys from 400 genomes; collisions would cross-render plants", len(seen))
	}
}
