package sim

import (
	"fmt"

	"atomicfarming/internal/plant"
)

// StrainID identifies a named strain. It is persisted in the discovery log,
// so it is a save-format constant: never rename one that has shipped.
type StrainID string

// StrainRarity is display and ordering only; it grants nothing.
type StrainRarity uint8

const (
	RarityCommon StrainRarity = iota
	RarityUncommon
	RarityRare
	RarityLegendary
	rarityCount
)

var rarityNames = [rarityCount]string{"Common", "Uncommon", "Rare", "Legendary"}

func (r StrainRarity) String() string {
	if r < rarityCount {
		return rarityNames[r]
	}
	return "Unknown"
}

// NamedStrain is a plant worth recognising.
//
// A strain is matched one of two ways, and the difference is what each is for:
//
//   - Match is a predicate over expressed traits. Because it describes visible
//     conditions, its Goal doubles as a breeding target the UI can show — this
//     is the kind a player can deliberately work toward.
//   - Signature is one exact genome. Reaching it by breeding is effectively
//     impossible, so it is for handcrafted strains that arrive as a bought or
//     granted seed.
//
// Exactly one of the two must be set; init validates it.
type NamedStrain struct {
	ID          StrainID
	Name        string
	Description string
	// Goal states the condition in player-facing terms, so a predicate strain
	// reads as something to aim at rather than a lottery win.
	Goal   string
	Rarity StrainRarity
	// Kind restricts the strain to one species. Empty matches any.
	Kind CropKind

	Match     func(p plant.Phenotype) bool
	Signature *plant.Genome

	// Specificity breaks ties when several strains match one plant; the
	// highest wins. Give a stricter strain a higher value than a looser one it
	// sits inside.
	Specificity int
}

// StrainCatalog is the code-defined set of named strains.
var StrainCatalog = map[StrainID]NamedStrain{}

// StrainCatalogOrder is the iteration and display order.
//
// IdentifyStrain walks this slice rather than the map: Go randomises map
// iteration order, so a plant resolved through the map could name itself
// differently between frames.
var StrainCatalogOrder []StrainID

// RegisterStrain adds a strain to the catalog. Duplicate IDs panic — an ID is
// a save-format identifier in the discovery log and must be unique.
func RegisterStrain(s NamedStrain) {
	if s.ID == "" {
		panic("sim: named strain registered with no ID")
	}
	if s.Name == "" {
		panic(fmt.Sprintf("sim: strain %q has no name", s.ID))
	}
	if (s.Match == nil) == (s.Signature == nil) {
		panic(fmt.Sprintf("sim: strain %q must set exactly one of Match or Signature", s.ID))
	}
	if _, exists := StrainCatalog[s.ID]; exists {
		panic(fmt.Sprintf("sim: duplicate strain registration for %q", s.ID))
	}
	StrainCatalog[s.ID] = s
	StrainCatalogOrder = append(StrainCatalogOrder, s.ID)
}

// IdentifyStrain returns the named strain a plant qualifies as.
//
// The result is derived, never stored. A persisted name could only ever go
// stale; deriving it means adding or retuning a strain applies retroactively
// to plants already sitting in a save, with no migration — the same rule that
// governs GlobalModifiers. See docs/adr/0012-named-strains.md.
func IdentifyStrain(kind CropKind, g plant.Genome, p plant.Phenotype) (NamedStrain, bool) {
	var best NamedStrain
	found := false
	for _, id := range StrainCatalogOrder {
		s, ok := StrainCatalog[id]
		if !ok {
			continue
		}
		if s.Kind != "" && s.Kind != kind {
			continue
		}
		if !strainMatches(s, g, p) {
			continue
		}
		if !found || s.Specificity > best.Specificity {
			best, found = s, true
		}
	}
	return best, found
}

func strainMatches(s NamedStrain, g plant.Genome, p plant.Phenotype) bool {
	if s.Signature != nil {
		return *s.Signature == g
	}
	return s.Match != nil && s.Match(p)
}

// StrainName is the display name for a plant: its strain if it has one, and
// otherwise the fallback the caller supplies (normally the species name).
func StrainName(kind CropKind, g plant.Genome, p plant.Phenotype, fallback string) string {
	if s, ok := IdentifyStrain(kind, g, p); ok {
		return s.Name
	}
	return fallback
}

// Discover records a strain in the player's log, reporting whether it was new.
// The log is the only part of the naming system that is real state.
func (s *GameState) Discover(id StrainID) bool {
	if s == nil || id == "" {
		return false
	}
	if s.DiscoveredStrains == nil {
		s.DiscoveredStrains = map[StrainID]bool{}
	}
	if s.DiscoveredStrains[id] {
		return false
	}
	s.DiscoveredStrains[id] = true
	return true
}

// DiscoverPlant identifies a plant and logs it. Returns the strain and whether
// it was newly discovered.
func (s *GameState) DiscoverPlant(kind CropKind, g plant.Genome, p plant.Phenotype) (NamedStrain, bool) {
	strain, ok := IdentifyStrain(kind, g, p)
	if !ok {
		return NamedStrain{}, false
	}
	return strain, s.Discover(strain.ID)
}

// DiscoveredCount is how many named strains the player has logged, ignoring
// any entry from a build that knew strains this one does not.
func (s *GameState) DiscoveredCount() int {
	n := 0
	for id, found := range s.DiscoveredStrains {
		if found {
			if _, known := StrainCatalog[id]; known {
				n++
			}
		}
	}
	return n
}
