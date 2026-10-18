package frame

import (
	"encoding/json"
	"testing"
	"whaleisle/internal/ui/browserpreview"
)

// State-only fixture: no renderer, native surfaces, Host connection or PTY.
func TestWorkspaceDockTabsState(t *testing.T) {
	w := &WorkspacePanes{}
	w.syncRoute("files")
	files := w.Root
	changes := w.selectKind(files, "diff")
	changes.Diff = &WorkspaceDiffState{Path: "retained.go"}
	browser := w.selectKind(changes, "browser")
	if w.Root != browser || w.Active != browser.ID {
		t.Fatal("new resource did not become active")
	}
	assertOrder := func(active *WorkspacePane, want ...string) {
		t.Helper()
		tabs := active.dockTabs()
		if len(tabs) != len(want) {
			t.Fatalf("tab count = %d; want %d", len(tabs), len(want))
		}
		for i, tab := range tabs {
			if tab.ID != want[i] {
				t.Fatalf("tab %d = %s; want %s", i, tab.ID, want[i])
			}
		}
	}
	assertOrder(w.Root, files.ID, changes.ID, browser.ID)
	if w.visible("files") || w.visible("diff") || !w.visible("browser") {
		t.Fatal("inactive tabs are reported visible")
	}
	restored, err := RestoreWorkspacePanes(w.Export())
	if err != nil {
		t.Fatal(err)
	}
	assertOrder(restored.Root, files.ID, changes.ID, browser.ID)
	if restored.Active != browser.ID || restored.Root.DockIndex != 2 || restored.Root.find(changes.ID).Diff.Path != "retained.go" {
		t.Fatal("active tab or inactive resource state was not restored")
	}
	// Controller identity and terminal source identity survive both sorts of move.
	controller := &browserpreview.Controller{}
	browser.browser = controller
	if !w.MoveWorkspaceTab(browser.ID, files.ID, 0) {
		t.Fatal("same-pane reorder rejected")
	}
	assertOrder(w.Root, browser.ID, files.ID, changes.ID)
	if w.Root != browser || browser.browser != controller {
		t.Fatal("reorder replaced resource occurrence")
	}
	terminal := w.leaf("host-terminal")
	terminal.TerminalSource = "existing-host-occurrence"
	w.Root = &WorkspacePane{ID: w.leaf("").ID, Axis: "horizontal", Ratio: 0.37, First: browser, Second: terminal}
	if !w.MoveWorkspaceTab(browser.ID, terminal.ID, 1) {
		t.Fatal("cross-pane move rejected")
	}
	if w.Root.First != files || w.Root.Second != browser || w.Root.Ratio != 0.37 {
		t.Fatal("move changed split geometry or selected the wrong resource")
	}
	assertOrder(w.Root.First, files.ID, changes.ID)
	assertOrder(w.Root.Second, terminal.ID, browser.ID)
	if browser.browser != controller || w.Root.find(browser.ID) != browser || terminal.TerminalSource != "existing-host-occurrence" {
		t.Fatal("cross-pane move cloned resource state")
	}
	if !w.MoveWorkspaceTab(files.ID, browser.ID, 2) || !w.MoveWorkspaceTab(changes.ID, browser.ID, 3) {
		t.Fatal("remaining source tabs did not move")
	}
	if w.Root.First != nil || w.Root != changes {
		t.Fatal("empty source split did not collapse")
	}
	assertOrder(w.Root, terminal.ID, browser.ID, files.ID, changes.ID)
	if w.Root.find(browser.ID) != browser || browser.browser != controller || w.Active != changes.ID {
		t.Fatal("collapse lost resource identity or active tab")
	}
	legacy := json.RawMessage(`{"root":{"id":"workspace-pane-1","kind":"files"},"active":"workspace-pane-1","next":1}`)
	old, err := RestoreWorkspacePanes(legacy)
	if err != nil {
		t.Fatal(err)
	}
	assertOrder(old.Root, "workspace-pane-1")
	if old.Root.Kind != "files" || old.Active != old.Root.ID {
		t.Fatal("legacy single leaf did not restore")
	}
}
