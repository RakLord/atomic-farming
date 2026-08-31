package sim

import "testing"

func TestRegisterCropRejectsDuplicates(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("re-registering a crop kind did not panic; kinds are save identifiers and must be unique")
		}
	}()
	RegisterCrop(kindTestCrop, func() Crop { return &testCrop{} })
}

func TestRegisterCropRejectsEmptyKindAndNilFactory(t *testing.T) {
	for name, call := range map[string]func(){
		"empty kind":  func() { RegisterCrop("", func() Crop { return &testCrop{} }) },
		"nil factory": func() { RegisterCrop("never_registered", nil) },
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("RegisterCrop with %s did not panic", name)
				}
			}()
			call()
		})
	}
}

func TestNewCropByKindReconstructsRegisteredKinds(t *testing.T) {
	crop, err := newCropByKind(kindTestCrop)
	if err != nil {
		t.Fatalf("newCropByKind: %v", err)
	}
	if crop.Kind() != kindTestCrop {
		t.Errorf("Kind() = %q, want %q", crop.Kind(), kindTestCrop)
	}
	if _, err := newCropByKind("no_such_crop"); err == nil {
		t.Error("newCropByKind on an unknown kind returned no error")
	}
}

func TestRegisteredCropKindsIncludesTheTestCrop(t *testing.T) {
	for _, k := range RegisteredCropKinds() {
		if k == kindTestCrop {
			return
		}
	}
	t.Errorf("RegisteredCropKinds() = %v, missing %q", RegisteredCropKinds(), kindTestCrop)
}
