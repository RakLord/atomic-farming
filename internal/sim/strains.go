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

// GeneCondition is one requirement a strain places on a gene.
//
// Requirements are data rather than a closure so the game can say how far a
// plant is from a strain — "Density 148/210" — instead of only whether it
// arrived. A closure could express more, such as a sum across two genes, but a
// requirement that cannot be stated as gene bands is one the player cannot aim
// at either, and aimability is the entire point of a bred strain.
type GeneCondition struct {
	Gene plant.GeneID
	// Min and Max are inclusive.
	Min, Max uint8
	// Outside inverts the band: the gene must fall outside [Min, Max] rather
	// than within it. This is what lets a single condition type express a
	// requirement like "strongly bent either way" without an Any/All split.
	Outside bool
}

// Holds reports whether a phenotype satisfies the condition.
func (c GeneCondition) Holds(p plant.Phenotype) bool {
	v := p.Get(c.Gene)
	inside := v >= c.Min && v <= c.Max
	if c.Outside {
		return !inside
	}
	return inside
}

// GeneName is the condition's gene in player-facing terms.
func (c GeneCondition) GeneName() string {
	if !c.Gene.Valid() {
		return "Unknown"
	}
	return plant.GeneCatalog[c.Gene].Name
}

// Requirement states the bound alone, for pairing with a current value.
func (c GeneCondition) Requirement() string {
	switch {
	case c.Outside:
		return fmt.Sprintf("outside %d-%d", c.Min, c.Max)
	case c.Min == 0:
		return fmt.Sprintf("%d or less", c.Max)
	case c.Max == 255:
		return fmt.Sprintf("%d+", c.Min)
	default:
		return fmt.Sprintf("%d-%d", c.Min, c.Max)
	}
}

// Describe is the whole requirement as a phrase.
func (c GeneCondition) Describe() string {
	return c.GeneName() + " " + c.Requirement()
}

// NamedStrain is a plant worth recognising.
//
// A strain is matched one of two ways, and the difference is what each is for:
//
//   - Conditions are requirements on expressed traits, all of which must hold.
//     Because they are data, the game can show how far a plant is from the
//     strain, which is what makes this kind something to work toward.
//   - Signature is one exact genome. Reaching it by breeding is effectively
//     impossible, so it is for handcrafted strains that arrive as a bought or
//     granted seed.
//
// Exactly one of the two must be set; RegisterStrain validates it.
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

	Conditions []GeneCondition
	Signature  *plant.Genome

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
	if (len(s.Conditions) == 0) == (s.Signature == nil) {
		panic(fmt.Sprintf("sim: strain %q must set exactly one of Conditions or Signature", s.ID))
	}
	for _, c := range s.Conditions {
		if !c.Gene.Valid() {
			panic(fmt.Sprintf("sim: strain %q has a condition on gene %d, which does not exist", s.ID, c.Gene))
		}
		if c.Min > c.Max {
			panic(fmt.Sprintf("sim: strain %q has an inverted band on %s", s.ID, c.GeneName()))
		}
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
		if !s.Matches(g, p) {
			continue
		}
		if !found || s.Specificity > best.Specificity {
			best, found = s, true
		}
	}
	return best, found
}

// Matches reports whether a plant qualifies as this strain.
func (s NamedStrain) Matches(g plant.Genome, p plant.Phenotype) bool {
	if s.Signature != nil {
		return *s.Signature == g
	}
	if len(s.Conditions) == 0 {
		return false
	}
	for _, c := range s.Conditions {
		if !c.Holds(p) {
			return false
		}
	}
	return true
}

// Breedable reports whether a strain can be reached by working toward it, as
// opposed to arriving as one exact handcrafted genome.
func (s NamedStrain) Breedable() bool { return s.Signature == nil }

// ConditionProgress is one requirement measured against a plant.
type ConditionProgress struct {
	Condition GeneCondition
	Current   uint8
	Met       bool
}

// StrainProgress is how close a plant is to one strain.
type StrainProgress struct {
	Strain     NamedStrain
	Conditions []ConditionProgress
	Met        bool
	Breedable  bool
}

// StrainProgressFor lists every strain a species can become, with each
// requirement measured against the plant.
//
// Like IdentifyStrain it walks StrainCatalogOrder rather than the catalog map,
// so the list holds still between frames.
func StrainProgressFor(kind CropKind, g plant.Genome, p plant.Phenotype) []StrainProgress {
	var out []StrainProgress
	for _, id := range StrainCatalogOrder {
		strain, ok := StrainCatalog[id]
		if !ok || (strain.Kind != "" && strain.Kind != kind) {
			continue
		}

		progress := StrainProgress{
			Strain:    strain,
			Met:       strain.Matches(g, p),
			Breedable: strain.Breedable(),
		}
		for _, c := range strain.Conditions {
			progress.Conditions = append(progress.Conditions, ConditionProgress{
				Condition: c,
				Current:   p.Get(c.Gene),
				Met:       c.Holds(p),
			})
		}
		out = append(out, progress)
	}
	return out
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
