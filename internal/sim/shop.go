package sim

import (
	"fmt"

	"atomicfarming/internal/bignum"
	"atomicfarming/internal/plant"
)

// SeedOfferID identifies a shop entry. Not persisted — the shop is code.
type SeedOfferID string

// SeedOffer is a seed the shop sells.
//
// Rare seeds need no economy of their own: they set RequiresUnlock and are
// gated by the existing unlock catalog, so buying one is two ordinary cash
// purchases — the unlock, then the seed.
type SeedOffer struct {
	ID          SeedOfferID
	Kind        CropKind
	Name        string
	Description string
	Genome      plant.Genome
	BaseCost    bignum.Decimal
	// RequiresUnlock gates the offer. Empty means always on sale.
	RequiresUnlock UnlockID
}

var (
	SeedShop      = map[SeedOfferID]SeedOffer{}
	SeedShopOrder []SeedOfferID
)

// RegisterSeedOffer adds an entry to the shop. Concrete crops call it from
// init() alongside their own registration, so adding a crop still touches only
// one directory.
func RegisterSeedOffer(o SeedOffer) {
	if o.ID == "" {
		panic("sim: seed offer registered with no ID")
	}
	if _, exists := SeedShop[o.ID]; exists {
		panic(fmt.Sprintf("sim: duplicate seed offer %q", o.ID))
	}
	SeedShop[o.ID] = o
	SeedShopOrder = append(SeedShopOrder, o.ID)
}

// SeedOfferAvailable reports whether an offer is on sale for this player.
func SeedOfferAvailable(s *GameState, id SeedOfferID) bool {
	o, ok := SeedShop[id]
	if !ok {
		return false
	}
	return o.RequiresUnlock == "" || IsUnlocked(s, o.RequiresUnlock)
}

// SeedCost is an offer's price after global modifiers.
func SeedCost(s *GameState, id SeedOfferID) bignum.Decimal {
	o, ok := SeedShop[id]
	if !ok {
		return bignum.Zero()
	}
	mods := GlobalModifiers{}.Normalized()
	if s != nil {
		mods = s.Modifiers.Normalized()
	}
	return o.BaseCost.Mul(mods.SeedCostMul)
}

// CanBuySeed reports whether the offer is available and affordable.
func CanBuySeed(s *GameState, id SeedOfferID) bool {
	if s == nil || !SeedOfferAvailable(s, id) {
		return false
	}
	return s.Cash.GTE(SeedCost(s, id))
}

// BuySeed purchases one seed. All-or-nothing: a failure changes nothing.
func BuySeed(s *GameState, id SeedOfferID) error {
	if s == nil {
		return ErrNoSeed
	}
	o, ok := SeedShop[id]
	if !ok {
		return ErrNoSeed
	}
	if !SeedOfferAvailable(s, id) {
		return ErrLocked
	}
	cost := SeedCost(s, id)
	if s.Cash.LT(cost) {
		return ErrTooExpensive
	}
	s.Cash = s.Cash.Sub(cost)
	s.Inventory.Add(o.Kind, o.Genome, 1)
	// Buying a named seed counts as meeting it.
	if crop, err := newCropByKind(o.Kind); err == nil {
		s.DiscoverPlant(o.Kind, o.Genome, plant.Express(o.Genome, crop.Ranges()))
	}
	return nil
}
