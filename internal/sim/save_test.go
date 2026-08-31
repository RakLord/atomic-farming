package sim

import (
	"encoding/json"
	"testing"

	"atomicfarming/internal/bignum"
	"atomicfarming/internal/save"
)

// isolateSave points the desktop save path at a temp dir so tests never touch
// the player's real config directory.
func isolateSave(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
}

func TestSaveLoadRoundTrip(t *testing.T) {
	isolateSave(t)

	s := NewGameState()
	s.Cash = bignum.MustParse("1.25e9")
	s.Ticks = 777
	s.RunCount = 3
	s.Unlocks["some_unlock"] = true
	s.Grid.Resize(4, 5)
	p, _ := s.Grid.At(Position{X: 2, Y: 3})
	p.Crop = &testCrop{Watered: true, NumStages: 6}
	p.Growth = Growth{Stage: 2, Progress: 11, Ready: true}

	if err := s.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if s.LastSaveUnix == 0 {
		t.Error("Save did not stamp LastSaveUnix")
	}

	got, ok, err := Load()
	if err != nil || !ok {
		t.Fatalf("Load: ok=%v err=%v", ok, err)
	}

	if !got.Cash.Eq(s.Cash) {
		t.Errorf("Cash = %s, want %s", got.Cash, s.Cash)
	}
	if got.Ticks != 777 || got.RunCount != 3 {
		t.Errorf("Ticks=%d RunCount=%d, want 777/3", got.Ticks, got.RunCount)
	}
	if !got.Unlocks["some_unlock"] {
		t.Error("unlock did not survive the round trip")
	}
	if got.Grid.W != 4 || got.Grid.H != 5 {
		t.Errorf("grid is %dx%d, want 4x5", got.Grid.W, got.Grid.H)
	}

	loaded, ok := got.Grid.At(Position{X: 2, Y: 3})
	if !ok {
		t.Fatal("planted plot missing after load")
	}
	crop, ok := loaded.Crop.(*testCrop)
	if !ok {
		t.Fatalf("crop reconstructed as %T, want *testCrop", loaded.Crop)
	}
	if !crop.Watered || crop.NumStages != 6 {
		t.Errorf("crop config lost: %+v", crop)
	}
	if loaded.Growth != (Growth{Stage: 2, Progress: 11, Ready: true}) {
		t.Errorf("Growth = %+v, want {2 11 true}", loaded.Growth)
	}

	// Every other plot must come back empty.
	for i, plot := range got.Grid.Plots {
		if i == 3*4+2 {
			continue
		}
		if !plot.IsEmpty() {
			t.Errorf("plot index %d is not empty after load", i)
		}
	}
}

func TestLoadWithNoSaveReturnsNotOK(t *testing.T) {
	isolateSave(t)
	s, ok, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if ok || s != nil {
		t.Errorf("Load with no save = (%v, %v), want (nil, false)", s, ok)
	}
}

func TestLoadRejectsUnknownVersion(t *testing.T) {
	isolateSave(t)
	writeRawSave(t, `{"version":999,"state":{}}`)

	if _, ok, err := Load(); err == nil || ok {
		t.Errorf("Load of a future save = (ok=%v, err=%v), want an error", ok, err)
	}
}

func TestLoadRejectsUnknownCropKind(t *testing.T) {
	isolateSave(t)
	writeRawSave(t, `{"version":1,"state":{"grid":{"w":1,"h":1,"plots":[{"kind":"crop_from_a_future_build","crop":{}}]}}}`)

	if _, ok, err := Load(); err == nil || ok {
		t.Errorf("Load of an unknown crop = (ok=%v, err=%v), want an error", ok, err)
	}
}

// TestLoadRepairsInvariants covers the defensive pass Load runs after
// unmarshal: a hand-edited or truncated save must not be able to panic the
// tick loop.
func TestLoadRepairsInvariants(t *testing.T) {
	isolateSave(t)
	writeRawSave(t, `{"version":1,"state":{"layer":"nonsense","tick_rate":0,"grid":{"w":2,"h":2,"plots":[{}]}}}`)

	got, ok, err := Load()
	if err != nil || !ok {
		t.Fatalf("Load: ok=%v err=%v", ok, err)
	}
	if got.Layer != LayerField {
		t.Errorf("Layer = %q, want %q", got.Layer, LayerField)
	}
	if got.TickRate != DefaultTickRate {
		t.Errorf("TickRate = %d, want %d", got.TickRate, DefaultTickRate)
	}
	// The save claims a 2x2 farm holding one plot. Both are repaired: the plot
	// slice is grown to match the dimensions, and the farm itself is brought up
	// to the base size, since nothing entitles a save to a smaller one.
	if got.Grid.W != DefaultGridW || got.Grid.H != DefaultGridH {
		t.Errorf("farm is %dx%d, want the base %dx%d", got.Grid.W, got.Grid.H, DefaultGridW, DefaultGridH)
	}
	if len(got.Grid.Plots) != got.Grid.W*got.Grid.H {
		t.Errorf("len(Plots) = %d, want %d — the truncated grid was not repaired",
			len(got.Grid.Plots), got.Grid.W*got.Grid.H)
	}
	if got.Unlocks == nil {
		t.Error("Unlocks is nil after load")
	}
}

// TestLoadDiscardsPersistedModifiers proves the derived-cache rule: whatever
// multipliers a save carries are thrown away and recomputed from the owned
// unlock set, so retuning an unlock applies to existing saves with no
// migration.
func TestLoadDiscardsPersistedModifiers(t *testing.T) {
	isolateSave(t)
	writeRawSave(t, `{"version":1,"state":{"modifiers":{"yield_mul":"9.9e9"},"unlocks":{}}}`)

	got, ok, err := Load()
	if err != nil || !ok {
		t.Fatalf("Load: ok=%v err=%v", ok, err)
	}
	if !got.Modifiers.YieldMul.IsZero() {
		t.Errorf("YieldMul = %s, want zero — a stale cached multiplier was trusted", got.Modifiers.YieldMul)
	}
}

func TestEmptyPlotMarshalsWithoutCropFields(t *testing.T) {
	blob, err := json.Marshal(Plot{})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(blob) != "{}" {
		t.Errorf("empty plot marshaled as %s, want {}", blob)
	}
}

// writeRawSave plants a hand-written save blob so tests can exercise Load's
// handling of saves this build did not produce.
func writeRawSave(t *testing.T, blob string) {
	t.Helper()
	if err := save.Write(saveKey, blob); err != nil {
		t.Fatalf("writing raw save: %v", err)
	}
}
