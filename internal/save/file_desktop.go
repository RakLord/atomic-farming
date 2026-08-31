//go:build !(js && wasm)

package save

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

func path(key string) string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "atomic-farming", key+".json")
}

func Write(key, value string) error {
	p := path(key)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("save: mkdir %s: %w", filepath.Dir(p), err)
	}
	if err := os.WriteFile(p, []byte(value), 0o644); err != nil {
		return fmt.Errorf("save: write %s: %w", p, err)
	}
	return nil
}

// Read returns the stored value, or (ok=false, err=nil) if no value has been
// stored for this key. A non-nil error indicates an I/O failure distinct from
// "not present".
func Read(key string) (string, bool, error) {
	b, err := os.ReadFile(path(key))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("save: read %s: %w", path(key), err)
	}
	return string(b), true, nil
}
