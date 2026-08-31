package sim

// Layer identifies one rung of the prestige ladder. Ascending out of a layer
// resets the state that layer owns (see reset.go) while durable progression
// carries forward.
//
// Layers beyond the base are deliberately unnamed until their gameplay
// exists. See docs/adr/0007-layer-model-and-reset-registry.md.
type Layer string

// LayerField is the base layer: the farm itself.
const LayerField Layer = "field"

// LayerOrder lists every registered layer, innermost first. The reset-rule
// coverage test walks this, so a new Layer constant must be added here.
var LayerOrder = []Layer{LayerField}

// Valid reports whether l is a known layer.
func (l Layer) Valid() bool {
	for _, known := range LayerOrder {
		if l == known {
			return true
		}
	}
	return false
}
