// Package crops holds every concrete Crop implementation.
//
// Each crop lives in one file, owns its sim.CropKind constant, and registers
// itself with sim from init(). The game binary blank-imports this package so
// those registrations run before a save is loaded — without them, sim.Load
// cannot reconstruct planted crops.
//
// Adding a crop touches nothing outside this directory: no edits to the tick
// loop, the save layer, or the registry.
//
// The package is deliberately empty in the scaffold; Phase 1 adds the first
// crops.
package crops
