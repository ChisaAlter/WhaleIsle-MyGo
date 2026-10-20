package files

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

func tabStripFixture() (Model, string) {
	root := filepath.Join("workspace", "fixture")
	tabs := &Tabs{Workspaces: map[string]*WorkspaceTabs{}}
	tabs.Workspaces[filepath.Clean(root)] = &WorkspaceTabs{
		Open:   []*FileTab{{Path: "dir/alpha.go"}, {Path: "beta.md"}},
		Active: "beta.md",
	}
	selected := ""
	return Model{Tabs: tabs, WorkspaceRoot: root, SelectedFile: &selected}, selected
}

// TestTabStripRendersChipsNotGlyphState pins the strip's shape: every open tab is
// a chip carrying its file name, and the active/unsaved state no longer rides on
// "▸" and "•" characters glued to the label.
func TestTabStripRendersChipsNotGlyphState(t *testing.T) {
	m, _ := tabStripFixture()
	tt := ui.NewTester(func(c *ui.Context) {
		paintTabs(c, m, ui.RGB(24, 24, 24), ui.RGB(120, 120, 120))
	}, 720, 120)
	text := strings.Join(tt.Texts(), "\n")
	if !strings.Contains(text, "alpha.go") || !strings.Contains(text, "beta.md") {
		t.Fatalf("tab names missing from the strip: %s", text)
	}
	for _, glyph := range []string{"▸", "•"} {
		if strings.Contains(text, glyph) {
			t.Fatalf("strip still prints %q as state: %s", glyph, text)
		}
	}
}

// TestTabStripCloseOnlyOnTheActiveTab keeps the close control off inactive tabs
// until they are hovered, so a quiet strip is not a row of close boxes.
func TestTabStripCloseOnlyOnTheActiveTab(t *testing.T) {
	m, _ := tabStripFixture()
	tt := ui.NewTester(func(c *ui.Context) {
		paintTabs(c, m, ui.RGB(24, 24, 24), ui.RGB(120, 120, 120))
	}, 720, 120)
	closeLabel := i18n.T(i18n.KeySurfacesFilesClose)
	shown := 0
	for _, text := range tt.Texts() {
		if text == closeLabel {
			shown++
		}
	}
	if shown != 1 {
		t.Fatalf("expected one close control (active tab), got %d: %v", shown, tt.Texts())
	}
}
