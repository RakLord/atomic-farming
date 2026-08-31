package render

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"atomicfarming/internal/plant"
	"atomicfarming/internal/plant/morph"
)

// DefaultSpriteCacheSize caps how many plant textures are held at once. A
// full farm shows at most 144 plots, and a field of one strain shares a single
// sprite, so this is generous in practice while bounding texture memory on
// the WASM build.
const DefaultSpriteCacheSize = 64

type spriteKey struct {
	pheno  uint64
	bucket int
	scale  int
}

type spriteEntry struct {
	img  *ebiten.Image
	used uint64
}

// SpriteCache rasterises plants on demand and keeps the most recently used
// textures. It is not safe for concurrent use; the render loop is the only
// caller.
type SpriteCache struct {
	entries map[spriteKey]*spriteEntry
	cap     int
	clock   uint64
	hits    int
	misses  int
}

func NewSpriteCache(capacity int) *SpriteCache {
	if capacity < 1 {
		capacity = DefaultSpriteCacheSize
	}
	return &SpriteCache{entries: make(map[spriteKey]*spriteEntry, capacity), cap: capacity}
}

// Sprite returns the texture for a plant of the given phenotype at maturity t,
// drawn at scale pixels per unit of plant space.
//
// Maturity is quantised into StageBuckets before lookup, so a plant growing
// through a tick does not invalidate its texture on every frame.
func (c *SpriteCache) Sprite(p plant.Phenotype, t float64, scale int) *ebiten.Image {
	if scale < 1 {
		scale = 1
	}
	key := spriteKey{pheno: PlantPhenotypeKey(p), bucket: bucketOf(t), scale: scale}

	c.clock++
	if e, ok := c.entries[key]; ok {
		e.used = c.clock
		c.hits++
		return e.img
	}

	c.misses++
	img := rasterise(morph.Build(p, bucketMaturity(key.bucket)), scale)
	if len(c.entries) >= c.cap {
		c.evictLeastRecentlyUsed()
	}
	c.entries[key] = &spriteEntry{img: img, used: c.clock}
	return img
}

// evictLeastRecentlyUsed drops one entry and releases its texture.
//
// A linear scan is fine here: the cache is small and capped, an eviction only
// happens on a miss once the cache is full, and a scan avoids maintaining a
// second ordering structure that could drift out of step with the map.
func (c *SpriteCache) evictLeastRecentlyUsed() {
	var oldestKey spriteKey
	var oldest *spriteEntry
	for k, e := range c.entries {
		if oldest == nil || e.used < oldest.used {
			oldestKey, oldest = k, e
		}
	}
	if oldest == nil {
		return
	}
	delete(c.entries, oldestKey)
	// Release the GPU texture rather than waiting for the finaliser.
	oldest.img.Deallocate()
}

// Len is how many textures are currently held.
func (c *SpriteCache) Len() int { return len(c.entries) }

// Stats reports cache hits and misses, for the lab's diagnostics.
func (c *SpriteCache) Stats() (hits, misses int) { return c.hits, c.misses }

// Clear drops every texture. Used when the farm is reset or the plot size
// changes enough that cached scales are all stale.
func (c *SpriteCache) Clear() {
	for k, e := range c.entries {
		e.img.Deallocate()
		delete(c.entries, k)
	}
	c.hits, c.misses = 0, 0
}

// DrawPlant draws a plant standing on the soil line at the bottom of the given
// rect, horizontally centred, clipped to the rect.
//
// The sprite is both wider and taller than the rect — foliage spreads past the
// stem, and the canvas reserves room below the soil for leaves that rest on
// the ground — so it is positioned by the plant's own origin. Bottom-aligning
// the sprite instead would leave every plant hovering a quarter of a tile
// above the ground.
func (c *SpriteCache) DrawPlant(dst *ebiten.Image, p plant.Phenotype, t float64, x, y, w, h int) {
	// Scale so the above-ground part of the canvas fills the rect's height.
	scale := int(float64(h) / morph.CanvasMaxY)
	c.drawAt(dst, p, t, x, y, w, h, scale)
}

// DrawPlantFitted draws the plant scaled so its full width also fits inside
// the rect. Used for previews, where a plant clipped at the sides looks broken;
// on the farm, overhanging neighbours look natural.
func (c *SpriteCache) DrawPlantFitted(dst *ebiten.Image, p plant.Phenotype, t float64, x, y, w, h int) {
	byWidth := float64(w) / morph.CanvasWidth
	byHeight := float64(h) / morph.CanvasMaxY
	c.drawAt(dst, p, t, x, y, w, h, int(math.Min(byWidth, byHeight)))
}

func (c *SpriteCache) drawAt(dst *ebiten.Image, p plant.Phenotype, t float64, x, y, w, h, scale int) {
	if scale < 1 || w <= 0 || h <= 0 {
		return
	}
	img := c.Sprite(p, t, scale)
	ox, oy := spriteOrigin(scale)

	// Clip to the rect: the canvas dips below the soil line, and that overhang
	// must not paint over whatever sits underneath.
	clip := image.Rect(x, y, x+w, y+h).Intersect(dst.Bounds())
	if clip.Empty() {
		return
	}
	target := dst.SubImage(clip).(*ebiten.Image)

	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterLinear
	// Place the plant's own origin — the base of the stem — on the rect floor.
	op.GeoM.Translate(float64(x)+float64(w)/2-ox, float64(y+h)-oy)
	target.DrawImage(img, op)
}
