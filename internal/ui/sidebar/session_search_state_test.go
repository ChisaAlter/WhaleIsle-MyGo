package sidebar

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestSessionSearchDismissalKeepsTypingEmptyAndNonemptyOutside(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale("en")
	defer i18n.SetLocale(previous)
	query := ""
	state := sessionSearchState{}
	menu := false
	focus := false
	m := Model{SearchQuery: &query, SearchFocus: &focus, searchState: &state}
	colors := theme.Resolve("deepseek.light").Light
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			workspaceHeading(c, m, &menu, chrome.Color(colors.FG), chrome.Color(colors.Muted))
			ui.Button(c, "Outside search").Height(40)
		})
	}, 340, 180)
	open := func() {
		t.Helper()
		if err := tt.Click(i18n.T(i18n.KeySidebarSearch)); err != nil {
			t.Fatal(err)
		}
		tt.Frame()
		if !state.Expanded {
			t.Fatal("opening search did not survive own press")
		}
	}
	open()
	if err := tt.Click("Outside search"); err != nil {
		t.Fatal(err)
	}
	if state.Expanded {
		t.Fatal("empty search did not collapse on outside click")
	}
	open()
	tt.Type("x")
	tt.Key(0, ui.KeyBackspace)
	if query != "" || !state.Expanded {
		t.Fatalf("backspace collapsed expanded input: %q %+v", query, state)
	}
	tt.Type("needle")
	if err := tt.Click("Outside search"); err != nil {
		t.Fatal(err)
	}
	if query != "needle" || !state.Expanded {
		t.Fatal("outside click discarded a nonempty search")
	}
	if err := tt.Click("Clear search"); err != nil {
		t.Fatal(err)
	}
	if query != "" || state.Expanded {
		t.Fatal("clear did not reset and collapse search")
	}
	focus = true
	tt.Frame()
	if !state.Expanded || focus {
		t.Fatal("external focus request did not expand and consume once")
	}
	tt.Key(0, ui.KeyEscape)
	if state.Expanded {
		t.Fatal("Escape did not collapse search")
	}
	m.Snapshot.WorkspaceLayout.GroupBy = "flat"
	tt.Frame()
	if !tt.HasText("Sessions") {
		t.Fatal("flat grouping retained the workspace heading")
	}
}

func TestSessionSearchResultHasTwoRowsAndReturnsToExistingTree(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale("en")
	defer i18n.SetLocale(previous)
	query := "needle"
	state := sessionSearchState{Expanded: true}
	opened := ""
	snap := session.ViewState{WorkspaceReady: true, ScratchCwd: "/scratch", SelectedID: "old", SearchRPCQuery: query, Workspaces: []session.WorkspaceRow{{WorkspaceID: "w", Title: "A long workspace label for clipping", SessionIDs: []string{"hit"}}}, Items: []session.Item{{SessionID: "hit", Title: "needle title", Running: true}}, SearchResults: []api.SessionSearchValueItemsItem{{SessionId: "hit", Snippet: "needle in long content"}}}
	snap.WorkspaceLayout.GroupBy = "flat"
	m := Model{SearchQuery: &query, searchState: &state, Snapshot: snap, OnSelect: func(id string) { opened = id }}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Width(280).Children(func() { sessionSearchResultsView(c, m, ui.RGB(0, 0, 0), ui.RGB(100, 100, 100)) })
	}, 280, 180)
	row, ok := tt.Find("Open search result hit")
	if !ok || row.H < 48 {
		t.Fatalf("search result has no48px seat: %+v", row)
	}
	title, _ := tt.Find("needle title")
	workspace, _ := tt.Find("A long workspace label for clipping")
	snippet, _ := tt.Find("needle in long content")
	meta, _ := tt.Find("Search result context hit")
	if title.Y >= workspace.Y || workspace.Y != snippet.Y || workspace.X >= snippet.X || workspace.W > meta.W*.4+1 || meta.X < row.X+20 {
		t.Fatalf("source two-row context violated row=%+v title=%+v ws=%+v snippet=%+v meta=%+v", row, title, workspace, snippet, meta)
	}
	if tt.HasText("● needle title") {
		t.Fatal("status leaked into the title instead of the status slot")
	}
	if err := tt.Click("Open search result hit"); err != nil {
		t.Fatal(err)
	}
	if opened != "hit" || query != "" || state.Expanded || state.Reveal != "hit" || state.RevealSerial == 0 {
		t.Fatalf("result did not close search and retain exact tree reveal: open=%q q=%q state=%+v", opened, query, state)
	}
}

func TestSessionSearchRevealConsumptionPreservesMarkAndAllowsFoldReset(t *testing.T) {
	state := sessionSearchState{Reveal: "hit-55", RevealSerial: 1}
	marks := workspaceWatermarkState{Marks: map[string]int{}}
	group := session.WorkspaceGroup{Key: "w", Kind: "workspace"}
	for i := 1; i <= 55; i++ {
		group.Items = append(group.Items, session.Item{SessionID: fmt.Sprintf("hit-%d", i), UpdatedAt: float64(56 - i)})
	}
	m := Model{searchState: &state, workspaceWatermarks: &marks}
	m.Snapshot.WorkspaceLayout.OrderBy = "updated"
	ui.NewTester(func(c *ui.Context) {
		visible := visibleWorkspaceSessionRows(m, group)
		retainWorkspaceWatermark(m, group.Key, visible.Watermark)
		row := ui.Box(c.Key("revealed-tree-row")).Height(32)
		revealSessionSearchRow(row, m, "hit-55")
	}, 300, 100)
	if state.Reveal != "" || state.RevealSerial != 1 || marks.Marks[group.Key] != 55 {
		t.Fatalf("reveal was not consumed after retaining its unlimited mark: state=%+v marks=%v", state, marks.Marks)
	}
	if got := visibleWorkspaceSessionRows(m, group).Watermark; got != 55 {
		t.Fatalf("consumed reveal lost its retained mark: %d", got)
	}
	resetWorkspaceWatermark(m, group.Key, []session.WorkspaceGroup{group})
	if got := visibleWorkspaceSessionRows(m, group).Watermark; got != workspaceWatermarkBase {
		t.Fatalf("stale search reveal defeated fold reset: %d", got)
	}
}
