package sidebar

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestBatchSearchRealCheckboxDispatchesOnceAndKeepsArchivedSearchInert(t *testing.T) {
	i18n.SetLocale("en")
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	query := "match"
	selected, opened := 0, 0
	m := Model{SearchQuery: &query, ThemeID: "deepseek.light", Batch: session.SessionBatch{Enabled: true, Selected: map[string]string{}}, Snapshot: session.ViewState{WorkspaceReady: true, WorkspaceLayout: session.WorkspaceLayoutState{ArchivedFilter: "show"}, Workspaces: []session.WorkspaceRow{{WorkspaceID: "w", Title: "Workspace", SessionIDs: []string{"plain", "archived", "blank"}}}, ArchivedSessionIDs: []string{"archived"}, Items: []session.Item{{SessionID: "plain", Title: "match plain"}, {SessionID: "archived", Title: "match archived"}, {SessionID: "blank", Title: "match blank", Blank: true}}}, OnSelect: func(string) { opened++ }}
	m.OnBatchEdit = func(action string, targets []session.SessionBatchTarget) {
		if action != "select" || len(targets) != 1 {
			t.Fatalf("wrong search batch action=%s targets=%v", action, targets)
		}
		selected++
		id := targets[0].ID
		if _, has := m.Batch.Selected[id]; has {
			delete(m.Batch.Selected, id)
		} else {
			m.Batch.Selected[id] = targets[0].Title
		}
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Width(360).Children(func() { sessionSearchResultsView(c, m, c.Theme().Text, c.Theme().TextMuted) })
	}, 440, 500)
	if tt.HasText("Select search checkbox blank") || tt.HasText("match blank") {
		t.Fatal("known blank entered selectable search results")
	}
	for _, id := range []string{"plain", "archived"} {
		r, ok := tt.Find("Select search checkbox " + id)
		if !ok || r.W != 16 || r.H != 16 {
			t.Fatalf("%s checkbox geometry=%+v/%t", id, r, ok)
		}
		before := selected
		if err := tt.Click("Select search checkbox " + id); err != nil {
			t.Fatal(err)
		}
		if selected != before+1 {
			t.Fatalf("%s checkbox dispatched %d times", id, selected-before)
		}
		before = selected
		if err := tt.Click("Select session " + id); err != nil {
			t.Fatal(err)
		}
		if selected != before+1 {
			t.Fatalf("%s search body dispatched %d times", id, selected-before)
		}
	}
	if opened != 0 || query != "match" {
		t.Fatal("batch search navigated or cleared query")
	}
	before := selected
	for _, confirming := range []bool{false, true} {
		m.Batch.Busy, m.Batch.Confirming = !confirming, confirming
		tt.Frame()
		_ = tt.Click("Select search checkbox plain")
		_ = tt.Click("Select session plain")
		if selected != before || opened != 0 {
			t.Fatal("busy/confirming search changed choices")
		}
	}
	m.Batch.Busy, m.Batch.Confirming, m.Batch.Enabled = false, false, false
	tt.Frame()
	if err := tt.Click("Open search result archived"); err != nil {
		t.Fatal(err)
	}
	if opened != 0 || query != "match" {
		t.Fatal("ordinary archived search opened a session")
	}
	if err := tt.Click("Open search result plain"); err != nil {
		t.Fatal(err)
	}
	if opened != 1 || query != "" {
		t.Fatal("ordinary search navigation changed")
	}
}

func batchResultFixture(token uint64, results ...session.SessionBatchResult) Model {
	return Model{Collapsed: true, ThemeID: "deepseek.light", ConnectionEpoch: 3, Snapshot: session.ViewState{WorkspaceLayout: session.WorkspaceLayoutState{HostGeneration: 4}}, Batch: session.SessionBatch{Enabled: true, Token: token, Kind: "archive", Attempted: len(results), Results: results}}
}

func TestBatchResultFailureStaysInTheSidebarFlowWithItsDetail(t *testing.T) {
	i18n.SetLocale("en")
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	m := batchResultFixture(9, session.SessionBatchResult{SessionBatchTarget: session.SessionBatchTarget{ID: "a", Title: "A"}}, session.SessionBatchResult{SessionBatchTarget: session.SessionBatchTarget{ID: "b", Title: "B"}, Error: "Host refused running session"}, session.SessionBatchResult{SessionBatchTarget: session.SessionBatchTarget{ID: "c", Title: "C"}, Error: strings.Repeat("detail ", 30) + "last error word"})
	m.Collapsed = false
	m.Batch.Selected = map[string]string{"b": "B", "c": "C"}
	dismissed := 0
	m.OnBatchEdit = func(action string, _ []session.SessionBatchTarget) {
		if action != "dismiss" {
			t.Fatal(action)
		}
		dismissed++
		m.Batch.Results = nil
	}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 800, 600)
	summary := "Archived 1 sessions；2 sessions unfinished"
	rect, ok := tt.Find(summary)
	if !ok {
		t.Fatalf("the failed batch has no receipt in the sidebar: %q", tt.Texts())
	}
	if rect.W < 700 {
		t.Fatalf("receipt is a %.0f-wide card, not a row of the sidebar", rect.W)
	}
	if tt.HasText("Host refused running session") || tt.HasText("last error word") {
		t.Fatalf("the Host's own error text stands in the receipt's sentence: %q", tt.Texts())
	}
	if err := tt.Click("Details"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("B: Host refused running session") || !tt.HasText("last error word") {
		t.Fatalf("the detail disclosure dropped a failed row: %q", tt.Texts())
	}
	tt.SetTime(time.Now().Add(time.Hour))
	tt.Frame()
	if !tt.HasText(summary) || dismissed != 0 {
		t.Fatal("the failure receipt expired before the reader closed it")
	}
	if err := tt.Click("Dismiss results"); err != nil {
		t.Fatal(err)
	}
	if dismissed != 1 || len(m.Batch.Selected) != 2 || len(m.Batch.Results) != 0 {
		t.Fatal("dismiss changed the failed selection or did not clear the result")
	}
	tt.Frame()
	if tt.HasText(summary) {
		t.Fatal("dismissed receipt reopened")
	}
}

func TestBatchResultSuccessIsAnAnnouncementNotAPageCard(t *testing.T) {
	i18n.SetLocale("en")
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	m := batchResultFixture(10, session.SessionBatchResult{SessionBatchTarget: session.SessionBatchTarget{ID: "a", Title: "A"}})
	base := batchResultFixture(10)
	m.Collapsed, base.Collapsed = false, false
	dismissed := 0
	m.OnBatchEdit = func(action string, _ []session.SessionBatchTarget) {
		if action != "dismiss" {
			t.Fatal(action)
		}
		dismissed++
		m.Batch.Results = nil
	}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 800, 600)
	if !announced(tt.Announcements(), "Archived 1 sessions") {
		t.Fatal("the completed batch was never announced")
	}
	quiet := ui.NewTester(func(c *ui.Context) { View(c, base) }, 800, 600)
	for _, anchor := range []string{"Plugins", "Scheduled tasks"} {
		with, okWith := tt.Find(anchor)
		without, okWithout := quiet.Find(anchor)
		if !okWith || !okWithout || with.W == 0 || with != without {
			t.Fatalf("%q moved when a completed batch landed: %+v / %+v", anchor, with, without)
		}
	}
	tt.SetTime(time.Now().Add(time.Hour))
	tt.Frame()
	if dismissed != 0 {
		t.Fatal("a success asked the model to dismiss the batch")
	}
	m.Batch.Token = 11
	m.Batch.Kind = "delete"
	m.Batch.Busy = true
	m.Batch.Results = []session.SessionBatchResult{{SessionBatchTarget: session.SessionBatchTarget{ID: "partial", Title: "Partial"}}}
	tt.Frame()
	if tt.HasText("Partial") || announced(tt.Announcements(), "Deleted 1 sessions") {
		t.Fatal("a busy batch announced its partial results")
	}
	m.Batch.Busy = false
	tt.Frame()
	if !announced(tt.Announcements(), "Deleted 1 sessions") {
		t.Fatal("the replacement batch did not announce its completion")
	}
}

func announced(list []string, s string) bool {
	for _, item := range list {
		if strings.Contains(item, s) {
			return true
		}
	}
	return false
}

func TestBatchResultOwnerScopeHostAndTokenRetireStaleFeedback(t *testing.T) {
	result := []session.SessionBatchResult{{SessionBatchTarget: session.SessionBatchTarget{ID: "a"}, Error: "failure"}}
	b := session.SessionBatch{Token: 1, Results: result}
	base := batchResultOwner{Token: 1, Host: 2, Connection: 3, Scope: session.NewSessionBatchScope(true, "workspace", "default", "")}
	var n batchResultNotice
	if !n.observe(base, b) {
		t.Fatal("initial completed failure did not show")
	}
	next := base
	next.Scope.Query = "different"
	if n.observe(next, b) {
		t.Fatal("visible result replayed after browser scope change")
	}
	b.Token = 2
	next.Token = 2
	if !n.observe(next, b) {
		t.Fatal("new token did not reset dismissed scope")
	}
	next.Host++
	next.Connection++
	if n.observe(next, b) {
		t.Fatal("old operation receipt replayed under a different Host")
	}
	b.Token++
	next.Token = b.Token
	b.Busy = true
	if n.observe(next, b) {
		t.Fatal("new Host partial results announced while busy")
	}
	next.Scope.Query = "during run"
	n.observe(next, b)
	b.Busy = false
	if !n.observe(next, b) {
		t.Fatal("scope switch during confirmed operation suppressed its final result")
	}
}
