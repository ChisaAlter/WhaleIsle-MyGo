//go:build linux && (amd64 || arm64)

package linux

import "github.com/ebitengine/purego"

var (
	trayIndicatorGetType func() uintptr
	traySignalLookup     func(name *byte, typ uintptr) uint32
	cbTrayActivate       ptr
)

// Ayatana's public "activate" signal forwards StatusNotifierItem.Activate.
// Probe the signal rather than assuming a library version: distributions may
// backport it, while the older libappindicator ABI has no primary-click signal.
func initTrayActivation() {
	if libIndicator == 0 {
		return
	}
	bind(libIndicator, &trayIndicatorGetType, "app_indicator_get_type")
	bind(libGObject, &traySignalLookup, "g_signal_lookup")
	if cbTrayActivate == 0 {
		cbTrayActivate = purego.NewCallback(func(ind ptr, x, y int32, data ptr) {
			dispatchTrayActivation(ind, data)
		})
	}
}

func (t *tray) connectActivation() {
	// NewTray has already constructed the object, so its class and signals
	// have been initialized before g_signal_lookup is called.
	if trayIndicatorGetType != nil && traySignalLookup != nil &&
		traySignalLookup(cs("activate"), trayIndicatorGetType()) != 0 {
		connect(t.ind, "activate", cbTrayActivate, ptr(t.id))
	}
}

func dispatchTrayActivation(ind, data ptr) {
	// Native callbacks run on the GTK main thread. The ID table is also the
	// lifetime boundary: callbacks for an already removed tray are ignored.
	if t := theBackend.trays[int(data)]; t != nil && t.ind == ind && t.h != nil {
		t.h.Clicked()
	}
}
