// Session previews use the same retained root and row registration as production.
package sidebar

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	schedstore "whaleisle/internal/store/schedule"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

type hoverDraftInput struct {
	archived  map[string]bool
	pressed   map[string]bool
	dragging  bool
	menu      bool
	mutations int
}

func sessionHoverDraftFixture(t *testing.T) (*ui.Tester, *Model, *hoverDraftInput, time.Time) {
	t.Helper()
	locale := i18n.Locale()
	i18n.SetLocale("en")
	t.Cleanup(func() { i18n.SetLocale(locale) })
	start := time.Date(2026, 10, 28, 1, 0, 0, 0, time.UTC)
	m := &Model{Snapshot: session.ViewState{SelectedID: "a", WorkspaceLayout: session.WorkspaceLayoutState{HostGeneration: 21}, Items: []session.Item{
		{SessionID: "a", Title: "Original title A", Cwd: "SHOULD-NOT-APPEAR-AS-PREVIEW-DATA", UpdatedAt: float64(start.Add(-5 * time.Minute).UnixMilli())},
		{SessionID: "b", Title: "Original title B", UpdatedAt: float64(start.Add(-10 * time.Minute).UnixMilli())},
	}}}
	in := &hoverDraftInput{archived: map[string]bool{}, pressed: map[string]bool{}}
	tt := ui.NewTester(func(c *ui.Context) {
		root := ui.Column(c).Width(240).FillHeight()
		*m = beginSessionHover(root, *m)
		root.Children(func() {
			for _, item := range m.Snapshot.Items {
				row := ui.ButtonBase(c.Key(item.SessionID)).Label("Session row " + item.SessionID).Width(240).Height(32)
				row.Children(func() { ui.Text(c, item.Title).SingleLine() })
				if in.menu {
					row.Menu(func(menu *ui.Menu) { menu.Item("Only item") })
				}
				if row.Clicked() {
					in.mutations++
				}
				if in.archived[item.SessionID] {
					archivedSessionHoverCard(c, *m, row, item, item.Title, in.pressed[item.SessionID])
				} else {
					sessionHoverCard(c, *m, row, item, item.Title, sessionStatus(*m, item), theme.Colors{}, in.pressed[item.SessionID])
				}
			}
		})
		renderSessionHover(c, *m, root, in.dragging)
	}, 760, 700)
	tt.SetTime(start)
	return tt, m, in, start
}

func hoverDraftRow(t *testing.T, tt *ui.Tester, id string) {
	t.Helper()
	r, ok := tt.Find("Session row " + id)
	if !ok {
		t.Fatalf("missing %s", id)
	}
	tt.Move(r.X+60, r.Y+r.H/2)
	tt.Frame()
}

func TestDraftSessionHoverDelayWholeCardCopyAndFeedback(t *testing.T) {
	tt, m, in, start := sessionHoverDraftFixture(t)
	m.Snapshot.Items[0].Title = "  " + strings.Repeat("Full original title must remain visible across many wrapped lines. ", 7) + " EXACT END  "
	tt.Frame()
	tt.SetClipboard("unchanged")
	hoverDraftRow(t, tt, "a")
	tt.SetTime(start.Add(799 * time.Millisecond))
	if tt.HasText("Session preview: a") {
		t.Fatal("opened before800ms")
	}
	tt.SetTime(start.Add(800 * time.Millisecond))
	card, ok := tt.Find("Copy session title a")
	if !ok || card.W != 244 {
		t.Fatalf("wrong whole-card geometry: %+v %q", card, tt.Texts())
	}
	if card.H <= 3*20+24 {
		t.Fatalf("full title still clipped to3lines: %+v", card)
	}
	if !tt.HasText("Session preview: a") || !tt.HasText(m.Snapshot.Items[0].Title) || tt.HasText(m.Snapshot.Items[0].Cwd) {
		t.Fatalf("missing full title or fabricated cwd: %q", tt.Texts())
	}
	// Cross the actual8DIP anchor/card gap within the source's200ms grace.
	row, _ := tt.Find("Session row a")
	tt.Move(row.X+row.W+4, card.Y+8)
	tt.SetTime(start.Add(900 * time.Millisecond))
	tt.Move(card.X+20, card.Y+20)
	if err := tt.Click("Copy session title a"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if tt.Clipboard() != m.Snapshot.Items[0].Title || !tt.HasText("Copied") || in.mutations != 0 || m.Snapshot.SelectedID != "a" {
		t.Fatalf("wrong copy/side effects clipboard=%q mutations=%d texts=%q", tt.Clipboard(), in.mutations, tt.Texts())
	}
	feedback, _ := tt.Find("Copy session title a")
	if feedback.H != card.H {
		t.Fatal("copy feedback resized card")
	}
	tt.SetTime(start.Add(1899 * time.Millisecond))
	if !tt.HasText("Copied") {
		t.Fatal("feedback shorter than1s")
	}
	tt.SetTime(start.Add(1900 * time.Millisecond))
	if tt.HasText("Copied") {
		t.Fatal("feedback did not restore content")
	}
	for index, key := range []ui.Key{ui.KeyEnter, ui.KeySpace} {
		tt.SetClipboard("keyboard unchanged")
		tt.Key(0, key)
		tt.Frame()
		if tt.Clipboard() != m.Snapshot.Items[0].Title || in.mutations != 0 {
			t.Fatalf("wholecard key%v failed", key)
		}
		tt.SetTime(start.Add(time.Duration(3+index) * time.Second))
	}
}

func TestDraftSessionHoverGroupedTransferArchiveDelayAndLeave(t *testing.T) {
	tt, m, in, start := sessionHoverDraftFixture(t)
	hoverDraftRow(t, tt, "a")
	tt.SetTime(start.Add(800 * time.Millisecond))
	hoverDraftRow(t, tt, "b")
	if !tt.HasText("Session preview: b") || tt.HasText("Session preview: a") {
		t.Fatal("ordinary warm transfer failed")
	}
	tt.Move(700, 650)
	tt.SetTime(start.Add(999 * time.Millisecond))
	if !tt.HasText("Session preview: b") {
		t.Fatal("lost200ms grace")
	}
	tt.SetTime(start.Add(1000 * time.Millisecond))
	if tt.HasText("Session preview: b") {
		t.Fatal("card remained after200ms")
	}
	in.archived["b"] = true
	m.Snapshot.ArchivedSessionIDs = []string{"b"}
	tt.Frame()
	hoverDraftRow(t, tt, "b")
	tt.SetTime(start.Add(1499 * time.Millisecond))
	if tt.HasText("Session preview: b") {
		t.Fatal("archive opened before500ms")
	}
	tt.SetTime(start.Add(1500 * time.Millisecond))
	if !tt.HasText("Session preview: b") || !tt.HasText("Archived") || tt.HasText("Idle") {
		t.Fatalf("archive body wrong: %q", tt.Texts())
	}
}

func TestDraftSessionHoverTaskInlineTimeIndependentAndArchivedStatus(t *testing.T) {
	tt, m, _, start := sessionHoverDraftFixture(t)
	m.PendingStatuses = map[string]string{"a": "approval"}
	m.Schedules = schedstore.ViewState{ClientReady: true, CatalogSettled: true, Rows: []schedstore.Entry{
		{ID: "future", Title: "Future task", SessionID: "a", Status: "active", ScheduledAt: start.Add(time.Hour).Format(time.RFC3339)},
		{ID: "past", Title: "Overdue task", SessionID: "a", Status: "active", ScheduledAt: start.Add(-time.Hour).Format(time.RFC3339)},
		{ID: "later", Title: "Omitted task", SessionID: "a", Status: "active", ScheduledAt: start.Add(2 * time.Hour).Format(time.RFC3339)},
		{ID: "ended", Title: "Ended hidden", SessionID: "a", Status: "inactive"},
		{ID: "other", Title: "Other session hidden", SessionID: "b", Status: "active"},
	}}
	tt.Frame()
	hoverDraftRow(t, tt, "a")
	tt.SetTime(start.Add(800 * time.Millisecond))
	for _, text := range []string{"Original title A", "5min ago", "Overdue task", "Future task", "1 more", "Waiting for approval"} {
		if !tt.HasText(text) {
			t.Fatalf("unified body missing%q: %q", text, tt.Texts())
		}
	}
	for _, text := range []string{"Omitted task", "Ended hidden", "Other session hidden", "SHOULD-NOT-APPEAR"} {
		if tt.HasText(text) {
			t.Fatalf("unexpected%q", text)
		}
	}
	texts := strings.Join(tt.Texts(), "\n")
	if strings.Index(texts, "Overdue task") > strings.Index(texts, "Future task") || strings.Index(texts, "Waiting for approval") < strings.Index(texts, "Future task") {
		t.Fatal("task/status order differs")
	}
	m.Snapshot.ArchivedSessionIDs = []string{"a"}
	tt.Frame()
	if !tt.HasText("Waiting for approval") || !tt.HasText("Archived") {
		t.Fatal("archive suppressed live status")
	}
	m.PendingStatuses = nil
	m.Snapshot.Items[0].CompletionUnread = true
	tt.Frame()
	if tt.HasText("Completed") || tt.HasText("Idle") || !tt.HasText("Archived") {
		t.Fatal("archive did not suppress idle/done")
	}
}

func TestDraftSessionHoverSuppressionResetAndOriginalCopyData(t *testing.T) {
	for _, mode := range []string{"batch", "drag", "pressed descendant", "host", "removed", "collapsed"} {
		t.Run(mode, func(t *testing.T) {
			tt, m, in, start := sessionHoverDraftFixture(t)
			hoverDraftRow(t, tt, "a")
			tt.SetTime(start.Add(800 * time.Millisecond))
			if !tt.HasText("Session preview: a") {
				t.Fatal("not open")
			}
			switch mode {
			case "batch":
				m.Batch.Enabled = true
			case "drag":
				in.dragging = true
			case "pressed descendant":
				in.pressed["a"] = true
			case "host":
				m.Snapshot.WorkspaceLayout.HostGeneration++
			case "removed":
				m.Snapshot.Items = m.Snapshot.Items[1:]
			case "collapsed":
				m.Collapsed = true
			}
			tt.Frame()
			if tt.HasText("Session preview: a") {
				t.Fatalf("%s retained stale preview", mode)
			}
		})
	}
	for _, test := range []struct {
		item session.Item
		want string
	}{
		{session.Item{Title: "Raw title"}, "Raw title"}, {session.Item{Title: "  exact spaces  "}, "  exact spaces  "}, {session.Item{Title: "Fallback", Blank: true}, ""}, {session.Item{}, ""},
	} {
		if got := sessionHoverCopyTitle(test.item); got != test.want {
			t.Fatalf("copy=%q want=%q", got, test.want)
		}
	}
}

func TestSessionTitlePressObservationPreservesClickOwner(t *testing.T) {
	for _, renameEnabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "row owns title", true: "rename title owns click"}[renameEnabled], func(t *testing.T) {
			item := session.Item{SessionID: "actual", Title: "Actual selectable session title"}
			selected, renamed := 0, 0
			m := Model{Snapshot: session.ViewState{Items: []session.Item{item}}, OnSelect: func(id string) {
				if id != item.SessionID {
					t.Fatal(id)
				}
				selected++
			}}
			if renameEnabled {
				m.OnBeginRename = func(string, string) { renamed++ }
			}
			tt := ui.NewTester(func(c *ui.Context) {
				root := ui.Column(c).Width(280).FillHeight()
				m = beginSessionHover(root, m)
				root.Children(func() { rowSession(c, item, 0, m, c.Theme().Text, c.Theme().TextMuted, c.Theme().Border) })
				renderSessionHover(c, m, root, m.sessionHover.dragging)
			}, 640, 240)
			r, ok := tt.Find(item.Title)
			if !ok {
				t.Fatal("row missing")
			}
			// Hit actual title ink/lane; this must reach exactly one existing
			// select owner, whether rename support supplied a title owner or not.
			tt.ClickAt(r.X+70, r.Y+r.H/2)
			if selected != 1 || renamed != 0 {
				t.Fatalf("title selected=%d renamed=%d", selected, renamed)
			}
		})
	}
}

func TestSessionHoverSubagentActivityRequiresAuthoritativeKnownCount(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale("en")
	t.Cleanup(func() { i18n.SetLocale(previous) })
	item := session.Item{SessionID: "parent", Running: true}
	for _, test := range []struct {
		name     string
		activity session.SidebarSubagentActivity
		want     bool
	}{
		{"known positive", session.SidebarSubagentActivity{Known: true, Count: 2}, true},
		{"unknown positive must not leak", session.SidebarSubagentActivity{Known: false, Count: 99}, false},
		{"known zero", session.SidebarSubagentActivity{Known: true, Count: 0}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			m := Model{Snapshot: session.ViewState{SidebarSubagentActivity: map[string]session.SidebarSubagentActivity{item.SessionID: test.activity}}}
			statuses := sessionHoverStatuses(m, item, false)
			if len(statuses) == 0 || statuses[0].label != "Running" {
				t.Fatal("child activity replaced parent's own running state")
			}
			found := false
			for _, status := range statuses {
				if strings.Contains(status.label, "running subagents") {
					found = true
					if status.state != widgets.StateRunning || status.label != "2 running subagents" {
						t.Fatalf("wrong child activity: %+v", status)
					}
				}
			}
			if found != test.want {
				t.Fatalf("activity %+v visible=%v want=%v", test.activity, found, test.want)
			}
		})
	}
}

func TestDraftSessionHoverMenuSuppressesExistingCard(t *testing.T) {
	tt, _, in, start := sessionHoverDraftFixture(t)
	in.menu = true
	tt.Frame()
	hoverDraftRow(t, tt, "a")
	tt.SetTime(start.Add(800 * time.Millisecond))
	if err := tt.Click("Session row a"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if tt.HasText("Session preview: a") {
		t.Fatal("menu retained hover card")
	}
}
