package sidebar

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// Removing archive visibility from the search projection or the action hooks
// must fail this real sidebar interaction test.
func TestSearchArchivedFilterAndActions(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale("zh")
	query := "needle"
	snap := session.ViewState{WorkspaceReady: true, ScratchCwd: "/scratch", SearchRPCQuery: query,
		ArchivedSessionIDs: []string{"archived", "body"},
		Items:              []session.Item{{SessionID: "active", Title: "needle active", Cwd: "/scratch"}, {SessionID: "archived", Title: "needle archive", Cwd: "/scratch"}, {SessionID: "body", Title: "old conversation", Cwd: "/scratch"}},
		SearchResults:      []api.SessionSearchValueItemsItem{{SessionId: "body", Snippet: "needle in body"}},
	}
	restored, deleted, opened := "", "", ""
	var selected []session.SessionBatchTarget
	m := Model{Snapshot: snap, SearchQuery: &query, OnSelect: func(id string) { opened = id }, OnUnarchive: func(id string) { restored = id }, OnDelete: func(id string) { deleted = id }, OnBatchEdit: func(_ string, targets []session.SessionBatchTarget) { selected = targets }}
	m.OnArchivedFilter = func(filter string) { m.Snapshot.WorkspaceLayout.ArchivedFilter = filter }
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 480, 1400)
	if tt.HasText("needle archive") {
		t.Fatal("default search includes archive")
	}
	for _, extra := range []string{"Show archived results", "Hide archived results", "Only archived results", "Restore search result archived", "Delete search result body"} {
		if _, ok := tt.Find(extra); ok {
			t.Fatalf("non-source inline control %q", extra)
		}
	}
	// The shared view option owns both projections, not a second filter row.
	m.OnArchivedFilter("show")
	tt.Frame()
	if !tt.HasText("needle archive") || !tt.HasText("needle in body") || !tt.HasText("needle active") {
		t.Fatal("show mode omitted local or content matches")
	}
	if err := tt.Click("Open search result archived"); err != nil {
		t.Fatal(err)
	}
	if opened != "" || query != "needle" {
		t.Fatal("archived result opened or restored on row click")
	}
	if err := tt.Click(sessionActionsLabel("archived", "needle archive")); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem(i18n.T(i18n.KeySidebarUnarchive)); err != nil {
		t.Fatal(err)
	}
	if restored != "archived" || opened != "" || query != "needle" {
		t.Fatalf("restore navigated or lost query: %q %q %q", restored, opened, query)
	}
	if err := tt.RightClick("Open search result body"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem(i18n.T(i18n.KeySidebarDelete)); err != nil {
		t.Fatal(err)
	}
	if deleted != "body" || opened != "" {
		t.Fatalf("delete target %q open %q", deleted, opened)
	}
	m.OnArchivedFilter("only")
	tt.Frame()
	if tt.HasText("needle active") || !tt.HasText("needle archive") {
		t.Fatal("only mode filter mismatch")
	}
	m.Batch.Enabled = true
	tt.Frame()
	if _, ok := tt.Find(sessionActionsLabel("body", "old conversation")); ok {
		t.Fatal("row actions visible in batch mode")
	}
	if err := tt.Click("Select session body"); err != nil {
		t.Fatal(err)
	}
	if len(selected) != 1 || selected[0].ID != "body" {
		t.Fatalf("batch target=%v", selected)
	}
	m.OnArchivedFilter("default")
	tt.Frame()
	if tt.HasText("needle archive") {
		t.Fatal("hide mode retained archive")
	}
}

func TestSearchArchiveProjectionAndBatchShareFilter(t *testing.T) {
	query, filter := "needle", "only"
	snap := session.ViewState{WorkspaceReady: true, ScratchCwd: "/scratch", SearchRPCQuery: query,
		ArchivedSessionIDs: []string{"archive", "body", "child"},
		Items:              []session.Item{{SessionID: "active", Title: "needle", Cwd: "/scratch"}, {SessionID: "archive", Title: "needle", Cwd: "/scratch"}, {SessionID: "body", Title: "body only", Cwd: "/scratch"}, {SessionID: "child", Title: "needle", Origin: "subagent", Cwd: "/scratch"}},
		SearchResults:      []api.SessionSearchValueItemsItem{{SessionId: "body", Snippet: "needle"}, {SessionId: "body", Snippet: "duplicate"}},
	}
	m := Model{Snapshot: snap, SearchQuery: &query, SearchArchivedFilter: &filter}
	targets := VisibleBatchTargets(m)
	if len(targets) != 2 {
		t.Fatalf("archive batch targets=%v", targets)
	}
	rows, _ := deriveSessionSearchResults(snap, query, filter)
	for i, row := range rows {
		if !row.Archived || targets[i].ID != row.Item.SessionID {
			t.Fatalf("batch/search mismatch: %v %v", targets, rows)
		}
	}
	snap.SearchRPCQuery = "stale"
	rows, _ = deriveSessionSearchResults(snap, query, filter)
	if len(rows) != 1 || rows[0].Snippet != "" {
		t.Fatalf("stale content leaked: %+v", rows)
	}
	snap.SearchLoading = true
	rows, _ = deriveSessionSearchResults(snap, query, "show")
	if len(rows) != 2 {
		t.Fatalf("loading lost local results: %+v", rows)
	}
	rows, _ = deriveSessionSearchResults(snap, query, "invalid")
	if len(rows) != 1 || rows[0].Archived {
		t.Fatalf("unknown filter must hide archived: %+v", rows)
	}
}
