package sidebar

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestArchivedHeaderOpensAndLeavesArchivedView(t *testing.T) {
	filter := "default"
	m := Model{BrowseArchivedFilter: &filter, Snapshot: session.ViewState{WorkspaceReady: true, ScratchCwd: "/scratch"}}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 280, 700)
	label := i18n.T(i18n.KeySidebarArchivedSection)
	if err := tt.Click(label); err != nil {
		t.Fatal(err)
	}
	if filter != "only" {
		t.Fatalf("archived header filter=%q, want only", filter)
	}
	tt.Frame()
	if err := tt.Click(label); err != nil {
		t.Fatal(err)
	}
	if filter != "default" {
		t.Fatalf("archived header did not return: filter=%q", filter)
	}
}
