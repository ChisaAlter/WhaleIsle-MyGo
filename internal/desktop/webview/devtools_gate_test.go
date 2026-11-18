package webview

import "testing"

type stubDevToolsPage struct {
	value *bool
	calls int
}

func (s *stubDevToolsPage) SetDevToolsEnabled(enabled bool) bool {
	s.calls++
	v := enabled
	s.value = &v
	return true
}

// TestDevToolsGateReachesPages pins the S03 fix: the gate is applied to a page
// when it is registered and again to every existing page when the setting
// changes, so a denied page cannot open the inspector through WebView2's own
// Ctrl+Shift+I accelerator.
func TestDevToolsGateReachesPages(t *testing.T) {
	previous := devToolsShortcutAllowed
	devToolsMu.Lock()
	devToolsPages = nil
	devToolsMu.Unlock()
	t.Cleanup(func() {
		devToolsShortcutAllowed = previous
		devToolsMu.Lock()
		devToolsPages = nil
		devToolsMu.Unlock()
	})

	allowed := true
	devToolsShortcutAllowed = func() bool { return allowed }
	page := &stubDevToolsPage{}
	registerDevToolsPage(page)
	if page.value == nil || !*page.value || page.calls != 1 {
		t.Fatalf("registration did not enable the page: value=%v calls=%d", page.value, page.calls)
	}

	// Turning the gate off must reach the page that already exists.
	allowed = false
	applyDevToolsGate()
	if page.value == nil || *page.value {
		t.Fatalf("gate change did not disable the page: %v", page.value)
	}
	if page.calls != 2 {
		t.Fatalf("calls = %d, want 2", page.calls)
	}

	// A page created while denied starts disabled.
	second := &stubDevToolsPage{}
	registerDevToolsPage(second)
	if second.value == nil || *second.value {
		t.Fatalf("a page registered while denied was enabled: %v", second.value)
	}

	// Setting the gate through the public setter pushes it to both pages.
	allowed = true
	SetDevToolsShortcutAllowed(func() bool { return true })
	for i, p := range []*stubDevToolsPage{page, second} {
		if p.value == nil || !*p.value {
			t.Fatalf("page %d not re-enabled: %v", i, p.value)
		}
	}
	registerDevToolsPage(nil) // must not panic or add a page
	registerDevToolsPage(page)
	registerDevToolsPage(second)
	if len(devToolsPages) != 2 {
		t.Fatalf("pages after duplicate registration = %d, want 2", len(devToolsPages))
	}

	unregisterDevToolsPage(page)
	unregisterDevToolsPage(nil)
	if len(devToolsPages) != 1 {
		t.Fatalf("pages after unregister = %d, want 1", len(devToolsPages))
	}
	callsBefore := page.calls
	SetDevToolsShortcutAllowed(func() bool { return false })
	if page.calls != callsBefore {
		t.Fatalf("unregistered page still received gate update: calls %d -> %d", callsBefore, page.calls)
	}
	if second.value == nil || *second.value {
		t.Fatalf("remaining page did not receive gate update: %v", second.value)
	}
}
