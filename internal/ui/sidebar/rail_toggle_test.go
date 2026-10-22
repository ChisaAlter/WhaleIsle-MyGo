package sidebar

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestCollapsedRailBrandToggleExpands(t *testing.T) {
	toggled := 0
	m := Model{Collapsed: true, OnToggle: func() { toggled++ }, Snapshot: session.ViewState{WorkspaceReady: true}}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 72, 800)
	if err := tt.Click(i18n.T(i18n.KeySidebarExpand)); err != nil {
		t.Fatal(err)
	}
	if toggled != 1 {
		t.Fatalf("expand calls=%d, want 1", toggled)
	}
}
