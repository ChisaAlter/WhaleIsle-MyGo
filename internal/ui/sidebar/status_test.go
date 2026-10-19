package sidebar

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/store/session"
)

func TestSidebarSessionStatusPriority(t *testing.T) {
	it := session.Item{SessionID: "a", Running: true, CompletionUnread: true}
	m := Model{PendingStatuses: map[string]string{"a": "plan-review"}}
	if sessionStatus(m, it) != "plan-review" {
		t.Fatal("running hid pending")
	}
	delete(m.PendingStatuses, "a")
	if sessionStatus(m, it) != "running" {
		t.Fatal("completion hid running")
	}
	it.Running = false
	if sessionStatus(m, it) != "completed" {
		t.Fatal("unread completion missing")
	}
	it.CompletionUnread = false
	if sessionStatus(m, it) != "" {
		t.Fatal("unknown status invented")
	}
	for _, kind := range []string{"approval", "question", "plan-review"} {
		if statusLabel(kind) == "" || statusPrefix(kind) == "" {
			t.Fatal(kind)
		}
	}
}

func TestSidebarStatusNormalAndSearchRows(t *testing.T) {
	item := session.Item{SessionID: "a", Title: "Status fixture", Running: true, CompletionUnread: true}
	query := "Status"
	m := Model{Snapshot: session.ViewState{WorkspaceReady: true, Workspaces: []session.WorkspaceRow{{WorkspaceID: "ws", SessionIDs: []string{"a"}}}, Items: []session.Item{item}}, PendingStatuses: map[string]string{"a": "question"}, SearchQuery: &query}
	want := statusPrefix("question") + item.Title
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { rowSession(c, item, 0, m, ui.RGB(1, 2, 3), ui.RGB(4, 5, 6), ui.Transparent) })
	}, 400, 300)
	if _, ok := tt.Find(statusLabel("question")); !ok {
		t.Fatal("normal row omitted waiting-answer status slot")
	}
	search := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { sessionSearchResultsView(c, m, ui.RGB(1, 2, 3), ui.RGB(4, 5, 6)) })
	}, 400, 400)
	if _, ok := search.Find(want); !ok {
		t.Fatal("search row omitted waiting-answer label")
	}
}
