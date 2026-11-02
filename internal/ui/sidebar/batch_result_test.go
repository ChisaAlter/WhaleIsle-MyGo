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

func TestBatchResultRootToastRetainsAllErrorsAndDismissesOnlyResults(t *testing.T) {
	i18n.SetLocale("en")
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	m := batchResultFixture(9, session.SessionBatchResult{SessionBatchTarget: session.SessionBatchTarget{ID: "a", Title: "A"}}, session.SessionBatchResult{SessionBatchTarget: session.SessionBatchTarget{ID: "b", Title: "B"}, Error: "Host refused running session"}, session.SessionBatchResult{SessionBatchTarget: session.SessionBatchTarget{ID: "c", Title: "C"}, Error: strings.Repeat("detail ", 30) + "last error word"})
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
	if !tt.HasText("Archived 1 sessions；2 sessions unfinished") || !tt.HasText("B: Host refused running session") || !tt.HasText("last error word") {
		t.Fatalf("root result dropped summary or full wire errors: %q", tt.Texts())
	}
	if _, ok := tt.Find("Batch failure b"); !ok {
		t.Fatal("first failure has no addressable detail")
	}
	if _, ok := tt.Find("Batch failure c"); !ok {
		t.Fatal("last failure has no addressable detail")
	}
	tt.SetTime(time.Now().Add(time.Hour))
	if !tt.HasText("Batch operation results") || dismissed != 0 {
		t.Fatal("failure toast expired before acknowledgement")
	}
	if err := tt.Click("Dismiss results"); err != nil {
		t.Fatal(err)
	}
	if dismissed != 1 || len(m.Batch.Selected) != 2 || len(m.Batch.Results) != 0 {
		t.Fatal("dismiss changed failed selection or did not clear result")
	}
	tt.Frame()
	if tt.HasText("Batch operation results") {
		t.Fatal("dismissed result reopened")
	}
}

func TestBatchResultSuccessExpiresOnceWithoutClearingReplacementOrBusyBatch(t *testing.T) {
	i18n.SetLocale("en")
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	m := batchResultFixture(10, session.SessionBatchResult{SessionBatchTarget: session.SessionBatchTarget{ID: "a", Title: "A"}})
	dismissed := 0
	m.OnBatchEdit = func(action string, _ []session.SessionBatchTarget) {
		if action != "dismiss" {
			t.Fatal(action)
		}
		dismissed++
		m.Batch.Results = nil
	}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 800, 600)
	if !tt.HasText("Archived 1 sessions") {
		t.Fatal("completed batch success absent")
	}
	tt.SetTime(time.Now().Add(time.Second))
	if dismissed != 0 {
		t.Fatal("success was dismissed before source hold elapsed")
	}
	m.Batch.Token = 11
	m.Batch.Kind = "delete"
	m.Batch.Busy = true
	m.Batch.Results = []session.SessionBatchResult{{SessionBatchTarget: session.SessionBatchTarget{ID: "partial", Title: "Partial"}}}
	tt.Frame()
	tt.SetTime(time.Now().Add(10 * time.Second))
	if dismissed != 0 || tt.HasText("Batch operation results") {
		t.Fatal("old success wake cleared replacement or announced partial busy results")
	}
	m.Batch.Busy = false
	tt.Frame()
	if !tt.HasText("Deleted 1 sessions") {
		t.Fatal("replacement batch did not announce completion")
	}
	tt.SetTime(time.Now().Add(14 * time.Second))
	if dismissed != 1 || tt.HasText("Batch operation results") {
		t.Fatal("completed success did not auto-expire exactly once")
	}
	tt.Frame()
	if dismissed != 1 {
		t.Fatal("result expiry was repeated")
	}
}

func TestBatchResultOwnerScopeHostAndTokenRetireStaleFeedback(t *testing.T) {
	result := []session.SessionBatchResult{{SessionBatchTarget: session.SessionBatchTarget{ID: "a"}, Error: "failure"}}
	b := session.SessionBatch{Token: 1, Results: result}
	base := batchResultOwner{Token: 1, Host: 2, Connection: 3, Scope: session.NewSessionBatchScope(true, "workspace", "default", "")}
	var n batchResultNotice
	if !n.observe(base, b, time.Now()) {
		t.Fatal("initial completed failure did not show")
	}
	next := base
	next.Scope.Query = "different"
	if n.observe(next, b, time.Now()) {
		t.Fatal("visible result replayed after browser scope change")
	}
	b.Token = 2
	next.Token = 2
	if !n.observe(next, b, time.Now()) {
		t.Fatal("new token did not reset dismissed scope")
	}
	next.Host++
	next.Connection++
	if n.observe(next, b, time.Now()) {
		t.Fatal("old operation receipt replayed under a different Host")
	}
	b.Token++
	next.Token = b.Token
	b.Busy = true
	if n.observe(next, b, time.Now()) {
		t.Fatal("new Host partial results announced while busy")
	}
	next.Scope.Query = "during run"
	n.observe(next, b, time.Now())
	b.Busy = false
	if !n.observe(next, b, time.Now()) {
		t.Fatal("scope switch during confirmed operation suppressed its final result")
	}
}
