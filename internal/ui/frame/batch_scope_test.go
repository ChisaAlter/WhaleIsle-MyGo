package frame

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

func TestNavigationViewBindsRenderedBatchScopeBeforeReadingSelection(t *testing.T) {
	sizes := DefaultLayout()
	query := "original"
	m := Model{Store: session.New(), Sizes: &sizes, ThemeID: "deepseek.light", SessionSearch: &query}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Width(280).FillHeight().Children(func() { navigationView(c, m, m.Store.Snapshot()) })
	}, 1200, 800)
	m.Store.EditSessionBatch("toggle", nil)
	selectOne := func() { m.Store.EditSessionBatch("select", []session.SessionBatchTarget{{ID: "a", Title: "A"}}) }
	assertClear := func(reason string) {
		t.Helper()
		tt.Frame()
		if got := m.Store.SessionBatch(); len(got.Selected) != 0 || !got.Enabled {
			t.Fatalf("%s retained choices %+v", reason, got)
		}
	}
	selectOne()
	query = "new"
	assertClear("search")
	selectOne()
	m.Store.SetWorkspaceGrouping("flat")
	assertClear("grouping")
	selectOne()
	m.Store.SetWorkspaceArchivedFilter("only")
	assertClear("archive visibility")
	selectOne()
	sizes.ToggleSidebar()
	assertClear("wide to rail")
	selectOne()
	m.Store.SetWorkspaceOrdering("updated")
	m.Store.ToggleWorkspaceGroup("group")
	tt.Frame()
	if len(m.Store.SessionBatch().Selected) != 1 {
		t.Fatal("sorting/group disclosure changed browser selection scope")
	}
}
