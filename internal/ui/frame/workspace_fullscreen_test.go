package frame

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/store/session"
)

func TestWorkspaceFullscreenRetainsWholeDock(t *testing.T) {
	sizes := DefaultLayout()
	sizes.Bottom = 0
	tab := "files"
	m := Model{Store: session.New(), Sizes: &sizes, RightPaneOpen: true, RightPaneTab: &tab}
	prepareWorkspace(m)
	SplitWorkspacePane(m, false)
	w := sizes.Workspace
	root, first, second := w.Root, w.Root.First, w.Root.Second
	// Distinct kinds make sibling visibility observable without a real Host.
	first.Kind, second.Kind = "files", "diff"
	w.Active, w.route, tab = first.ID, "files", "files"
	ratio := root.Ratio
	ToggleWorkspacePaneFullscreen(m)
	if !w.visible("files") || !w.visible("diff") {
		t.Fatal("fullscreen hid a sibling resource")
	}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 1200, 800)
	tt.Frame()
	if w.Root != root || root.First != first || root.Second != second || root.Ratio != ratio {
		t.Fatal("fullscreen replaced the dock or its split geometry")
	}
	// Both panes must expose the global restore action, not leave the sibling
	// offering a second fullscreen action that hides the first resource.
	restores := 0
	for _, text := range tt.Texts() {
		if text == paneLabel("退出全屏", "Restore pane") {
			restores++
		}
	}
	if restores != 2 {
		t.Fatalf("restore controls=%d, want both resource panes", restores)
	}
	ToggleWorkspacePaneFullscreen(m)
	tt.Frame()
	if w.Fullscreen != "" || w.Root != root || root.First != first || root.Second != second || root.Ratio != ratio {
		t.Fatal("restoring changed retained resources")
	}
}
