package webview

import (
	"sync"

	"github.com/egoist/mygo"
)

// DevToolsShortcutAllowed mirrors reference/electron/src/main/devtools-shortcut.js:16-19:
// development builds always allow the chord; packaged builds only when the user
// turned on the openDevTools setting.
func DevToolsShortcutAllowed(packaged, openDevTools bool) bool {
	return !packaged || openDevTools
}

// IsDevToolsToggle matches reference/electron/src/main/devtools-shortcut.js:22-32:
// Ctrl+Shift+I (Windows/Linux) or Cmd+Alt+I (macOS), key press only, with no
// other modifier held.
func IsDevToolsToggle(in mygo.KeyInput) bool {
	if !in.Down || in.Key != "i" {
		return false
	}
	ctrlShift := in.Control && in.Shift && !in.Meta && !in.Alt
	metaAlt := in.Meta && in.Alt && !in.Control && !in.Shift
	return ctrlShift || metaAlt
}

// devToolsShortcutAllowed reports whether the chord may act in this process. The
// shell replaces it with its own openDevTools setting (SetDevToolsShortcutAllowed).
var devToolsShortcutAllowed = func() bool { return DevToolsShortcutAllowed(mygo.App.IsPackaged(), false) }

// SetDevToolsShortcutAllowed lets the shell apply its configuration; nil keeps
// the default (development always allowed, packaged builds never).
func SetDevToolsShortcutAllowed(fn func() bool) {
	if fn == nil {
		return
	}
	devToolsShortcutAllowed = fn
	// Pages that already exist must follow the setting too: their WebView2
	// AreDevToolsEnabled is what the built-in Ctrl+Shift+I accelerator checks.
	applyDevToolsGate()
}

// installDevToolsShortcut gives one page the window-scoped chord: only that
// window's devtools toggle, exactly as the native per-webContents listener does
// (devtools-shortcut.js:1-33). The menu item keeps working; this adds a chord.
func (h *Host) installDevToolsShortcut(pg *mygo.Page) {
	if h == nil || pg == nil || !devToolsShortcutAllowed() {
		return
	}
	pg.OnBeforeInput(func(in mygo.KeyInput) bool {
		if !IsDevToolsToggle(in) {
			return false
		}
		h.devToolsMu.Lock()
		open := h.devToolsOpen
		h.devToolsOpen = !open
		h.devToolsMu.Unlock()
		if open {
			pg.CloseDevTools()
		} else {
			pg.OpenDevTools()
		}
		return true
	})
}

// devToolsPage is the runtime inspector switch a page exposes.
type devToolsPage interface {
	SetDevToolsEnabled(enabled bool) bool
}

var (
	devToolsMu    sync.Mutex
	devToolsPages []devToolsPage
)

// registerDevToolsPage applies the current gate to pg and keeps it in the list
// so a later setting change reaches pages that already exist.
func registerDevToolsPage(pg devToolsPage) {
	if pg == nil {
		return
	}
	allowed := devToolsShortcutAllowed()
	pg.SetDevToolsEnabled(allowed)
	devToolsMu.Lock()
	devToolsPages = append(devToolsPages, pg)
	devToolsMu.Unlock()
}

// applyDevToolsGate pushes the current gate to every registered page.
func applyDevToolsGate() {
	allowed := devToolsShortcutAllowed()
	devToolsMu.Lock()
	pages := append([]devToolsPage(nil), devToolsPages...)
	devToolsMu.Unlock()
	for _, pg := range pages {
		pg.SetDevToolsEnabled(allowed)
	}
}
