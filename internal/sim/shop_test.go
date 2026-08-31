package sim

import (
	"testing"

	"atomicfarming/internal/bignum"
	"atomicfarming/internal/plant"
)

// withShop registers offers and unlocks for one test and restores the
// catalogs afterwards; both are package state shared by every test.
func withShop(t *testing.T, offers []SeedOffer, unlocks []Unlock) {
	t.Helper()
	savedShop := make(map[SeedOfferID]SeedOffer, len(SeedShop))
	for k, v := range SeedShop {
		savedShop[k] = v
	}
	savedShopOrder := append([]SeedOfferID(nil), SeedShopOrder...)
	savedUnlocks := make(map[UnlockID]Unlock, len(UnlockCatalog))
	for k, v := range UnlockCatalog {
		savedUnlocks[k] = v
	}
	savedUnlockOrder := append([]UnlockID(nil), UnlockCatalogOrder...)
	t.Cleanup(func() {
		SeedShop, SeedShopOrder = savedShop, savedShopOrder
		UnlockCatalog, UnlockCatalogOrder = savedUnlocks, savedUnlockOrder
	})

	for _, u := range unlocks {
		RegisterUnlock(u)
	}
	for _, o := range offers {
		RegisterSeedOffer(o)
	}
}

const (
	testOffer  SeedOfferID = "test_offer"
	testGated  SeedOfferID = "test_gated"
	testUnlock UnlockID    = "test_unlock"
)

func basicShop(t *testing.T) {
	withShop(t,
		[]SeedOffer{
			{ID: testOffer, Kind: kindTestCrop, Name: "Test Seed",
				Genome: plant.DefaultGenome(), BaseCost: bignum.MustParse("10")},
			{ID: testGated, Kind: kindTestCrop, Name: "Gated Seed",
				Genome: plant.RandomGenome(99), BaseCost: bignum.MustParse("100"),
				RequiresUnlock: testUnlock},
		},
		[]Unlock{{ID: testUnlock, Name: "Test Licence", Cost: bignum.MustParse("50")}},
	)
}

func TestBuyingASeedDeductsCashAndStocksIt(t *testing.T) {
	basicShop(t)
	s := NewGameState()
	s.Cash = bignum.MustParse("100")
	before := s.Inventory.Total()

	if err := BuySeed(s, testOffer); err != nil {
		t.Fatalf("BuySeed: %v", err)
	}
	if want := bignum.MustParse("90"); !s.Cash.Eq(want) {
		t.Errorf("cash = %s, want %s", s.Cash, want)
	}
	if s.Inventory.Total() != before+1 {
		t.Errorf("inventory holds %d, want %d", s.Inventory.Total(), before+1)
	}
}

func TestBuyingWithoutCashChangesNothing(t *testing.T) {
	basicShop(t)
	s := NewGameState()
	s.Cash = bignum.MustParse("3")
	before := s.Inventory.Total()

	if err := BuySeed(s, testOffer); err != ErrTooExpensive {
		t.Fatalf("BuySeed gave %v, want ErrTooExpensive", err)
	}
	if !s.Cash.Eq(bignum.MustParse("3")) || s.Inventory.Total() != before {
		t.Error("a failed purchase changed state")
	}
	if CanBuySeed(s, testOffer) {
		t.Error("CanBuySeed says yes without the cash")
	}
}

func TestSeedCostRespectsTheGlobalModifier(t *testing.T) {
	basicShop(t)
	s := NewGameState()
	s.Modifiers.SeedCostMul = bignum.MustParse("0.5")
	if want := bignum.MustParse("5"); !SeedCost(s, testOffer).Eq(want) {
		t.Errorf("cost = %s, want %s", SeedCost(s, testOffer), want)
	}
}

// TestRareSeedsAreGatedByAnUnlock covers the decision that rare seeds need no
// economy of their own: an unlock flag plus an ordinary purchase.
func TestRareSeedsAreGatedByAnUnlock(t *testing.T) {
	basicShop(t)
	s := NewGameState()
	s.Cash = bignum.MustParse("1000")

	if SeedOfferAvailable(s, testGated) {
		t.Fatal("a gated offer was available before its unlock")
	}
	if err := BuySeed(s, testGated); err != ErrLocked {
		t.Fatalf("buying a locked seed gave %v, want ErrLocked", err)
	}

	if err := PurchaseUnlock(s, testUnlock); err != nil {
		t.Fatalf("PurchaseUnlock: %v", err)
	}
	if !SeedOfferAvailable(s, testGated) {
		t.Fatal("the offer is still locked after buying its unlock")
	}
	if err := BuySeed(s, testGated); err != nil {
		t.Fatalf("BuySeed after unlocking: %v", err)
	}
}

func TestPurchasingAnUnlockIsAllOrNothing(t *testing.T) {
	basicShop(t)
	s := NewGameState()
	s.Cash = bignum.MustParse("10")

	if err := PurchaseUnlock(s, testUnlock); err != ErrTooExpensive {
		t.Fatalf("got %v, want ErrTooExpensive", err)
	}
	if IsUnlocked(s, testUnlock) {
		t.Error("the unlock was granted despite the purchase failing")
	}

	s.Cash = bignum.MustParse("60")
	if err := PurchaseUnlock(s, testUnlock); err != nil {
		t.Fatalf("PurchaseUnlock: %v", err)
	}
	if err := PurchaseUnlock(s, testUnlock); err == nil {
		t.Error("an already-owned unlock was purchased twice")
	}
}

func TestUnknownOfferIsRejected(t *testing.T) {
	basicShop(t)
	s := NewGameState()
	s.Cash = bignum.MustParse("1000")
	if err := BuySeed(s, "no_such_offer"); err != ErrNoSeed {
		t.Errorf("got %v, want ErrNoSeed", err)
	}
}
