package render

import (
	"testing"

	"atomicfarming/internal/plant"
)

func TestSpriteCacheReusesRasterisedPlants(t *testing.T) {
	c := NewSpriteCache(8)
	p := plant.ExpressFull(plant.RandomGenome(1))

	first := c.Sprite(p, 0.5, 32)
	second := c.Sprite(p, 0.5, 32)
	if first != second {
		t.Error("the same plant at the same maturity rasterised twice")
	}
	if hits, misses := c.Stats(); hits != 1 || misses != 1 {
		t.Errorf("stats = %d hits, %d misses; want 1 and 1", hits, misses)
	}
}

// TestSpriteCacheQuantisesMaturity is why the cache is affordable: a plant
// growing a fraction of a stage must not force a re-rasterisation.
func TestSpriteCacheQuantisesMaturity(t *testing.T) {
	c := NewSpriteCache(8)
	p := plant.ExpressFull(plant.RandomGenome(2))

	base := c.Sprite(p, 0.50, 32)
	nudged := c.Sprite(p, 0.50+1.0/(StageBuckets*4), 32)
	if base != nudged {
		t.Error("a sub-bucket maturity change forced a new texture")
	}
	if stepped := c.Sprite(p, 0.50+2.0/StageBuckets, 32); stepped == base {
		t.Error("a two-bucket maturity change reused the old texture")
	}
}

func TestSpriteCacheKeysOnScale(t *testing.T) {
	c := NewSpriteCache(8)
	p := plant.ExpressFull(plant.RandomGenome(3))
	if small, large := c.Sprite(p, 1, 24), c.Sprite(p, 1, 48); small == large {
		t.Error("two plot sizes shared one texture; the farm would draw blurred plants after expanding")
	}
}

func TestSpriteCacheRespectsItsCapacity(t *testing.T) {
	const capacity = 4
	c := NewSpriteCache(capacity)
	for seed := uint64(0); seed < 40; seed++ {
		c.Sprite(plant.ExpressFull(plant.RandomGenome(seed)), 1, 16)
		if c.Len() > capacity {
			t.Fatalf("cache grew to %d entries, past its capacity of %d", c.Len(), capacity)
		}
	}
	if c.Len() != capacity {
		t.Errorf("cache settled at %d entries, want %d", c.Len(), capacity)
	}
}

func TestSpriteCacheEvictsTheLeastRecentlyUsed(t *testing.T) {
	c := NewSpriteCache(2)
	a := plant.ExpressFull(plant.RandomGenome(10))
	b := plant.ExpressFull(plant.RandomGenome(11))
	d := plant.ExpressFull(plant.RandomGenome(12))

	keptA := c.Sprite(a, 1, 16)
	c.Sprite(b, 1, 16)
	c.Sprite(a, 1, 16) // touch a, making b the least recently used
	c.Sprite(d, 1, 16) // evicts b

	if again := c.Sprite(a, 1, 16); again != keptA {
		t.Error("the recently used plant was evicted instead of the stale one")
	}
}

func TestSpriteCacheClearReleasesEverything(t *testing.T) {
	c := NewSpriteCache(8)
	for seed := uint64(0); seed < 5; seed++ {
		c.Sprite(plant.ExpressFull(plant.RandomGenome(seed)), 1, 16)
	}
	c.Clear()
	if c.Len() != 0 {
		t.Errorf("cache holds %d entries after Clear", c.Len())
	}
	if hits, misses := c.Stats(); hits != 0 || misses != 0 {
		t.Errorf("stats not reset after Clear: %d hits, %d misses", hits, misses)
	}
}

func TestSpriteCacheGuardsAgainstNonsenseInput(t *testing.T) {
	c := NewSpriteCache(0) // falls back to the default capacity
	if c.cap != DefaultSpriteCacheSize {
		t.Errorf("capacity = %d, want the default %d", c.cap, DefaultSpriteCacheSize)
	}
	if img := c.Sprite(plant.ExpressFull(plant.DefaultGenome()), 0.5, -4); img == nil {
		t.Error("a nonsense scale produced no sprite")
	}
}
