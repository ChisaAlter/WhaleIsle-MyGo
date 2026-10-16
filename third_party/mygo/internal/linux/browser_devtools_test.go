//go:build linux && (amd64 || arm64)

package linux

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

func TestBrowserInspectorLifecycle(t *testing.T) {
	mockBrowserAPI(t)
	oldSettings, oldExtras, oldInspector := webkitWebViewGetSettings, webkitSettingsSetEnableDeveloperExtras, webkitWebViewGetInspector
	oldShow, oldClose, oldView := webkitWebInspectorShow, webkitWebInspectorClose, webkitWebInspectorGetWebView
	oldCount, oldItem, oldAction, oldRemove := browserMenuCount, browserMenuItem, browserMenuAction, browserMenuRemove
	t.Cleanup(func() {
		webkitWebViewGetSettings, webkitSettingsSetEnableDeveloperExtras, webkitWebViewGetInspector = oldSettings, oldExtras, oldInspector
		webkitWebInspectorShow, webkitWebInspectorClose, webkitWebInspectorGetWebView = oldShow, oldClose, oldView
		browserMenuCount, browserMenuItem, browserMenuAction, browserMenuRemove = oldCount, oldItem, oldAction, oldRemove
	})
	browserMenuCount = func(ptr) uint32 { return 0 }
	browserMenuItem = func(ptr, uint32) ptr { return 0 }
	browserMenuAction = func(ptr) int32 { return 0 }
	browserMenuRemove = func(ptr, ptr) {}
	b := &Backend{name: "Inspector fixture"}
	profile, err := b.acquireBrowserProfile(strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	w := &window{b: b, web: profile.context, browser: profile, opts: &platform.WindowOptions{BrowserProfile: true, BrowserPartition: profile.key}}
	enabled, shows, closes, signals := false, 0, 0, 0
	webkitWebViewGetSettings = func(ptr) ptr { return 7 }
	webkitSettingsSetEnableDeveloperExtras = func(s ptr, on bool) {
		if s != 7 {
			t.Fatal("wrong settings")
		}
		enabled = on
	}
	webkitWebViewGetInspector = func(ptr) ptr { return 8 }
	webkitWebInspectorShow = func(i ptr) {
		if i != 8 || !enabled {
			t.Fatal("inspector shown without scoped extras")
		}
		shows++
	}
	webkitWebInspectorClose = func(i ptr) {
		if i != 8 {
			t.Fatal("wrong inspector")
		}
		closes++
	}
	webkitWebInspectorGetWebView = func(ptr) ptr { return 9 }
	gSignalConnectData = func(_ ptr, _ *byte, _, _, _ ptr, _ int32) uint64 { signals++; return 1 }
	if !w.BrowserDevToolsAvailable() {
		t.Fatal("ready Browser inspector unavailable")
	}
	if err := w.OpenBrowserDevTools(); err != nil {
		t.Fatal(err)
	}
	if err := w.OpenBrowserDevTools(); err != nil {
		t.Fatal(err)
	}
	if shows != 2 || signals != 1 || w.opts.DevTools {
		t.Fatal("open changed policy or registered per-call signals")
	}
	w.CloseDevTools()
	if enabled || closes != 1 || !w.browserInspectorClosing {
		t.Fatal("close did not reset developer preference")
	}
	if w.OpenBrowserDevTools() == nil {
		t.Fatal("reopened during native close")
	}
	w.browserInspectorClosed(8)
	if err := w.OpenBrowserDevTools(); err != nil {
		t.Fatal(err)
	}
	w.browserInspectorClosed(8) // user closes the native inspector
	if enabled || w.browserInspectorRequested || w.browserInspectorClosing {
		t.Fatal("native close left developer extras enabled")
	}
	w.closed = true
	if w.BrowserDevToolsAvailable() || w.OpenBrowserDevTools() == nil {
		t.Fatal("closed Browser accepted inspector request")
	}
}

func TestBrowserInspectorShortcutPolicy(t *testing.T) {
	for _, key := range []uint32{0xffc9, 'I', 'i', 'J', 'j', 'C', 'c'} {
		state := uint32(0)
		if key != 0xffc9 {
			state = 1<<0 | 1<<2
		}
		if !browserInspectorShortcut(key, state) {
			t.Errorf("inspector shortcut escaped: %x", key)
		}
	}
	for _, pair := range [][2]uint32{{'c', 1 << 2}, {'i', 0}, {'j', 1 << 0}, {0xffc7, 0}, {'r', 1 << 2}} {
		if browserInspectorShortcut(pair[0], pair[1]) {
			t.Errorf("ordinary shortcut blocked: %v", pair)
		}
	}
}

func TestBrowserInspectorMenuKeepsOrdinaryActions(t *testing.T) {
	oldCount, oldItem, oldAction, oldRemove := browserMenuCount, browserMenuItem, browserMenuAction, browserMenuRemove
	t.Cleanup(func() {
		browserMenuCount, browserMenuItem, browserMenuAction, browserMenuRemove = oldCount, oldItem, oldAction, oldRemove
	})
	items := []ptr{14, 31, 16, 31}
	browserMenuCount = func(ptr) uint32 { return uint32(len(items)) }
	browserMenuItem = func(_ ptr, pos uint32) ptr { return items[pos] }
	browserMenuAction = func(item ptr) int32 { return int32(item) }
	browserMenuRemove = func(_ ptr, item ptr) {
		for i := len(items) - 1; i >= 0; i-- {
			if items[i] == item {
				items = append(items[:i], items[i+1:]...)
				return
			}
		}
	}
	filterBrowserInspectorMenu(1)
	if len(items) != 2 || items[0] != 14 || items[1] != 16 {
		t.Fatalf("ordinary menu changed: %v", items)
	}
}
