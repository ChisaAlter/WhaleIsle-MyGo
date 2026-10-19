package schedule

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

type catalogTransport func(*http.Request) (*http.Response, error)

func (f catalogTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// Pure transport fixtures: never open a socket or connect to a Host.
func TestCatalogControlsOwnerRefreshScopeAndPresentation(t *testing.T) {
	oldRecord := `{"id":"job","title":"Morning","prompt":"Check updates","kind":"daily","time":"09:00:00.000","timeZone":"Asia/Shanghai","scheduledAt":"2026-10-16T01:00:00Z","sessionId":"owner","status":"active"}`
	newRecord := strings.Replace(oldRecord, "2026-10-16", "2026-10-17", 1)
	var count atomic.Int32
	var catalogs, lists atomic.Int32
	var fallback atomic.Bool
	var failAll atomic.Bool
	var blockNext atomic.Bool
	blocked, release := make(chan struct{}), make(chan struct{})
	notified := make(chan struct{}, 32)
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "https://fixture.invalid"}}
	client.HTTP = &http.Client{Transport: catalogTransport(func(r *http.Request) (*http.Response, error) {
		var body struct {
			RPCID   string `json:"rpcId"`
			Method  string `json:"method"`
			Payload struct {
				Args struct {
					Request struct {
						SessionID string `json:"sessionId"`
					} `json:"request"`
				} `json:"args"`
			} `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		count.Add(1)
		if body.Method == "schedule/catalog" {
			catalogs.Add(1)
		} else if body.Method == "schedule/list" {
			lists.Add(1)
		}
		result := map[string]any{"ok": true, "value": []json.RawMessage{json.RawMessage(newRecord)}}
		if blockNext.CompareAndSwap(true, false) {
			close(blocked)
			<-release
			result["value"] = []json.RawMessage{json.RawMessage(oldRecord)}
		}
		if body.Method == "schedule/catalog" && fallback.Load() {
			result = map[string]any{"ok": false, "error": map[string]any{"code": "unsupported", "message": "catalog unavailable"}}
		}
		if body.Method == "schedule/list" {
			if !failAll.Load() && body.Payload.Args.Request.SessionID != "selected" {
				t.Errorf("wrong fallback owner: %q", body.Payload.Args.Request.SessionID)
			}
			result["value"] = []json.RawMessage{json.RawMessage(strings.Replace(newRecord, `,"sessionId":"owner","status":"active"`, "", 1))}
		} else if body.Method != "schedule/catalog" {
			t.Errorf("unexpected RPC %s", body.Method)
		}
		if failAll.Load() {
			result = map[string]any{"ok": false, "error": map[string]any{"code": "offline", "message": "fixture failure"}}
		}
		raw, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": body.RPCID, "result": result})
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
	})}
	notify := func() {
		select {
		case notified <- struct{}{}:
		default:
		}
	}
	s := New()
	s.Attach(client, notify)
	wait := func(store *Store) ViewState {
		t.Helper()
		deadline := time.After(2 * time.Second)
		for {
			snap := store.Snapshot()
			if !snap.Loading {
				return snap
			}
			select {
			case <-notified:
			case <-deadline:
				t.Fatal("fixture refresh did not settle")
			}
		}
	}
	owner := s.BeginEvents(context.Background(), client)
	s.OnEventsReady(owner)
	if count.Load() != 0 {
		t.Fatal("unopened catalog fetched in background")
	}
	s.Refresh(context.Background(), "selected")
	snap := wait(s)
	if len(snap.Rows) != 1 || snap.Rows[0].SessionID != "owner" || snap.Rows[0].Status != "active" {
		t.Fatalf("catalog: %+v", snap)
	}
	before := count.Load()
	s.OnEmit(owner, dsh.EventsEmit{Type: dsh.EventEmit, Event: "schedule/change"})
	if count.Load() != before {
		t.Fatal("session history event treated as Host notification")
	}
	blockNext.Store(true)
	s.OnEmit(owner, dsh.EventsEmit{Type: dsh.EventEmit, Event: "schedule/changed"})
	select {
	case <-blocked:
	case <-time.After(2 * time.Second):
		t.Fatal("changed notification did not read catalog")
	}
	nextOwner := s.BeginEvents(context.Background(), client)
	s.OnEventsReady(nextOwner)
	snap = wait(s)
	if !strings.Contains(snap.Rows[0].ScheduledAt, "2026-10-17") {
		t.Fatalf("newest catalog missing: %+v", snap)
	}
	close(release)
	before = count.Load()
	s.OnEmit(owner, dsh.EventsEmit{Type: dsh.EventEmit, Event: "schedule/changed"})
	if count.Load() != before {
		t.Fatal("old stream dispatched refresh")
	}
	ctx, cancel := context.WithCancel(context.Background())
	canceledOwner := s.BeginEvents(ctx, client)
	cancel()
	s.OnEventsReady(canceledOwner)
	if count.Load() != before {
		t.Fatal("canceled stream dispatched refresh")
	}
	// A settled global source retains its complete answer and scope on failure;
	// it must not replace that answer with a partial selected-Session fallback.
	global := snap
	if global.Err != "" || !global.CatalogReady || !global.CatalogSettled || global.CatalogRevision == 0 ||
		!strings.Contains(global.CatalogHint, "All sessions") || len(global.Rows) != 1 ||
		global.Rows[0].SessionID != "owner" || global.Rows[0].Status != "active" ||
		global.Rows[0].Origin != client.Endpoint.Origin || global.Rows[0].ID != "job" ||
		global.Rows[0].Title != "Morning" || string(global.Rows[0].Raw) != newRecord || lists.Load() != 0 {
		t.Fatalf("global source did not settle authoritative fixture facts: %+v", global)
	}
	beforeCatalog, beforeList := catalogs.Load(), lists.Load()
	fallback.Store(true)
	s.Refresh(context.Background(), "selected")
	snap = wait(s)
	if !strings.Contains(snap.Err, "unsupported") || !strings.Contains(snap.Err, "catalog unavailable") ||
		snap.CatalogReady || !snap.CatalogSettled || !reflect.DeepEqual(snap.Rows, global.Rows) ||
		snap.CatalogHint != global.CatalogHint || !strings.Contains(snap.CatalogHint, "All sessions") ||
		strings.Contains(snap.CatalogHint, "Current session only") || snap.CatalogRevision != global.CatalogRevision ||
		snap.CatalogRequest != global.CatalogRequest+1 || catalogs.Load() != beforeCatalog+1 || lists.Load() != beforeList {
		t.Fatalf("failed catalog replaced settled global scope/facts or queried a partial list: %+v catalog=%d list=%d", snap, catalogs.Load()-beforeCatalog, lists.Load()-beforeList)
	}
	beforeCatalog, beforeList = catalogs.Load(), lists.Load()
	failAll.Store(true)
	s.Refresh(context.Background(), "another-session")
	snap = wait(s)
	if !strings.Contains(snap.Err, "offline") || snap.CatalogReady || !snap.CatalogSettled ||
		!reflect.DeepEqual(snap.Rows, global.Rows) || snap.CatalogHint != global.CatalogHint ||
		snap.CatalogRevision != global.CatalogRevision || snap.CatalogRequest != global.CatalogRequest+2 ||
		catalogs.Load() != beforeCatalog+1 || lists.Load() != beforeList {
		t.Fatalf("failed read under another Session relabeled global data: %+v catalog=%d list=%d", snap, catalogs.Load()-beforeCatalog, lists.Load()-beforeList)
	}
	other := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "https://other.invalid"}, HTTP: client.HTTP}
	s.Attach(other, nil)
	s.OnEmit(nextOwner, dsh.EventsEmit{Type: dsh.EventEmit, Event: "schedule/changed"})
	if len(s.Snapshot().Rows) != 0 || s.Snapshot().Loading || s.Snapshot().CatalogSettled {
		t.Fatal("old Host repopulated catalog")
	}
	// Compatibility fallback is a separate first-read case, before this source
	// has ever settled a global catalog. Its rows must stay labeled as partial.
	fresh := New()
	fresh.Attach(other, notify)
	failAll.Store(false)
	beforeCatalog, beforeList = catalogs.Load(), lists.Load()
	fresh.Refresh(context.Background(), "selected")
	partial := wait(fresh)
	if partial.Err != "" || partial.CatalogReady || partial.CatalogSettled || partial.CatalogRevision != 0 ||
		partial.CatalogRequest != 1 || !strings.Contains(partial.CatalogHint, "Current session only: selected") ||
		strings.Contains(partial.CatalogHint, "All sessions") || len(partial.Rows) != 1 ||
		partial.Rows[0].SessionID != "selected" || partial.Rows[0].Status != "active" ||
		partial.Rows[0].Origin != other.Endpoint.Origin || partial.Rows[0].ID != "job" || partial.Rows[0].Title != "Morning" ||
		catalogs.Load() != beforeCatalog+1 || lists.Load() != beforeList+1 {
		t.Fatalf("first-read fallback lost its partial owner/status or request route: %+v catalog=%d list=%d", partial, catalogs.Load()-beforeCatalog, lists.Load()-beforeList)
	}
	beforeCatalog, beforeList = catalogs.Load(), lists.Load()
	failAll.Store(true)
	fresh.Refresh(context.Background(), "another-session")
	failedPartial := wait(fresh)
	if !strings.Contains(failedPartial.Err, "offline") || failedPartial.CatalogReady || failedPartial.CatalogSettled ||
		failedPartial.CatalogRevision != 0 || failedPartial.CatalogRequest != partial.CatalogRequest+1 ||
		!reflect.DeepEqual(failedPartial.Rows, partial.Rows) || failedPartial.CatalogHint != partial.CatalogHint ||
		catalogs.Load() != beforeCatalog+1 || lists.Load() != beforeList+1 {
		t.Fatalf("failed fallback relabeled retained selected-Session data: %+v catalog=%d list=%d", failedPartial, catalogs.Load()-beforeCatalog, lists.Load()-beforeList)
	}
	rows := parseScheduleList(api.ListResult4{json.RawMessage(newRecord), json.RawMessage(strings.ReplaceAll(strings.ReplaceAll(oldRecord, `"active"`, `"inactive"`), `"job"`, `"ended"`))})
	if got := FilterEntries(rows, "check UPDATES", "active"); len(got) != 1 || got[0].ID != "job" {
		t.Fatalf("filter: %+v", got)
	}
	if len(FilterEntries(rows, "owner", "inactive")) != 1 || len(FilterEntries(rows, "absent", "all")) != 0 {
		t.Fatal("session/status query failed")
	}
	if FilterEntries(rows, "", "all")[0].ID != "ended" || rows[0].ID != "job" {
		t.Fatal("sort did not preserve source rows")
	}
	label := TimingLabel(rows[0])
	if !strings.Contains(label, "09:00:00 +08:00") || !strings.Contains(label, "Asia/Shanghai") || !strings.Contains(label, "Next run") {
		t.Fatalf("time zone label: %s", label)
	}
	if strings.Contains(TimingLabel(rows[1]), "Next run") || !strings.Contains(TimingLabel(rows[1]), "Ended") {
		t.Fatal("inactive task has a next run")
	}
	invalid := rows[0]
	invalid.ScheduledAt = "bad"
	if strings.Contains(TimingLabel(invalid), "0001") {
		t.Fatal("invalid time invented a date")
	}
}
