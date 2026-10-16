//go:build windows && (amd64 || arm64)

package windows

import (
	"github.com/egoist/mygo/internal/accelerator"
	"unsafe"
)

func (w *window) reportWebApplicationKey(args uintptr, vk uint32) bool {
	receiver, ok := w.h.(interface {
		WebApplicationKey(uint32, bool, bool, bool, bool, bool)
	})
	if !ok {
		return false
	}
	var lparam int32
	// ICoreWebView2AcceleratorKeyPressedEventArgs.get_KeyEventLParam.
	if failed(comCall(args, 5, uintptr(unsafe.Pointer(&lparam)))) {
		return false
	}
	mods := currentModifiers()
	// Right Alt + Control is AltGr on Windows layouts; preserve composition.
	rightAlt, _, _ := procGetKeyState.Call(0xA5)
	if mods&accelerator.Ctrl != 0 && int16(rightAlt) < 0 {
		return false
	}
	if priority, ok := w.h.(interface {
		ProductWebKey(uint32, bool, bool, bool, bool, bool) bool
	}); ok {
		if priority.ProductWebKey(vk, mods&accelerator.Ctrl != 0, mods&accelerator.Alt != 0, mods&accelerator.Shift != 0, mods&accelerator.Super != 0, uint32(lparam)&(1<<30) != 0) {
			return true
		}
	}
	receiver.WebApplicationKey(vk, mods&accelerator.Ctrl != 0, mods&accelerator.Alt != 0, mods&accelerator.Shift != 0, mods&accelerator.Super != 0, uint32(lparam)&(1<<30) != 0)
	return false
}
