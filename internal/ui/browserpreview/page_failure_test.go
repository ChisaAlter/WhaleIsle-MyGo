package browserpreview

import (
	"strings"
	"testing"

	desktop "whaleisle/internal/desktop/webview"
)

func TestPageFailureBelongsToExistingTabAndNavigation(t *testing.T) {
	tab := &Tab{ID: 1}
	b := &Controller{Tabs: []*Tab{tab}}
	current := true
	called := 0
	b.SetPageFailureHandler(func(desktop.PageFailure) { called++ })
	failure := desktop.PageFailure{Kind: "render-process-gone", Message: "Renderer stopped", Current: func() bool { return current }}
	b.pageFailure(tab)(failure)
	if called != 1 || !strings.Contains(tab.Error, "重新载入") {
		t.Fatalf("page failure line does not name the next step: called=%d error=%q", called, tab.Error)
	}
	if strings.Contains(tab.Error, "Renderer stopped") {
		t.Fatalf("page failure tab carries the runtime's raw text: %q", tab.Error)
	}
	current = false
	b.pageFailure(tab)(failure)
	current = true
	b.Tabs = nil
	b.pageFailure(tab)(failure)
	if called != 1 {
		t.Fatal("stale navigation/tab failure reached shell recovery", called)
	}
}
