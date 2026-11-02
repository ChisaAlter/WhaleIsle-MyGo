package sidebar

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	schedstore "whaleisle/internal/store/schedule"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func sidebarScheduleFixture() schedstore.ViewState {
	now := time.Now()
	rows := []schedstore.Entry{
		{ID: "future", Title: "Future task", ScheduledAt: now.Add(time.Hour).Format(time.RFC3339Nano)},
		{ID: "late", Title: "Recent overdue task", ScheduledAt: now.Add(-time.Minute).Format(time.RFC3339Nano)},
		{ID: "old", Title: "Oldest overdue task", ScheduledAt: now.Add(-time.Hour).Format(time.RFC3339Nano)},
	}
	for i := range rows {
		rows[i].Origin, rows[i].SessionID, rows[i].Status = "owned-host", "row", "active"
		rows[i].Raw = json.RawMessage(`{"kind":"every","everySeconds":600}`)
	}
	return schedstore.ViewState{ClientReady: true, CatalogSettled: true, ConnectionEpoch: 4, Origin: "owned-host", Rows: rows}
}

func TestScheduleMarkIsPassiveAndNeverReplacesAttention(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	item := session.Item{SessionID: "row", Title: "Scheduled session"}
	selected := 0
	m := Model{Schedules: sidebarScheduleFixture(), OnSelect: func(string) { selected++ }}
	tt := ui.NewTester(func(c *ui.Context) {
		root := ui.Column(c).Fill()
		m = beginSessionHover(root, m)
		root.Children(func() { rowSession(c, item, 0, m, c.Theme().Text, c.Theme().TextMuted, c.Theme().Border) })
		renderSessionHover(c, m, root, m.sessionHover.dragging)
	}, 300, 200)
	mark, ok := tt.Find("3 scheduled tasks")
	if !ok || mark.W != 16 || mark.H != 20 {
		t.Fatalf("missing compact idle mark: %+v", mark)
	}
	if err := tt.Click("3 scheduled tasks"); err != nil {
		t.Fatal(err)
	}
	if selected != 0 {
		t.Fatal("passive task mark activated its Session")
	}
	for _, status := range []string{"running", "completed", "approval", "question", "plan-review"} {
		item.Running, item.CompletionUnread = status == "running", status == "completed"
		m.PendingStatuses = map[string]string{"row": status}
		tt.Frame()
		if _, ok := tt.Find("3 scheduled tasks"); ok {
			t.Fatalf("task mark replaced %s attention", status)
		}
		if _, ok := tt.Find(statusLabel(status)); !ok {
			t.Fatalf("missing %s state", status)
		}
	}
	item.Running, item.CompletionUnread = false, false
	m.PendingStatuses = nil
	m.Schedules.Loading, m.Schedules.Err = true, "owned refresh failure"
	tt.Frame()
	if _, ok := tt.Find("3 scheduled tasks"); !ok {
		t.Fatal("refresh/error blanked settled idle mark")
	}
	m.Schedules.Origin = "new-host"
	tt.Frame()
	if _, ok := tt.Find("3 scheduled tasks"); ok {
		t.Fatal("old Host tasks survived a different current origin")
	}
}

func TestScheduleMarkUsesOriginalGlyphAndLocalizedSemanticLabel(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	defer i18n.SetLocale(old)
	tt := ui.NewTester(func(c *ui.Context) {
		sessionScheduleMark(c, 2, ui.RGB(207, 211, 214))
	}, 80, 50)
	mark, ok := tt.Find("2 个自动化任务")
	if !ok || mark.W != 16 || mark.H != 20 {
		t.Fatalf("wrong localized passive mark: %+v", mark)
	}
	if !strings.Contains(groupPresenceAccessLabels(tt), "2 个自动化任务") {
		_ = tt.SetValue("absent accessibility bootstrap field", "")
		if !strings.Contains(groupPresenceAccessLabels(tt), "2 个自动化任务") {
			t.Fatal("localized mark missing from memory accessibility tree")
		}
	}
	// The one decorative glyph stays out of keyboard traversal; its centered
	// 14DIP geometry is observed directly in the memory renderer.
	ctx := reflect.ValueOf(tt).Elem().FieldByName("rt").Elem().FieldByName("c")
	chunks, used := ctx.FieldByName("chunks"), int(ctx.FieldByName("used").Int())
	icons := 0
	for i := 0; i < used; i++ {
		chunk := chunks.Index(i / chunks.Index(0).Len())
		n := chunk.Index(i % chunk.Len())
		if n.FieldByName("svg").IsNil() {
			continue
		}
		icons++
		box := scheduleLayoutBox(n)
		if box.W != 14 || box.H != 14 || box.X != mark.X+1 || box.Y != mark.Y+3 {
			t.Fatalf("clock differs from original 14DIP centered glyph: %+v", box)
		}
	}
	if icons != 1 {
		t.Fatalf("passive mark has %d clock glyphs", icons)
	}
}

func TestScheduleHoverShowsTwoOverdueTasksAndOmission(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	item := session.Item{SessionID: "row", Title: "Scheduled session"}
	m := Model{Schedules: sidebarScheduleFixture()}
	tt := ui.NewTester(func(c *ui.Context) {
		root := ui.Column(c).Fill().Padding(30)
		m = beginSessionHover(root, m)
		root.Children(func() { rowSession(c, item, 0, m, c.Theme().Text, c.Theme().TextMuted, c.Theme().Border) })
		renderSessionHover(c, m, root, m.sessionHover.dragging)
	}, 400, 350)
	// The interactive row card opens through the source's 800ms pointer gate.
	// Its task section shares the full-title/status preview, not a second tooltip.
	start := time.Now()
	tt.SetTime(start)
	row, found := tt.Find(item.Title)
	if !found {
		t.Fatal("scheduled session row missing")
	}
	tt.Move(row.X+40, row.Y+row.H/2)
	tt.SetTime(start.Add(sessionHoverDelay))
	if !tt.HasText("Oldest overdue task") || !tt.HasText("Recent overdue task") || tt.HasText("Future task") || !tt.HasText("1 more") || !tt.HasText(item.Title) {
		t.Fatalf("unbounded or unordered preview: %v", tt.Texts())
	}
	preview, ok := tt.Find("Session preview: row")
	if !ok || preview.W > 376 || preview.H > 326 {
		t.Fatalf("preview outside viewport: %+v", preview)
	}
	m.Schedules.Rows = nil
	m.Schedules.Origin = "new-host"
	m.Schedules.ConnectionEpoch++
	tt.Frame()
	if tt.HasText("Oldest overdue task") {
		t.Fatal("old owner preview survived catalog/Host change")
	}
}

func TestSchedulePreviewUsesStableOrderAndDeviceClockDistance(t *testing.T) {
	now := time.Date(2026, 12, 1, 12, 0, 0, 0, time.UTC)
	rows := []schedstore.Entry{{ID: "future", ScheduledAt: now.Add(time.Hour).Format(time.RFC3339)}, {ID: "second", ScheduledAt: now.Add(-time.Minute).Format(time.RFC3339)}, {ID: "first", ScheduledAt: now.Add(-time.Minute).Format(time.RFC3339)}}
	ordered := orderSchedulePreview(rows, now)
	if ordered[0].ID != "second" || ordered[1].ID != "first" || ordered[2].ID != "future" || rows[0].ID != "future" {
		t.Fatal("overdue stable order mutated or reordered the source")
	}
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	if line := schedulePreviewTiming(rows[0], now); !strings.Contains(line, "in 1 hour") {
		t.Fatal(line)
	}
	if line := schedulePreviewTiming(rows[1], now); !strings.Contains(line, "1 minute overdue") {
		t.Fatal(line)
	}
}
