//go:build js && wasm

package save

import (
	"fmt"
	"syscall/js"
)

var storage = js.Global().Get("localStorage")

// Write stores value under key. LocalStorage.setItem throws (as a JS
// exception) on quota exhaustion or when storage is disabled; syscall/js
// turns those into Go panics, which we convert into errors so callers can
// tell a save actually happened.
func Write(key, value string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("save: localStorage.setItem(%q) threw: %v", key, r)
		}
	}()
	storage.Call("setItem", key, value)
	return nil
}

// Read returns the stored value. (ok=false, err=nil) means the key is not
// set. A non-nil error indicates LocalStorage itself failed (e.g. disabled
// by browser settings).
func Read(key string) (_ string, _ bool, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("save: localStorage.getItem(%q) threw: %v", key, r)
		}
	}()
	v := storage.Call("getItem", key)
	if v.IsNull() || v.IsUndefined() {
		return "", false, nil
	}
	return v.String(), true, nil
}
