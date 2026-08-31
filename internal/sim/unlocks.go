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

// IsUnlocked reports whether the player owns the given unlock.
func IsUnlocked(s *GameState, id UnlockID) bool {
	return s != nil && s.Unlocks[id]
}
