package sim

import (
	"encoding/json"
	"fmt"
	"time"

	"atomicfarming/internal/plant"
	"atomicfarming/internal/save"
)

const (
	saveKey = "state"
	// currentVersion is the save envelope version. Bump it for any
	// non-additive change to the shape of GameState; purely additive fields
	// with acceptable zero values do not need one.
	// See docs/adr/0002-versioned-save-envelope.md.
	currentVersion = 1
)

type saveEnvelope struct {
	Version int             `json:"version"`
	State   json.RawMessage `json:"state"`
}

// plotJSON is the on-disk shape of a Plot. The crop's kind is stored
// alongside its own JSON so the registry can reconstruct the concrete type.
type plotJSON struct {
	Kind   CropKind        `json:"kind,omitempty"`
	Crop   json.RawMessage `json:"crop,omitempty"`
	Growth *Growth         `json:"growth,omitempty"`
	Genome *plant.Genome   `json:"genome,omitempty"`
	Seed   uint64          `json:"seed,omitempty"`
}

// MarshalJSON writes an empty plot as {} and a planted one as its kind plus
// the crop's own encoding.
func (p Plot) MarshalJSON() ([]byte, error) {
	var out plotJSON
	if p.Crop != nil {
		inner, err := json.Marshal(p.Crop)
		if err != nil {
			return nil, err
		}
		out.Kind = p.Crop.Kind()
		out.Crop = inner
		growth := p.Growth
		out.Growth = &growth
		genome := p.Genome
		out.Genome = &genome
		out.Seed = p.Seed
	}
	return json.Marshal(out)
}

func (p *Plot) UnmarshalJSON(data []byte) error {
	var in plotJSON
	if err := json.Unmarshal(data, &in); err != nil {
		return err
	}
	if in.Kind == "" || len(in.Crop) == 0 {
		*p = Plot{}
		return nil
	}
	crop, err := newCropByKind(in.Kind)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(in.Crop, crop); err != nil {
		return err
	}
	p.Crop = crop
	p.Growth = Growth{}
	if in.Growth != nil {
		p.Growth = *in.Growth
	}
	p.Genome = plant.Genome{}
	if in.Genome != nil {
		p.Genome = *in.Genome
	}
	p.Seed = in.Seed
	return nil
}

// Save stamps the save time, serializes the state into the versioned
// envelope, and writes it through internal/save.
func (s *GameState) Save() error {
	s.LastSaveUnix = time.Now().Unix()
	body, err := json.Marshal(s)
	if err != nil {
		return err
	}
	blob, err := json.Marshal(saveEnvelope{Version: currentVersion, State: body})
	if err != nil {
		return err
	}
	return save.Write(saveKey, string(blob))
}

// Load reads the versioned envelope and returns the restored state. When no
// save exists it returns (nil, false, nil). An unknown version is an error,
// so the caller can decide to boot fresh rather than guess at a migration.
func Load() (*GameState, bool, error) {
	raw, ok, err := save.Read(saveKey)
	if err != nil {
		return nil, false, err
	}
	if !ok || raw == "" {
		return nil, false, nil
	}
	var env saveEnvelope
	if err := json.Unmarshal([]byte(raw), &env); err != nil {
		return nil, false, err
	}
	if env.Version != currentVersion {
		return nil, false, fmt.Errorf("sim: unsupported save version %d", env.Version)
	}
	state := NewGameState()
	if err := json.Unmarshal(env.State, state); err != nil {
		return nil, false, err
	}
	repairState(state)
	return state, true, nil
}

// repairState brings a freshly unmarshaled state back to its invariants:
// non-nil maps and grid, a valid layer, a sane tick rate, and a modifier
// cache derived from the authoritative unlock set.
func repairState(s *GameState) {
	if s == nil {
		return
	}
	if s.Grid == nil {
		s.Grid = NewGrid(DefaultGridW, DefaultGridH)
	}
	s.Grid.normalize()
	if s.Unlocks == nil {
		s.Unlocks = map[UnlockID]bool{}
	}
	if !s.Layer.Valid() {
		s.Layer = LayerField
	}
	if s.TickRate <= 0 {
		s.TickRate = DefaultTickRate
	}
	if s.WorldSeed == 0 {
		s.WorldSeed = plant.Hash64(uint64(s.Ticks) + 1)
	}
	if s.DiscoveredStrains == nil {
		s.DiscoveredStrains = map[StrainID]bool{}
	}
	s.Inventory.prune()
	// A crop with no genome predates the genome layer, or came from a
	// hand-edited save. Give it the catalog default rather than the all-zero
	// genome, which expresses as a plant with no stem at all.
	//
	// Every planted plot then has its phenotype rebuilt: it is a derived cache
	// that is never persisted, so it is empty until recomputed here.
	for i := range s.Grid.Plots {
		plot := &s.Grid.Plots[i]
		if plot.Crop == nil {
			continue
		}
		if plot.Genome.IsZero() {
			plot.Genome = plant.DefaultGenome()
		}
		plot.Express()
	}
	// Unlocks is the source of truth; whatever Modifiers the save carried is
	// discarded so retuned unlock effects apply on load.
	rebuildModifiers(s)
}
