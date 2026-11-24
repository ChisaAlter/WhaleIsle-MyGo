package files

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// Toggling the auto-refresh checkbox must update the workspace tab state.
// The bound stack local used to be compared before applyInputs flushed the
// click, so the toggle never landed.
func TestAutoRefreshCheckboxAppliesToggle(t *testing.T) {
	root := t.TempDir()
	m := Model{
		BindingID:     "fixture",
		WorkspaceRoot: root,
		Tabs:          &Tabs{Workspaces: map[string]*WorkspaceTabs{}},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			View(c, m)
		})
	}, 720, 500)
	if err := tt.Click(label("自动刷新文件树", "Auto-refresh tree")); err != nil {
		t.Fatalf("auto-refresh checkbox not reachable: %v", err)
	}
	w := tabsFor(m)
	if w == nil || !w.AutoRefreshDisabled {
		t.Fatal("click did not disable auto refresh")
	}
	if err := tt.Click(label("自动刷新文件树", "Auto-refresh tree")); err != nil {
		t.Fatalf("auto-refresh checkbox not reachable for re-enable: %v", err)
	}
	if w.AutoRefreshDisabled {
		t.Fatal("second click did not re-enable auto refresh")
	}
}
