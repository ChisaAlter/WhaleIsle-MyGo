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
	// Both panes keep an accessible restore glyph in their compact tab strip.
	for _, pane := range []*WorkspacePane{first, second} {
		local := m
		local.workspacePane = pane
		chrome := ui.NewTester(func(c *ui.Context) { workspacePaneControls(c, local) }, 400, 300)
		_, restore := chrome.Find(paneLabel("退出全屏", "Restore pane"))
		_, maximize := chrome.Find(paneLabel("面板全屏", "Maximize pane"))
		if !restore || maximize {
			t.Fatal("fullscreen sibling does not offer the shared restore action")
		}
	}
	ToggleWorkspacePaneFullscreen(m)
	tt.Frame()
	if w.Fullscreen != "" || w.Root != root || root.First != first || root.Second != second || root.Ratio != ratio {
		t.Fatal("restoring changed retained resources")
	}
}
