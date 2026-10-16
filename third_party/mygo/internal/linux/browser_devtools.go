//go:build linux && (amd64 || arm64)

package linux

import (
	"errors"

	"github.com/ebitengine/purego"
)

var (
	browserMenuCount                                                        func(menu ptr) uint32
	browserMenuItem                                                         func(menu ptr, position uint32) ptr
	browserMenuAction                                                       func(item ptr) int32
	browserMenuRemove                                                       func(menu, item ptr)
	cbBrowserInspectorClosed, cbBrowserInspectorKey, cbBrowserInspectorMenu ptr
)

func loadBrowserDevToolsAPI(lib ptr) {
	bind(lib, &browserMenuCount, "webkit_context_menu_get_n_items")
	bind(lib, &browserMenuItem, "webkit_context_menu_get_item_at_position")
	bind(lib, &browserMenuAction, "webkit_context_menu_item_get_stock_action")
	bind(lib, &browserMenuRemove, "webkit_context_menu_remove")
}

// Registered once with the other backend callbacks, never once per open/tab.
func initBrowserDevToolsCallbacks() {
	cbBrowserInspectorClosed = purego.NewCallback(func(inspector, data ptr) {
		if w := theBackend.window(data); w != nil {
			w.browserInspectorClosed(inspector)
		}
	})
	cbBrowserInspectorKey = purego.NewCallback(func(widget, event, data ptr) bool {
		w := theBackend.window(data)
		if w == nil || !w.opts.BrowserProfile {
			return false
		}
		return browserInspectorShortcut(field[uint32](event, 28), field[uint32](event, 24))
	})
	cbBrowserInspectorMenu = purego.NewCallback(func(web, menu, event, hit, data ptr) bool {
		w := theBackend.window(data)
		if w == nil || !w.opts.BrowserProfile {
			return false
		}
		filterBrowserInspectorMenu(menu)
		return false // preserve the normal menu, minus Inspect Element
	})
}

func browserInspectorShortcut(key, state uint32) bool {
	if key == 0xffc9 {
		return true
	} // GDK_KEY_F12
	const ctrlShift = 1<<2 | 1<<0
	if state&ctrlShift != ctrlShift {
		return false
	}
	return key == 'i' || key == 'I' || key == 'j' || key == 'J' || key == 'c' || key == 'C'
}
func filterBrowserInspectorMenu(menu ptr) {
	if browserMenuCount == nil || browserMenuItem == nil || browserMenuAction == nil || browserMenuRemove == nil {
		return
	}
	// Iterate backwards since removal shifts subsequent positions.
	for n := browserMenuCount(menu); n > 0; n-- {
		item := browserMenuItem(menu, n-1)
		if item != 0 && browserMenuAction(item) == 31 { // WEBKIT_CONTEXT_MENU_ACTION_INSPECT_ELEMENT
			browserMenuRemove(menu, item)
		}
	}
}

func (w *window) BrowserDevToolsAvailable() bool {
	return w.browserReady() && browserMenuCount != nil && browserMenuItem != nil && browserMenuAction != nil && browserMenuRemove != nil
}

func (w *window) OpenBrowserDevTools() error {
	if !w.BrowserDevToolsAvailable() {
		return errors.New("mygo: Browser inspector requires a ready partition and WebKitGTK context-menu APIs")
	}
	if w.browserInspectorClosing {
		return errors.New("mygo: Browser inspector is still closing")
	}
	settings := webkitWebViewGetSettings(w.web)
	webkitSettingsSetEnableDeveloperExtras(settings, true)
	inspector := webkitWebViewGetInspector(w.web)
	if inspector == 0 {
		webkitSettingsSetEnableDeveloperExtras(settings, false)
		return errors.New("mygo: WebKitGTK did not provide a Browser inspector")
	}
	if w.browserInspector == 0 {
		w.browserInspector = inspector
		connect(inspector, "closed", cbBrowserInspectorClosed, ptr(w.id))
	}
	w.browserInspectorRequested = true
	webkitWebInspectorShow(inspector)
	// show() is asynchronous and has no success result. This acknowledges the
	// native request, not that a frontend has been displayed. No reload is forced.
	return nil
}

func (w *window) browserInspectorClosed(inspector ptr) {
	if inspector != w.browserInspector || w.closed {
		return
	}
	w.browserInspectorRequested, w.browserInspectorClosing = false, false
	webkitSettingsSetEnableDeveloperExtras(webkitWebViewGetSettings(w.web), false)
}
func (w *window) closeBrowserInspector() {
	if w.browserInspector == 0 || !w.browserInspectorRequested || w.browserInspectorClosing {
		return
	}
	w.browserInspectorClosing = true
	w.browserInspectorRequested = false
	webkitSettingsSetEnableDeveloperExtras(webkitWebViewGetSettings(w.web), false)
	webkitWebInspectorClose(w.browserInspector)
}
