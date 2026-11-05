package browserpreview

import (
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
	if called != 1 || tab.Error != "Renderer stopped" {
		t.Fatal(called, tab.Error)
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
