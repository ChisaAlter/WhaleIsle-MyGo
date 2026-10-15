package schedule

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
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
	s := New()
	s.Attach(client, func() {
		select {
		case notified <- struct{}{}:
		default:
		}
	})
	wait := func() ViewState {
		t.Helper()
		deadline := time.After(2 * time.Second)
		for {
			snap := s.Snapshot()
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
	snap := wait()
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
	snap = wait()
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
	fallback.Store(true)
	s.Refresh(context.Background(), "selected")
	snap = wait()
	if !strings.Contains(snap.CatalogHint, "Current session only") || snap.Rows[0].SessionID != "selected" || snap.Rows[0].Status != "active" {
		t.Fatalf("unlabeled or inactive fallback: %+v", snap)
	}
	previousHint := snap.CatalogHint
	failAll.Store(true)
	s.Refresh(context.Background(), "another-session")
	snap = wait()
	if snap.Err == "" || snap.CatalogHint != previousHint || len(snap.Rows) != 1 || snap.Rows[0].SessionID != "selected" {
		t.Fatalf("failed read relabeled retained data: %+v", snap)
	}
	other := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "https://other.invalid"}, HTTP: client.HTTP}
	s.Attach(other, nil)
	s.OnEmit(nextOwner, dsh.EventsEmit{Type: dsh.EventEmit, Event: "schedule/changed"})
	if len(s.Snapshot().Rows) != 0 || s.Snapshot().Loading {
		t.Fatal("old Host repopulated catalog")
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
