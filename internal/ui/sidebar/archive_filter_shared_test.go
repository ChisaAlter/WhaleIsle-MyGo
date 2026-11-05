package sidebar

import (
	"reflect"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestArchivedFilterBrowsingAndSearchUseOneLiveStore(t *testing.T) {
	previous := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(previous) })
	i18n.SetLocale("en")
	for _, grouping := range []string{"workspace", "workspace-tree", "flat"} {
		t.Run(grouping, func(t *testing.T) {
			store := session.New()
			store.SetWorkspaceGrouping(grouping)
			query, open := "", false
			changes := []string{}
			m := Model{SearchQuery: &query, WorkspaceMenuOpen: &open, HideArchivedList: true, Snapshot: session.ViewState{
				WorkspaceReady: true, ScratchCwd: "/scratch", SelectedID: "active",
				Items:              []session.Item{{SessionID: "active", Title: "needle active", Cwd: "/project"}, {SessionID: "archived", Title: "needle archived", Cwd: "/project"}},
				Workspaces:         []session.WorkspaceRow{{WorkspaceID: "project", Title: "Project", Path: "/project", SessionIDs: []string{"active", "archived"}}},
				ArchivedSessionIDs: []string{"archived"},
			}}
			m.OnArchivedFilter = func(filter string) { changes = append(changes, filter); store.SetWorkspaceArchivedFilter(filter) }
			render := func(c *ui.Context) { m.Snapshot.WorkspaceLayout = store.Snapshot().WorkspaceLayout; View(c, m) }
			tt := ui.NewTester(render, 480, 1200)
			choose := func(label string) {
				t.Helper()
				if err := tt.Click("View options"); err != nil {
					t.Fatal(err)
				}
				if err := tt.Click(label); err != nil {
					t.Fatal(err)
				}
				tt.Key(0, ui.KeyEscape)
				tt.Key(0, ui.KeyEscape)
			}
			check := func(filter string, active, archived bool) {
				t.Helper()
				tt.Frame()
				if got := store.Snapshot().WorkspaceLayout.ArchivedFilter; got != filter || tt.HasText("needle active") != active || tt.HasText("needle archived") != archived {
					t.Fatalf("filter=%q want=%q active=%v archived=%v texts=%q", got, filter, active, archived, tt.Texts())
				}
				if browseArchiveFilter(m) != filter || searchArchiveFilter(m) != filter {
					t.Fatal("browse and search projections disagree")
				}
				if m.Snapshot.SelectedID != "active" || m.Snapshot.Workspaces[0].Path != "/project" || m.Snapshot.WorkspaceLayout.GroupBy != grouping {
					t.Fatal("filter changed selection, workspace, or grouping")
				}
				ids := []string{}
				for _, target := range VisibleBatchTargets(m) {
					ids = append(ids, target.ID)
				}
				want := []string{}
				if active {
					want = append(want, "active")
				}
				if archived {
					want = append(want, "archived")
				}
				if !reflect.DeepEqual(ids, want) {
					t.Fatalf("batch targets %v want %v", ids, want)
				}
			}
			check("default", true, false)
			choose("Only archived")
			check("only", false, true)
			query = "needle"
			tt.Frame()
			check("only", false, true)
			query = ""
			tt.Frame()
			choose("All conversations (show archived)")
			query = "needle"
			tt.Frame()
			check("show", true, true)
			query = ""
			tt.Frame()
			check("show", true, true)
			choose("Hide archived")
			check("default", true, false)
			query = "needle"
			tt.Frame()
			check("default", true, false)
			query = ""
			tt.Frame()
			choose("Only archived")
			query = ""
			tt.Frame()
			check("only", false, true)
			if !reflect.DeepEqual(changes, []string{"only", "show", "default", "only"}) {
				t.Fatalf("controls did not write the same callback: %v", changes)
			}
			// A fresh retained UI reads the shared choice without a ui.Local reset.
			tt = ui.NewTester(render, 480, 1200)
			check("only", false, true)
		})
	}
}

func TestArchivedFilterSnapshotWinsAndManagementExpansionStaysLocal(t *testing.T) {
	previous := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(previous) })
	i18n.SetLocale("en")
	browse, search := "only", "show"
	writes, toggles := 0, 0
	m := Model{BrowseArchivedFilter: &browse, SearchArchivedFilter: &search, Snapshot: session.ViewState{
		WorkspaceReady: true, ScratchCwd: "/scratch", SelectedID: "active",
		WorkspaceLayout: session.WorkspaceLayoutState{ArchivedFilter: "default", GroupBy: "workspace-tree", Collapsed: map[string]bool{"closed": true}},
		Items:           []session.Item{{SessionID: "active", Title: "Active session", Cwd: "/scratch"}, {SessionID: "archived", Title: "Archived session", Cwd: "/scratch"}}, ArchivedSessionIDs: []string{"archived"},
	}}
	m.OnArchivedFilter = func(filter string) { writes++; m.Snapshot.WorkspaceLayout.ArchivedFilter = filter }
	m.OnToggleWorkspaceGroup = func(string) { toggles++ }
	render := func(c *ui.Context) { View(c, m) }
	tt := ui.NewTester(render, 480, 1200)
	if browseArchiveFilter(m) != "default" || searchArchiveFilter(m) != "default" || !tt.HasText("Active session") || tt.HasText("Archived session") {
		t.Fatal("legacy pointer overrode store snapshot")
	}
	if err := tt.Click(i18n.T(i18n.KeySidebarArchivedSection)); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Archived session") || writes != 0 || toggles != 0 || m.Snapshot.SelectedID != "active" || m.Snapshot.ScratchCwd != "/scratch" || !m.Snapshot.WorkspaceLayout.Collapsed["closed"] {
		t.Fatal("management expansion changed navigation or filter")
	}
	tt = ui.NewTester(render, 480, 1200)
	if tt.HasText("Archived session") || !tt.HasText("Active session") || writes != 0 || browse != "only" || search != "show" {
		t.Fatal("management expansion persisted or stale pointers became live")
	}
}

func TestArchivedFilterOnlyEmptyReturnWritesSharedCallback(t *testing.T) {
	m := Model{Snapshot: session.ViewState{WorkspaceReady: true, ScratchCwd: "/scratch", WorkspaceLayout: session.WorkspaceLayoutState{ArchivedFilter: "only", Collapsed: map[string]bool{"::scratch": false}}, Items: []session.Item{{SessionID: "active", Title: "Return session", Cwd: "/scratch"}}}}
	writes := []string{}
	m.OnArchivedFilter = func(filter string) {
		writes = append(writes, filter)
		m.Snapshot.WorkspaceLayout.ArchivedFilter = filter
	}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 480, 1200)
	if err := tt.Click(workspaceText("查看其他会话", "View other sessions")); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(writes, []string{"default"}) || browseArchiveFilter(m) != "default" || searchArchiveFilter(m) != "default" || !tt.HasText("Return session") {
		t.Fatal("empty-state return bypassed shared preference")
	}
}

func TestArchivedFilterSearchKeepsPreferenceConflictAndRetryVisible(t *testing.T) {
	query := "needle"
	m := Model{SearchQuery: &query, Snapshot: session.ViewState{WorkspaceReady: true, ScratchCwd: "/scratch", WorkspaceLayout: session.WorkspaceLayoutState{ArchivedFilter: "only", PreferenceError: "view preference archivedFilter changed in another window"}}}
	writes, retries := 0, 0
	m.OnArchivedFilter = func(filter string) { writes++; m.Snapshot.WorkspaceLayout.ArchivedFilter = filter }
	m.OnRetryWorkspacePreferences = func() { retries++ }
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 480, 1200)
	if !tt.HasText(m.Snapshot.WorkspaceLayout.PreferenceError) || !tt.HasText(workspaceText("无匹配会话", "No matching sessions")) {
		t.Fatal("search hid a preference conflict or no-match feedback")
	}
	if err := tt.Click(workspaceText("重试视图偏好", "Retry view preferences")); err != nil {
		t.Fatal(err)
	}
	if retries != 1 || writes != 0 || query != "needle" || sharedArchiveFilter(m) != "only" {
		t.Fatal("preference retry changed query or pending filter")
	}
}
