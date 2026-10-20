package schedule

import (
	"context"
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/dsh"
	schedstore "whaleisle/internal/store/schedule"
	"whaleisle/internal/ui/i18n"
)

type scheduleUITransport func(*http.Request) (*http.Response, error)

func (f scheduleUITransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type scheduleUIFixture struct {
	mu            sync.Mutex
	rows          []map[string]any
	writes        []map[string]json.RawMessage
	methods       []string
	historyOwners []string
	failRead      bool
	holdRead      <-chan struct{}
	store         *schedstore.Store
	notified      chan struct{}
}

func newScheduleUIFixture(t *testing.T) *scheduleUIFixture {
	t.Helper()
	oldLocale := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(oldLocale) })
	f := &scheduleUIFixture{notified: make(chan struct{}, 32), rows: []map[string]any{
		{"id": "first", "sessionId": "owner-first", "title": "Morning check", "prompt": "Review the morning items", "kind": "every", "everySeconds": 600, "scheduledAt": "2026-12-01T01:00:00Z", "status": "active"},
		{"id": "second", "sessionId": "owner-second", "title": "Evening check", "prompt": "Review the evening items", "kind": "daily", "time": "21:00:00", "timeZone": "UTC", "scheduledAt": "2026-12-02T21:00:00Z", "status": "inactive"},
	}}
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "https://schedule-ui.fixture.invalid"}}
	client.HTTP = &http.Client{Transport: scheduleUITransport(func(r *http.Request) (*http.Response, error) {
		var body struct {
			RPCID   string `json:"rpcId"`
			Method  string `json:"method"`
			Payload struct {
				Args struct {
					Request map[string]json.RawMessage `json:"request"`
				} `json:"args"`
			} `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			return nil, err
		}
		f.mu.Lock()
		hold := f.holdRead
		f.mu.Unlock()
		if hold != nil && (body.Method == "schedule/catalog" || body.Method == "schedule/list") {
			<-hold
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		f.methods = append(f.methods, body.Method)
		request := body.Payload.Args.Request
		var value any
		var result map[string]any
		switch body.Method {
		case "schedule/catalog":
			value = f.rows
		case "schedule/list":
			value = f.rows
		case "schedule/history":
			f.historyOwners = append(f.historyOwners, string(request["sessionId"]))
			value = map[string]any{"id": "first", "records": []map[string]any{{"messageId": "receipt-first", "scheduledAt": "2026-12-01T01:00:00Z", "deliveredAt": "2026-12-01T01:00:01Z", "prompt": "Saved instruction from the exact owning session"}}}
		case "schedule/update":
			f.writes = append(f.writes, request)
			for _, row := range f.rows {
				if string(request["id"]) != fmt.Sprintf("%q", row["id"]) {
					continue
				}
				for _, field := range []string{"title", "prompt"} {
					if raw := request[field]; len(raw) > 0 {
						var text string
						_ = json.Unmarshal(raw, &text)
						row[field] = text
					}
				}
				value = map[string]any{"id": row["id"], "updated": true, "record": row}
			}
		case "schedule/delete":
			f.writes = append(f.writes, request)
			value = map[string]any{"id": "first", "deleted": true}
			f.rows = f.rows[1:]
		default:
			return nil, fmt.Errorf("unexpected fixture RPC %s", body.Method)
		}
		result = map[string]any{"ok": true, "value": value}
		if f.failRead && (body.Method == "schedule/catalog" || body.Method == "schedule/list") {
			result = map[string]any{"ok": false, "error": map[string]any{"code": "fixture_offline", "message": "Isolated read failed"}}
		}
		raw, err := json.Marshal(map[string]any{"type": "server-response", "rpcId": body.RPCID, "result": result})
		if err != nil {
			return nil, err
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
	})}
	f.store = schedstore.New()
	f.store.Attach(client, func() {
		select {
		case f.notified <- struct{}{}:
		default:
		}
	})
	f.store.Refresh(context.Background(), "viewer")
	f.wait(t, func() bool { return !f.store.Snapshot().Loading })
	return f
}

func (f *scheduleUIFixture) wait(t *testing.T, ready func() bool) {
	t.Helper()
	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()
	for !ready() {
		select {
		case <-f.notified:
		case <-deadline.C:
			t.Fatal("isolated schedule fixture did not settle")
		}
	}
}

func (f *scheduleUIFixture) panel() (*Panel, Model) {
	p := &Panel{initialized: true, connectionEpoch: f.store.Snapshot().ConnectionEpoch, statusFilter: "all", recentZones: []string{"UTC"}}
	return p, Model{Store: f.store, SessionID: "viewer", ThemeID: "deepseek.light", SessionLinks: map[string]string{"owner-first": "Original morning session"}, OnNewTask: func() {}, OnInvalidate: func() {
		select {
		case f.notified <- struct{}{}:
		default:
		}
	}}
}

func saveScheduleSnapshot(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	dir := surfaceGalleryDir(os.Getenv("SECONDARY_SNAPSHOT_DIR"))
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(filepath.Join(dir, "schedule-"+name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, tt.Image()); err != nil {
		t.Fatal(err)
	}
}

func TestScheduleCatalogDetailGeometryAndLazyRecords(t *testing.T) {
	f := newScheduleUIFixture(t)
	p, m := f.panel()
	tt := ui.NewTester(func(c *ui.Context) { p.View(c, m) }, 1280, 720)
	content, ok := tt.Find("Task catalog content")
	if !ok || content.W != 960 || content.X != 160 || content.Y != 0 {
		t.Fatalf("catalog frame: %+v %v", content, ok)
	}
	saveScheduleSnapshot(t, tt, "catalog")
	if err := tt.Click("Open task: Morning check"); err != nil {
		t.Fatal(err)
	}
	detail, ok := tt.Find("Task details")
	if !ok || detail.W < 600 || detail.W > 603 || detail.Y != 0 || detail.H != 720 {
		t.Fatalf("47%% detail: %+v %v", detail, ok)
	}
	strip, _ := tt.Find("Task detail views")
	if strip.H != 44 {
		t.Fatalf("detail strip: %+v", strip)
	}
	saveScheduleSnapshot(t, tt, "detail")
	f.mu.Lock()
	reads := len(f.historyOwners)
	f.mu.Unlock()
	if reads != 0 {
		t.Fatal("history read before Records opened")
	}
	tt.SetSize(600, 450)
	if _, ok := tt.Find("Task catalog scroll"); ok {
		t.Fatal("catalog stayed beside narrow detail")
	}
	detail, _ = tt.Find("Task details")
	if detail.W != 600 || detail.H != 450 || detail.Y != 0 {
		t.Fatalf("narrow detail: %+v", detail)
	}
	saveScheduleSnapshot(t, tt, "narrow-detail")
	if err := tt.Click("Records"); err != nil {
		t.Fatal(err)
	}
	f.wait(t, func() bool { p.mu.Lock(); defer p.mu.Unlock(); return p.historyState.Loaded })
	tt.Frame()
	if !tt.HasText("Saved instruction from the exact owning session") {
		t.Fatalf("records missing: %v", tt.Texts())
	}
	saveScheduleSnapshot(t, tt, "records")
	if _, ok := tt.Find("Automation task instruction"); ok {
		t.Fatal("Rules editor stayed mounted in Records")
	}
	records, ok := tt.Find("Task records scroll")
	if !ok || records.Y < 37 || records.Y+records.H > 451 {
		t.Fatalf("records scroll escaped detail: %+v", records)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.historyOwners) != 1 || f.historyOwners[0] != `"owner-first"` {
		t.Fatalf("history ownership: %v", f.historyOwners)
	}
}

func TestScheduleDraftRetainedAndEndedRuleReadable(t *testing.T) {
	f := newScheduleUIFixture(t)
	p, m := f.panel()
	tt := ui.NewTester(func(c *ui.Context) { p.View(c, m) }, 1000, 700)
	if err := tt.Click("Open task: Morning check"); err != nil {
		t.Fatal(err)
	}
	d := p.drafts[p.selected]
	d.Prompt = "My retained instruction"
	tt.Frame()
	if _, ok := tt.Find("Unsaved task changes"); !ok {
		t.Fatal("dirty save footer missing")
	}
	if err := tt.Click("Open task: Evening check"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Every day 21:00:00") || !tt.HasText("Review the evening items") {
		t.Fatalf("ended rule hidden: %v", tt.Texts())
	}
	saveScheduleSnapshot(t, tt, "inactive-detail")
	if _, ok := tt.Find("Automation task instruction"); ok {
		t.Fatal("ended instruction is still an editor")
	}
	if _, ok := tt.Find("Unsaved task changes"); ok {
		t.Fatal("ended task has dirty footer")
	}
	if err := tt.Click("Open task: Morning check"); err != nil {
		t.Fatal(err)
	}
	if p.drafts[p.selected].Prompt != "Review the morning items" {
		t.Fatal("task switch restored a draft belonging to the previous detail mount")
	}
	p.drafts[p.selected].Prompt = "Cancel this later edit"
	tt.Frame()
	if err := tt.Click("Cancel task changes"); err != nil {
		t.Fatal(err)
	}
	if p.drafts[p.selected].Prompt != "Review the morning items" {
		t.Fatal("Cancel did not restore committed instruction")
	}
	if _, ok := tt.Find("Unsaved task changes"); ok {
		t.Fatal("clean draft retained save footer")
	}
}

func TestCatalogSearchAndStatusSurviveSelectionAndNarrowClose(t *testing.T) {
	f := newScheduleUIFixture(t)
	p, m := f.panel()
	tt := ui.NewTester(func(c *ui.Context) { p.View(c, m) }, 1000, 700)
	if err := tt.Click("Search tasks"); err != nil {
		t.Fatal(err)
	}
	tt.Type("morning")
	if err := tt.Click("Enabled"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Open task: Morning check"); err != nil {
		t.Fatal(err)
	}
	if p.query != "morning" || p.statusFilter != "active" {
		t.Fatalf("select reset catalog context: query=%q status=%q", p.query, p.statusFilter)
	}
	if _, ok := tt.Find("Open task: Evening check"); ok {
		t.Fatal("selection revealed unrelated filtered row")
	}
	selected := p.selected
	draft := p.drafts[selected]
	draft.Prompt = "Retained filtered task draft"
	tt.Frame()
	tt.SetSize(600, 450)
	if err := tt.Click("Close details"); err != nil {
		t.Fatal(err)
	}
	if p.query != "morning" || p.statusFilter != "active" {
		t.Fatalf("narrow close reset catalog context: query=%q status=%q", p.query, p.statusFilter)
	}
	if _, ok := tt.Find("Open task: Morning check"); !ok {
		t.Fatal("close did not restore filtered catalog")
	}
	if _, ok := tt.Find("Open task: Evening check"); ok {
		t.Fatal("close revealed unrelated filtered row")
	}
	if err := tt.Click("Open task: Morning check"); err != nil {
		t.Fatal(err)
	}
	if p.drafts[p.selected] == draft || p.drafts[p.selected].Prompt != "Review the morning items" || p.drafts[p.selected].row.SessionID != "owner-first" {
		t.Fatal("reopening did not seed the current owner's committed rule")
	}
	if err := tt.Click("Close details"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Inactive"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("No matching tasks") {
		t.Fatal("retained search was not applied to the changed status")
	}
	if err := tt.Click("Clear search"); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.Find("Open task: Evening check"); !ok {
		t.Fatal("clearing retained search did not reveal inactive task")
	}
}

func TestScheduleSaveAndDeleteKeepCapturedOwner(t *testing.T) {
	f := newScheduleUIFixture(t)
	p, m := f.panel()
	tt := ui.NewTester(func(c *ui.Context) { p.View(c, m) }, 1000, 700)
	if err := tt.Click("Open task: Morning check"); err != nil {
		t.Fatal(err)
	}
	p.drafts[p.selected].Title = "Updated morning"
	tt.Frame()
	if err := tt.Click("Save changes"); err != nil {
		t.Fatal(err)
	}
	f.wait(t, func() bool { p.mu.Lock(); defer p.mu.Unlock(); return len(p.committed) > 0 })
	tt.Frame()
	f.mu.Lock()
	if len(f.writes) != 1 || string(f.writes[0]["sessionId"]) != `"owner-first"` || string(f.writes[0]["id"]) != `"first"` {
		t.Fatalf("save owner: %v", f.writes)
	}
	f.mu.Unlock()
	if err := tt.Click("Automation task actions"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("Delete task"); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.Find("Delete this task?"); !ok {
		t.Fatal("scoped delete dialog missing")
	}
	saveScheduleSnapshot(t, tt, "delete")
	if err := tt.Click("Cancel task deletion"); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	writes := len(f.writes)
	f.mu.Unlock()
	if writes != 1 {
		t.Fatal("Cancel sent a delete")
	}
	if err := tt.Click("Automation task actions"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("Delete task"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Confirm deletion"); err != nil {
		t.Fatal(err)
	}
	f.wait(t, func() bool { p.mu.Lock(); defer p.mu.Unlock(); return p.deleteAck != nil })
	f.wait(t, func() bool { snap := f.store.Snapshot(); return snap.CatalogReady && len(snap.Rows) == 1 })
	tt.Frame()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.writes) != 2 || string(f.writes[1]["sessionId"]) != `"owner-first"` {
		t.Fatalf("delete owner: %v", f.writes)
	}
	if _, ok := tt.Find("Task details"); ok {
		t.Fatal("deleted task detail stayed open")
	}
}

func TestScheduleEmptyLoadingFailureAndTaskTabChrome(t *testing.T) {
	f := newScheduleUIFixture(t)
	p, m := f.panel()
	closed := 0
	m.DetailID = "first"
	m.DetailSessionID = "owner-first"
	m.OnClose = func() { closed++ }
	tt := ui.NewTester(func(c *ui.Context) { p.View(c, m) }, 360, 300)
	strip, ok := tt.Find("Task detail views")
	if !ok || strip.H != 37 || strip.W > 360 {
		t.Fatalf("task-tab strip: %+v", strip)
	}
	if err := tt.Click("Close task"); err != nil {
		t.Fatal(err)
	}
	if closed != 1 {
		t.Fatal("task close callback changed")
	}
	m.DetailID = ""
	m.DetailSessionID = ""
	p.selected = ""
	f.mu.Lock()
	f.rows = nil
	f.mu.Unlock()
	f.store.Refresh(context.Background(), "viewer")
	f.wait(t, func() bool { return !f.store.Snapshot().Loading })
	tt.Frame()
	if !tt.HasText("No tasks yet.") {
		t.Fatalf("catalog empty state: %v", tt.Texts())
	}
	if err := tt.Click("Inactive"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("No inactive automation tasks") {
		t.Fatal("inactive-only empty state missing")
	}
	release := make(chan struct{})
	f.mu.Lock()
	f.holdRead = release
	f.mu.Unlock()
	f.store.Refresh(context.Background(), "viewer")
	tt.Frame()
	if _, ok := tt.Find("Loading tasks"); !ok {
		close(release)
		t.Fatal("initial catalog loading feedback missing")
	}
	close(release)
	f.wait(t, func() bool { return !f.store.Snapshot().Loading })
	f.mu.Lock()
	f.holdRead = nil
	f.mu.Unlock()
	f.mu.Lock()
	f.failRead = true
	f.mu.Unlock()
	f.store.Refresh(context.Background(), "viewer")
	f.wait(t, func() bool { return !f.store.Snapshot().Loading })
	tt.Frame()
	if !tt.HasText("Could not load tasks.") {
		t.Fatal("failed catalog has no recovery state")
	}
	f.mu.Lock()
	f.failRead = false
	f.mu.Unlock()
	if err := tt.Click("Retry"); err != nil {
		t.Fatal(err)
	}
	f.wait(t, func() bool { return !f.store.Snapshot().Loading })
	tt.Frame()
	if tt.HasText("Could not load tasks.") {
		t.Fatal("Retry did not recover")
	}
}

func TestScheduleCalendarFitsNarrowDetail(t *testing.T) {
	d := newTaskDraft(schedstore.Entry{Raw: json.RawMessage(`{"kind":"at","scheduledAt":"2026-12-01T01:00:00Z"}`)})
	d.picker.calendarOpen = true
	d.picker.month = time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	p := &Panel{recentZones: []string{"UTC"}}
	tt := ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Padding(12).Children(func() { p.datePicker(c, d) }) }, 280, 500)
	for _, label := range []string{"Mon", "Sun", "31"} {
		box, ok := tt.Find(label)
		if !ok || box.X < 12 || box.X+box.W > 269 {
			t.Fatalf("calendar %s escaped narrow detail: %+v", label, box)
		}
	}
}

func TestSavedPromptExpansionFollowsMeasuredOverflow(t *testing.T) {
	f := newScheduleUIFixture(t)
	p, m := f.panel()
	text := strings.Repeat("汉", 65)
	p.historyKey, p.historyRow = "fixture", f.store.Snapshot().Rows[0]
	p.historyState = deliveryHistoryState{Loaded: true, Page: schedstore.DeliveryPage{Records: []schedstore.DeliveryRecord{{MessageID: "short-wrapped", Prompt: &text}}}}
	tt := ui.NewTester(func(c *ui.Context) { p.deliveryHistoryView(c, m, ui.RGBA(0, 0, 0, 1), ui.RGBA(128, 128, 128, 1)) }, 280, 400)
	if err := tt.Click("Expand"); err != nil {
		t.Fatalf("short prompt clipped without expansion: %v", err)
	}
	box, _ := tt.Find(text)
	if box.H <= 44 {
		t.Fatalf("expanded prompt stayed clamped: %+v", box)
	}
	tt.SetSize(1600, 400)
	if err := tt.Click("Collapse"); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.Find("Expand"); ok {
		t.Fatal("prompt that fits still offers expansion")
	}
}

// surfaceGalleryDir routes a surface snapshot to the evidence gallery when
// SURFACES_GALLERY_DIR is set, so the same test writes documentation images
// without a second harness.
func surfaceGalleryDir(fallback string) string {
	if dir := os.Getenv("SURFACES_GALLERY_DIR"); dir != "" {
		return dir
	}
	return fallback
}
// saveSurfaceSnapshot writes the rendered surface for the evidence gallery when
// SURFACES_GALLERY_DIR is set.
func saveSurfaceSnapshot(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	dir := os.Getenv("SURFACES_GALLERY_DIR")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, tt.Image()); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}
// TestScheduleSurfaceSnapshot renders the task manager for
// evidence/after/surface-schedule.png.
func TestScheduleSurfaceSnapshot(t *testing.T) {
	f := newScheduleUIFixture(t)
	p, m := f.panel()
	tt := ui.NewTester(func(c *ui.Context) { p.View(c, m) }, 1000, 700)
	saveSurfaceSnapshot(t, tt, "surface-schedule.png")
}