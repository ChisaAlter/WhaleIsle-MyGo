//go:build linux && (amd64 || arm64)

package linux

import (
	"github.com/egoist/mygo/internal/platform"
	"strings"
	"testing"
)

func TestBrowserProfileCapabilitiesFailClosed(t *testing.T) {
	var native any = &window{}
	tools, ok := native.(interface {
		BrowserToolsAvailable() bool
		BrowserCookieToolsAvailable() bool
		BrowserSiteDataToolsAvailable() bool
	})
	if !ok {
		t.Fatal("Linux Browser profile controls are not implemented")
	}
	if tools.BrowserToolsAvailable() || tools.BrowserCookieToolsAvailable() || tools.BrowserSiteDataToolsAvailable() {
		t.Fatal("an uninitialized or ordinary window must not advertise Browser controls")
	}
}

// Native entry points are replaced only inside these tests. No WebKit library,
// browser directory, live Cookie store or GUI is touched.
func mockBrowserAPI(t *testing.T) {
	t.Helper()
	oldNew, oldContext, oldType, oldObject := browserManagerNew, browserContextNew, browserWebViewType, browserObjectNew
	oldGet, oldCookies, oldStorage := browserGetContext, browserGetCookieManager, browserCookieStorage
	oldMajor, oldMinor := browserMajorVersion, browserMinorVersion
	oldRef, oldUnref, oldSignal := gObjectRef, gObjectUnref, gSignalConnectData
	oldClear, oldFinish := webkitWebsiteDataManagerClear, webkitWebsiteDataManagerClearFinish
	t.Cleanup(func() {
		browserManagerNew, browserContextNew, browserWebViewType, browserObjectNew = oldNew, oldContext, oldType, oldObject
		browserGetContext, browserGetCookieManager, browserCookieStorage = oldGet, oldCookies, oldStorage
		browserMajorVersion, browserMinorVersion = oldMajor, oldMinor
		gObjectRef, gObjectUnref, gSignalConnectData = oldRef, oldUnref, oldSignal
		webkitWebsiteDataManagerClear, webkitWebsiteDataManagerClearFinish = oldClear, oldFinish
	})
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var serial ptr = 100
	browserManagerNew = func(_, _, _, _ *byte, _ ptr) ptr { serial++; return serial }
	browserContextNew = func(m ptr) ptr { return m + 1000 }
	browserWebViewType = func() ptr { return 1 }
	browserObjectNew = func(_ ptr, _ *byte, _ ptr, _ *byte, _ ptr, _ ptr) ptr { return 1 }
	browserGetContext = func(web ptr) ptr { return web }
	browserGetCookieManager = func(m ptr) ptr { return m + 2000 }
	browserCookieStorage = func(_ ptr, _ *byte, _ int32) {}
	browserMajorVersion = func() uint32 { return 2 }
	browserMinorVersion = func() uint32 { return 30 }
	gObjectRef = func(p ptr) ptr { return p }
	gObjectUnref = func(ptr) {}
	gSignalConnectData = func(_ ptr, _ *byte, _, _, _ ptr, _ int32) uint64 { return 1 }
}

func TestBrowserProfilesShareOnlyExactPartition(t *testing.T) {
	mockBrowserAPI(t)
	b := &Backend{name: "Browser fixture"}
	a, err := b.acquireBrowserProfile(strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	again, err := b.acquireBrowserProfile(a.key)
	if err != nil || a != again || a.refs != 2 {
		t.Fatalf("same partition did not share its context: %v", err)
	}
	other, err := b.acquireBrowserProfile(strings.Repeat("b", 64))
	if err != nil || other.context == a.context || other.manager == a.manager {
		t.Fatalf("different partitions share native data: %v", err)
	}
	w := &window{b: b, browser: a, web: a.context, opts: &platform.WindowOptions{BrowserProfile: true, BrowserPartition: a.key}}
	if !w.BrowserToolsAvailable() || !w.BrowserSiteDataToolsAvailable() || w.BrowserColorSchemeAvailable() {
		t.Fatal("incorrect profile capabilities")
	}
	w.web = other.context
	if w.BrowserToolsAvailable() {
		t.Fatal("mismatched native context was accepted")
	}
	w.releaseBrowserProfile()
	if a.refs != 1 || b.browserProfiles[a.key] != a {
		t.Fatal("closing one tab released another tab's context")
	}
	(&window{b: b, browser: again}).releaseBrowserProfile()
	if b.browserProfiles[a.key] != nil || b.browserProfiles[other.key] != other {
		t.Fatal("last-tab release affected the wrong partition")
	}
}

func TestBrowserProfilesRejectMissingAPIsAndInvalidKeys(t *testing.T) {
	mockBrowserAPI(t)
	b := &Backend{}
	if _, err := b.acquireBrowserProfile("../default"); err == nil {
		t.Fatal("invalid partition accepted")
	}
	browserMinorVersion = func() uint32 { return 15 }
	if _, err := b.acquireBrowserProfile(strings.Repeat("a", 64)); err == nil {
		t.Fatal("old runtime accepted")
	}
	browserMinorVersion = func() uint32 { return 30 }
	browserContextNew = func(ptr) ptr { return 0 }
	if _, err := b.acquireBrowserProfile(strings.Repeat("a", 64)); err == nil || len(b.browserProfiles) != 0 {
		t.Fatal("failed creation fell back or registered a profile")
	}
}

func TestBrowserClearUsesExactManagerAndTypes(t *testing.T) {
	mockBrowserAPI(t)
	b := &Backend{name: "Clear fixture"}
	p, err := b.acquireBrowserProfile(strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	w := &window{b: b, browser: p, web: p.context, opts: &platform.WindowOptions{BrowserProfile: true, BrowserPartition: p.key}}
	for _, tc := range []struct {
		name  string
		run   func(func(error))
		types uint32
	}{
		{"cache", w.ClearBrowserHTTPCache, 3},
		{"cookies", w.ClearBrowserLoginCookies, 256},
		{"site", w.ClearBrowserSiteData, 16 | 32 | 64 | 256 | 4096},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var request ptr
			refs, unrefs, completions := 0, 0, 0
			gObjectRef = func(m ptr) ptr {
				if m != p.manager {
					t.Fatal("wrong retained manager")
				}
				refs++
				return m
			}
			gObjectUnref = func(m ptr) {
				if m != p.manager {
					t.Fatal("wrong released manager")
				}
				unrefs++
			}
			webkitWebsiteDataManagerClear = func(m ptr, types uint32, span int64, cancel, cb, data ptr) {
				if m != p.manager || types != tc.types || span != 0 || cancel != 0 {
					t.Fatal("clear escaped requested manager/type scope")
				}
				request = data
			}
			webkitWebsiteDataManagerClearFinish = func(m, result ptr, err *ptr) bool { return m == p.manager && result == 9 }
			tc.run(func(err error) {
				if err != nil {
					t.Error(err)
				}
				completions++
			})
			if request == 0 || completions != 0 || refs != 1 || unrefs != 0 {
				t.Fatal("operation did not wait for native completion")
			}
			pending.take(request)(p.manager, 9)
			if completions != 1 || unrefs != 1 {
				t.Fatal("native completion leaked a reference or callback")
			}
		})
	}
	browserMinorVersion = func() uint32 { return 28 }
	if w.BrowserSiteDataToolsAvailable() {
		t.Fatal("service-worker clearing advertised before WebKitGTK 2.30")
	}
	called := false
	w.ClearBrowserSiteData(func(err error) { called = err != nil })
	if !called {
		t.Fatal("unsupported site-data operation did not fail")
	}
}
