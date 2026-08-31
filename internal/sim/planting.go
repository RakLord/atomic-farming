package sim

import "atomicfarming/internal/plant"

// PlantSeed sows the seed at inventory index stackIndex into plot p.
//
// The plot's roll seed is stamped here and never changes again, so everything
// that later happens to this plant — how fast it grows, whether it survives,
// what it yields — is fixed the moment it goes in the ground and replays
// identically. See docs/adr/0010-determinism.md.
func (s *GameState) PlantSeed(p Position, stackIndex int) error {
	if s == nil {
		return ErrNoSeed
	}
	plot, ok := s.Grid.At(p)
	if !ok {
		return ErrNoSuchPlot
	}
	if plot.Crop != nil {
		return ErrPlotOccupied
	}
	if stackIndex < 0 || stackIndex >= len(s.Inventory.Stacks) {
		return ErrNoSeed
	}

	stack := s.Inventory.Stacks[stackIndex]
	crop, err := newCropByKind(stack.Kind)
	if err != nil {
		return err
	}
	if !s.Inventory.Take(stack.Kind, stack.Genome) {
		return ErrNoSeed
	}

	*plot = Plot{Crop: crop, Genome: stack.Genome, Seed: s.NextPlantSeed()}
	plot.Express()
	s.DiscoverPlant(stack.Kind, plot.Genome, plot.Phenotype)
	return nil
}

// Uproot clears a plot, discarding whatever is growing there.
func (s *GameState) Uproot(p Position) error {
	plot, ok := s.Grid.At(p)
	if !ok {
		return ErrNoSuchPlot
	}
	if plot.Crop == nil {
		return ErrNoPlant
	}
	*plot = Plot{}
	return nil
}

// PlotStrainName is the display name for what is growing at p.
func (s *GameState) PlotStrainName(p Position) string {
	plot, ok := s.Grid.At(p)
	if !ok || plot.Crop == nil {
		return ""
	}
	return StrainName(plot.Crop.Kind(), plot.Genome, plot.Phenotype, CropDisplayName(plot.Crop.Kind()))
}

// SeedPhenotype expresses a seed's genome through its species ranges — what
// the plant would be if sown.
func SeedPhenotype(stack SeedStack) plant.Phenotype {
	crop, err := newCropByKind(stack.Kind)
	if err != nil {
		return plant.Phenotype{}
	}
	return plant.Express(stack.Genome, crop.Ranges())
}

// SeedStrainName is the display name for a seed stack.
func SeedStrainName(stack SeedStack) string {
	if _, err := newCropByKind(stack.Kind); err != nil {
		return string(stack.Kind)
	}
	return StrainName(stack.Kind, stack.Genome, SeedPhenotype(stack), CropDisplayName(stack.Kind))
}

// cropNames maps a kind to its player-facing name. Crops register their own.
var cropNames = map[CropKind]string{}

// RegisterCropName gives a kind a display name, called from a crop's init().
func RegisterCropName(kind CropKind, name string) { cropNames[kind] = name }

// CropDisplayName is a species' player-facing name, falling back to its kind.
func CropDisplayName(kind CropKind) string {
	if n, ok := cropNames[kind]; ok {
		return n
	}
	return string(kind)
}
