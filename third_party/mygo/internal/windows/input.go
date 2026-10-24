//go:build windows

package windows

import (
	"strings"

	"github.com/egoist/mygo/internal/platform"
)

// SetBeforeInput installs a key-input callback; see platform.SetBeforeInput.
func (w *window) SetBeforeInput(fn func(platform.KeyInput) bool) { w.beforeInput = fn }

func keyHeld(vk int) bool {
	r, _, _ := procGetKeyState.Call(uintptr(vk))
	return uint16(r)&0x8000 != 0
}

// keyName is the lower-case name of a virtual key, or "" when the app does not
// care about it: letters, digits and the punctuation WebView2 reports for them
// are enough for app shortcuts.
func keyName(vk uint32) string {
	switch {
	case vk >= 'A' && vk <= 'Z':
		return strings.ToLower(string(rune(vk)))
	case vk >= '0' && vk <= '9':
		return string(rune(vk))
	}
	return ""
}

// fireBeforeInput reports a key event to the app's callback; it reports whether
// the key was handled and must not reach the page.
func (w *window) fireBeforeInput(vk uint32, down bool) bool {
	if w.beforeInput == nil {
		return false
	}
	name := keyName(vk)
	if name == "" {
		return false
	}
	const (
		vkShift = 0x10
		vkCtrl  = 0x11
		vkAlt   = 0x12
		vkLWin  = 0x5B
	)
	return w.beforeInput(platform.KeyInput{
		Key:     name,
		Control: keyHeld(vkCtrl),
		Shift:   keyHeld(vkShift),
		Alt:     keyHeld(vkAlt),
		Meta:    keyHeld(vkLWin),
		Down:    down,
	})
}


