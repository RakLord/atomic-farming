package sim

// kindTestCrop is a crop kind that exists only for tests. Registering it here
// exercises the same registry path concrete crops will use.
const kindTestCrop CropKind = "test_crop"

type testCrop struct {
	// Watered is arbitrary per-crop config, present to prove that a crop's
	// own fields survive a save round-trip.
	Watered   bool `json:"watered,omitempty"`
	NumStages int  `json:"num_stages,omitempty"`
}

func (c *testCrop) Kind() CropKind { return kindTestCrop }

func (c *testCrop) Stages() int {
	if c.NumStages <= 0 {
		return 3
	}
	return c.NumStages
}

func (c *testCrop) Grow(ctx GrowContext, g Growth) Growth {
	g.Progress++
	return g
}

func init() { RegisterCrop(kindTestCrop, func() Crop { return &testCrop{} }) }
