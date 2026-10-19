package frame

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/dsh"
	schedstore "whaleisle/internal/store/schedule"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

type scheduleHeaderTransport func(*http.Request) (*http.Response, error)

func (f scheduleHeaderTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestScheduleHeaderZeroOneManyOwnerAndPopoverBounds(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	origin := "https://header-schedule.fixture.invalid"
	row := schedstore.Entry{ID: "first", SessionID: "viewer", Origin: origin, Title: "Morning task", Status: "active", ScheduledAt: "2026-12-01T01:00:00Z", Raw: json.RawMessage(`{"kind":"every","everySeconds":600}`)}
	snap := schedstore.ViewState{ClientReady: true, Origin: origin}
	var opened []schedstore.Entry
	m := Model{OnOpenScheduleTask: func(row schedstore.Entry) { opened = append(opened, row) }}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			ui.Row(c).FillWidth().Height(40).Padding(4, 20).Clip().Children(func() {
				ui.Text(c, "Session header").Grow(1)
				scheduleHeaderView(c, m, snap, "viewer", theme.Resolve("deepseek.light").Light)
			})
			ui.Text(c, "Conversation body")
		})
	}, 900, 700)
	if _, ok := tt.Find("Scheduled tasks"); ok {
		t.Fatal("zero schedule trigger visible")
	}
	snap.Rows = []schedstore.Entry{row}
	tt.Frame()
	if err := tt.Click("1 scheduled task"); err != nil {
		t.Fatal(err)
	}
	if len(opened) != 1 || opened[0].ID != "first" || opened[0].SessionID != "viewer" {
		t.Fatalf("sole task not direct-open: %+v", opened)
	}
	if _, ok := tt.Find("Scheduled tasks popover"); ok {
		t.Fatal("sole task unnecessarily opened list")
	}
	second := row
	second.ID = "second"
	second.Title = strings.Repeat("Long task title ", 9)
	unrelated := row
	unrelated.ID = "foreign"
	unrelated.SessionID = "other"
	unrelated.Title = "Other session task"
	snap.Rows = []schedstore.Entry{row, second, unrelated}
	tt.Frame()
	if err := tt.Click("2 scheduled tasks"); err != nil {
		t.Fatal(err)
	}
	trigger, _ := tt.Find("2 scheduled tasks")
	popup, ok := tt.Find("Scheduled tasks popover")
	if !ok || popup.W != 336 || popup.H > 420 || popup.X < 16 || popup.X+popup.W > 884 || popup.Y < trigger.Y+trigger.H+5 {
		t.Fatalf("schedule overlay: %+v", popup)
	}
	if tt.HasText("Other session task") {
		t.Fatal("header listed foreign session task")
	}
	headerSnapshot(t, tt, "schedule-header-many")
	if err := tt.Click("Open scheduled task: Morning task"); err != nil {
		t.Fatal(err)
	}
	if len(opened) != 2 || opened[1].Origin != origin {
		t.Fatal("menu lost captured task identity")
	}
	snap.Loading = true
	tt.Frame()
	if err := tt.Click("2 scheduled tasks"); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.Find("Loading scheduled tasks"); !ok {
		t.Fatal("retained schedule loading state missing")
	}
	snap.Loading = false
	snap.Err = "Isolated schedule read failed"
	tt.Frame()
	if !tt.HasText("Retry schedules") || !tt.HasText("Morning task") {
		t.Fatal("failed refresh discarded retained menu")
	}
	headerSnapshot(t, tt, "schedule-header-error")
	tt.Key(0, ui.KeyEscape)
	snap.Err = ""
	tt.SetSize(360, 420)
	if err := tt.Click("2 scheduled tasks"); err != nil {
		t.Fatal(err)
	}
	popup, _ = tt.Find("Scheduled tasks popover")
	if popup.W != 328 || popup.X < 16 || popup.X+popup.W > 344 || popup.H > 280 {
		t.Fatalf("narrow schedule popup: %+v", popup)
	}
	trigger, _ = tt.Find("2 scheduled tasks")
	if trigger.W != 28 || trigger.H != 28 {
		t.Fatalf("schedule trigger: %+v", trigger)
	}
	headerSnapshot(t, tt, "schedule-header-narrow")
}

func TestScheduleHeaderDeletionUsesCapturedOwningSession(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	var mu sync.Mutex
	rows := []map[string]any{{"id": "first", "title": "First task", "prompt": "Do first thing", "sessionId": "owner", "status": "active", "kind": "every", "everySeconds": 600, "scheduledAt": "2026-12-01T01:00:00Z"}, {"id": "second", "title": "Second task", "prompt": "Do second thing", "sessionId": "owner", "status": "active", "kind": "every", "everySeconds": 600, "scheduledAt": "2026-12-01T02:00:00Z"}}
	requests := make(chan map[string]json.RawMessage, 4)
	notify := make(chan struct{}, 16)
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "https://schedule-header-delete.fixture.invalid"}}
	client.HTTP = &http.Client{Transport: scheduleHeaderTransport(func(r *http.Request) (*http.Response, error) {
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
		mu.Lock()
		defer mu.Unlock()
		var value any = rows
		if body.Method == "schedule/delete" {
			requests <- body.Payload.Args.Request
			rows = rows[1:]
			value = map[string]any{"id": "first", "deleted": true}
		} else if body.Method != "schedule/catalog" {
			t.Errorf("unexpected isolated method %s", body.Method)
		}
		raw, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": body.RPCID, "result": map[string]any{"ok": true, "value": value}})
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
	})}
	store := schedstore.New()
	store.Attach(client, func() {
		select {
		case notify <- struct{}{}:
		default:
		}
	})
	store.Refresh(context.Background(), "different-viewer")
	wait := func(ready func() bool) {
		t.Helper()
		timer := time.NewTimer(2 * time.Second)
		defer timer.Stop()
		for !ready() {
			select {
			case <-notify:
			case <-timer.C:
				t.Fatal("isolated schedule action did not settle")
			}
		}
	}
	wait(func() bool { return !store.Snapshot().Loading })
	m := Model{ScheduleStore: store, OnOpenScheduleTask: func(schedstore.Entry) {}}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Row(c).Padding(8).Children(func() { scheduleHeaderView(c, m, store.Snapshot(), "owner", theme.Resolve("deepseek.light").Light) })
	}, 600, 400)
	if err := tt.Click("2 scheduled tasks"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Delete scheduled task: First task"); err != nil {
		t.Fatal(err)
	}
	select {
	case request := <-requests:
		if string(request["sessionId"]) != `"owner"` || string(request["id"]) != `"first"` {
			t.Fatalf("wrong delete target: %v", request)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("isolated delete missing")
	}
	wait(func() bool { return !store.Snapshot().Loading && len(store.Snapshot().Rows) == 1 })
	tt.Frame()
	if tt.HasText("First task") {
		t.Fatal("deleted task stayed in header menu")
	}
}
