package sim

import "atomicfarming/internal/bignum"

// UnlockID identifies a purchased global upgrade. It is a save-format
// constant: the set of owned IDs is what persists, not their effects.
type UnlockID string

// Unlock is one entry in the global upgrade catalog.
//
// Apply is a closure rather than a static multiplier so a single unlock can
// touch several modifier fields at once. Every Apply must be idempotent under
// rebuildModifiers: the same owned set must always produce the same
// GlobalModifiers, whatever order the map iterates in.
type Unlock struct {
	ID          UnlockID
	Name        string
	Description string
	Cost        bignum.Decimal
	Apply       func(m *GlobalModifiers)
}

// UnlockCatalog is the code-defined set of global upgrades. It is
// deliberately empty in the scaffold — Phase 2 fills it. Adding an entry
// requires no changes to the pipeline that reads it.
var UnlockCatalog = map[UnlockID]Unlock{}

// UnlockCatalogOrder is the display order for the catalog. Entries added to
// UnlockCatalog must be listed here to appear in the UI.
var UnlockCatalogOrder = []UnlockID{}

// RegisterUnlock adds an entry to the catalog. Concrete crops call it from
// init() to gate their rare seeds.
//
// Apply may be nil: an unlock that only gates a shop entry contributes no
// modifier, and rebuildModifiers already skips nil applies.
func RegisterUnlock(u Unlock) {
	if u.ID == "" {
		panic("sim: unlock registered with no ID")
	}
	if _, exists := UnlockCatalog[u.ID]; exists {
		panic("sim: duplicate unlock " + string(u.ID))
	}
	UnlockCatalog[u.ID] = u
	UnlockCatalogOrder = append(UnlockCatalogOrder, u.ID)
}

// CanPurchaseUnlock reports whether the player can afford an unpurchased unlock.
func CanPurchaseUnlock(s *GameState, id UnlockID) bool {
	u, ok := UnlockCatalog[id]
	if !ok || s == nil || IsUnlocked(s, id) {
		return false
	}
	return s.Cash.GTE(u.Cost)
}

// PurchaseUnlock buys a global upgrade. All-or-nothing: a failure changes
// nothing. Rebuilding the modifier cache here is what keeps Unlocks the single
// source of truth. See docs/adr/0006-global-modifier-pipeline.md.
func PurchaseUnlock(s *GameState, id UnlockID) error {
	if s == nil {
		return ErrLocked
	}
	u, ok := UnlockCatalog[id]
	if !ok {
		return ErrLocked
	}
	if IsUnlocked(s, id) {
		return ErrLocked
	}
	if s.Cash.LT(u.Cost) {
		return ErrTooExpensive
	}
	s.Cash = s.Cash.Sub(u.Cost)
	if s.Unlocks == nil {
		s.Unlocks = map[UnlockID]bool{}
	}
	s.Unlocks[id] = true
	rebuildModifiers(s)
	return nil
}

// IsUnlocked reports whether the player owns the given unlock.
func IsUnlocked(s *GameState, id UnlockID) bool {
	return s != nil && s.Unlocks[id]
}
