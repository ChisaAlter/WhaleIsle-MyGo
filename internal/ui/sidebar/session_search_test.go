package sidebar

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestNativeMergedSessionSearchViewFixture(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale("zh")
	query := " needle "
	snap := session.ViewState{WorkspaceReady: true, ScratchCwd: "/scratch", Workspaces: []session.WorkspaceRow{{WorkspaceID: "ws", Title: "Needle project", SessionIDs: []string{"local", "workspace", "remote1", "remote2", "blank", "child", "archived"}}}, ArchivedSessionIDs: []string{"archived"}, SearchRPCQuery: "needle", SearchHasMore: true,
		Items:         []session.Item{{SessionID: "local", Title: "Needle title", UpdatedAt: 5}, {SessionID: "workspace", Title: "Workspace title", UpdatedAt: 10}, {SessionID: "remote1", Title: "Ranked first", Cwd: "/scratch", UpdatedAt: 20}, {SessionID: "remote2", Title: "Ranked second", Cwd: "/scratch", UpdatedAt: 30}, {SessionID: "blank", Title: "Needle blank", Blank: true}, {SessionID: "child", Title: "Needle child", Origin: "subagent"}, {SessionID: "archived", Title: "Needle archive"}, {SessionID: "orphan", Title: "Needle orphan", Cwd: "/deleted"}},
		SearchResults: []api.SessionSearchValueItemsItem{{SessionId: "remote1", Snippet: "first body match"}, {SessionId: "remote2", Snippet: "second body match"}, {SessionId: "local", Snippet: "title and body"}, {SessionId: "remote1", Snippet: "duplicate"}, {SessionId: "orphan", Snippet: "hidden"}, {SessionId: "child", Snippet: "hidden"}}}
	// Only two workspace members are local matches; scratch-only rows use Host order.
	snap.Workspaces[0].SessionIDs = []string{"local", "workspace", "blank", "child", "archived"}
	rows, more := deriveSessionSearchResults(snap, query)
	want := []string{"workspace", "local", "remote1", "remote2"}
	if len(rows) != len(want) || !more {
		t.Fatalf("rows=%+v more=%v", rows, more)
	}
	for i, id := range want {
		if rows[i].Item.SessionID != id {
			t.Fatalf("rank %d=%s want %s", i, rows[i].Item.SessionID, id)
		}
	}
	if rows[1].Snippet != "title and body" || rows[2].Snippet != "first body match" {
		t.Fatal("first snippet/dedup lost")
	}
	snap.SearchLoading = true
	pending, _ := deriveSessionSearchResults(snap, query)
	if len(pending) != 2 || pending[0].Snippet != "" {
		t.Fatal("loading must retain only local matches")
	}
	snap.SearchLoading = false
	snap.SearchErr = "index unavailable"
	failed, _ := deriveSessionSearchResults(snap, query)
	if len(failed) != 2 {
		t.Fatal("error removed title/workspace results")
	}
	snap.SearchErr = ""
	snap.SearchRPCQuery = "other"
	stale, _ := deriveSessionSearchResults(snap, query)
	if len(stale) != 2 {
		t.Fatal("old query results leaked")
	}
	snap.SearchRPCQuery = "needle"
	snap.SelectedID = "local"
	snap.Items[0].Running = true
	snap.WorkspaceLayout.Collapsed = map[string]bool{"::scratch": true}
	var expanded []string
	var opened string
	m := Model{Snapshot: snap, SearchQuery: &query, ThemeID: "light", OnSelect: func(id string) { opened = id }, OnToggleWorkspaceGroup: func(id string) { expanded = append(expanded, id); snap.WorkspaceLayout.Collapsed[id] = false }}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 480, 1200)
	for _, label := range []string{"Needle title", "Workspace title", "first body match", fmt.Sprintf(workspaceText("仅显示前 %d 条结果，请缩小搜索范围。", "Only the first %d results are shown. Narrow your search."), session.SessionSearchResultLimit)} {
		if !tt.HasText(label) {
			t.Fatalf("missing actual sidebar search content %q", label)
		}
	}
	if err := tt.Click("Open search result remote1"); err != nil {
		t.Fatal(err)
	}
	if len(expanded) != 1 || expanded[0] != "::scratch" {
		t.Fatalf("result remained collapsed: %v", expanded)
	}
	if opened != "remote1" || query != "" {
		t.Fatalf("search open/clear: %q %q", opened, query)
	}
	// A second explicit search can open the same result again.
	query = "needle"
	opened = ""
	tt.Frame()
	if err := tt.Click("Open search result remote1"); err != nil {
		t.Fatal(err)
	}
	if opened != "remote1" {
		t.Fatal("repeated result opening lost")
	}

	// Workspace-tree ancestors reveal before ordinary navigation resumes.
	nested := session.ViewState{Workspaces: []session.WorkspaceRow{{WorkspaceID: "parent", Path: "/a"}, {WorkspaceID: "child-ws", Path: "/a/b", SessionIDs: []string{"hit"}}}, Items: []session.Item{{SessionID: "hit"}}}
	nested.WorkspaceLayout.GroupBy = "workspace-tree"
	nested.WorkspaceLayout.Collapsed = map[string]bool{"parent": true, "child-ws": true}
	var reveal []string
	revealSessionSearchResult(Model{Snapshot: nested, OnToggleWorkspaceGroup: func(id string) { reveal = append(reveal, id) }}, "hit")
	if len(reveal) != 2 || reveal[0] != "parent" || reveal[1] != "child-ws" {
		t.Fatalf("ancestor reveal=%v", reveal)
	}
	// Local matches alone are bounded and advertise truncation, never auto-page.
	big := session.ViewState{WorkspaceReady: true, ScratchCwd: "/scratch"}
	for i := 0; i < 23; i++ {
		big.Items = append(big.Items, session.Item{SessionID: fmt.Sprint(i), Title: "needle", Cwd: "/scratch", UpdatedAt: float64(i)})
	}
	bounded, more := deriveSessionSearchResults(big, "needle")
	if len(bounded) != 20 || !more {
		t.Fatalf("bound=%d more=%v", len(bounded), more)
	}
}
