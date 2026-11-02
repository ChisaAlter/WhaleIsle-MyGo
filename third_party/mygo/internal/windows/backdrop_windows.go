//go:build windows && (amd64 || arm64)

package windows

import (
	"log"
	"unsafe"

	"github.com/egoist/mygo/internal/platform"
)

var (
	procDwmIsCompositionEnabled = dwmapi.NewProc("DwmIsCompositionEnabled")
	procDwmGetWindowAttribute   = dwmapi.NewProc("DwmGetWindowAttribute")
)

// applyBackdrop uses the documented DWM system-backdrop API. It never claims a
// material when the call failed, composition is disabled, high contrast is on,
// or the window's redirection bitmap would hide it. Existing rounded native
// windows already use DirectComposition and can switch material in place.
func (w *window) applyBackdrop(material string) bool {
	w.materialApplied = false
	if !systemBackdrops() || !w.noRedirect {
		return false
	}
	var enabled int32
	if hr, _, _ := procDwmIsCompositionEnabled.Call(uintptr(unsafe.Pointer(&enabled))); failed(hr) || enabled == 0 {
		return false
	}
	backdrop := backdropFor(material)
	if material != "" && (theme{w.b}).Preferences().HighContrast {
		backdrop = backdropNone
	}
	m := margins{}
	if backdrop != backdropNone {
		m = margins{-1, -1, -1, -1}
	}
	if hr, _, _ := procDwmExtendFrameIntoClientArea.Call(w.hwnd, uintptr(unsafe.Pointer(&m))); failed(hr) {
		log.Print(hresultError("extending system backdrop frame", hr))
		return false
	}
	if hr, _, _ := procDwmSetWindowAttribute.Call(w.hwnd, dwmwaSystemBackdropType, uintptr(unsafe.Pointer(&backdrop)), 4); failed(hr) {
		log.Print(hresultError("setting system backdrop", hr))
		return false
	}
	var actual int32
	if hr, _, _ := procDwmGetWindowAttribute.Call(w.hwnd, dwmwaSystemBackdropType, uintptr(unsafe.Pointer(&actual)), 4); failed(hr) || actual != backdrop {
		log.Printf("mygo: system backdrop did not apply (requested=%d actual=%d HRESULT=%#08x)", backdrop, actual, uint32(hr))
		return false
	}
	w.materialApplied = actual != backdropNone
	return w.materialApplied
}

// Backdrop state is observable by native UI through ShowsMaterial; this small
// helper keeps the accessibility preference gate independent from the renderer.
var _ platform.MaterialSurface = (*surface)(nil)
