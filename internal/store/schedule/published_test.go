package schedule

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/errcopy"
	"whaleisle/internal/state"
)

// scheduleFixture answers every schedule RPC through handle; a nil value
// with a non-nil error fails the HTTP exchange itself.
func scheduleFixture(handle func(r *http.Request, method string) (any, error)) *dsh.Client {
	return &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://schedule-published.invalid"}, HTTP: &http.Client{Transport: epochRoundTrip(func(r *http.Request) (*http.Response, error) {
		var request struct {
			RPCID string `json:"rpcId"`
		}
		_ = json.NewDecoder(r.Body).Decode(&request)
		value, err := handle(r, strings.TrimPrefix(r.URL.Path, "/api/schedule/"))
		if err != nil {
			return nil, err
		}
		raw, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": request.RPCID, "result": map[string]any{"ok": true, "value": value}})
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
	})}}
}

func catalogRecord() []any {
	return []any{map[string]any{"id": "job", "title": "Morning", "sessionId": "owner", "status": "active", "kind": "at", "prompt": "p", "scheduledAt": "2026-12-01T01:00:00Z"}}
}

func waitSettled(t *testing.T, s *Store) *state.Version[ViewState] {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if v := s.Snapshot(); !v.Value.Loading {
			return v
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("schedule read did not settle")
	return nil
}

func TestScheduleSnapshotSharedUntilWrite(t *testing.T) {
	s := New()
	s.Attach(scheduleFixture(func(*http.Request, string) (any, error) { return catalogRecord(), nil }), nil)
	s.Refresh(context.Background(), "owner")
	first := waitSettled(t, s)
	if first != s.Snapshot() {
		t.Fatal("unchanged state must hand out the same *Version")
	}
	if got := testing.AllocsPerRun(100, func() { s.Snapshot() }); got != 0 {
		t.Fatalf("Snapshot allocates %g per run", got)
	}
	if len(first.Value.Rows) != 1 || &first.Value.Rows[0] != &s.Snapshot().Value.Rows[0] {
		t.Fatal("rows must be shared, not copied per read")
	}
}

func TestScheduleWritePublishesAndSignals(t *testing.T) {
	s := New()
	before := s.Snapshot()
	changed := s.Changed()
	s.Attach(scheduleFixture(func(*http.Request, string) (any, error) { return catalogRecord(), nil }), nil)
	after := s.Snapshot()
	if after == before || after.Rev != before.Rev+1 || !after.Value.ClientReady || after.Value.ConnectionEpoch != 1 {
		t.Fatalf("Attach did not publish Rev+1: %d -> %d %+v", before.Rev, after.Rev, after.Value)
	}
	select {
	case <-changed:
	default:
		t.Fatal("Changed channel did not close on publish")
	}
	s.Detach()
	if epoch := s.Snapshot().Value.ConnectionEpoch; epoch != 2 {
		t.Fatalf("Detach did not renew the connection lease: epoch %d", epoch)
	}
}

func TestScheduleStaleReadDroppedAndSnapshotUnblocked(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	cancelled := make(chan struct{})
	first := true
	client := scheduleFixture(func(r *http.Request, _ string) (any, error) {
		if first {
			first = false
			close(started)
			select {
			case <-r.Context().Done():
				close(cancelled)
			case <-release:
			}
		}
		return catalogRecord(), nil
	})
	s := New()
	s.Attach(client, nil)
	s.Refresh(context.Background(), "owner")
	<-started
	read := make(chan bool, 1)
	go func() { read <- s.Snapshot().Value.Loading }()
	select {
	case loading := <-read:
		if !loading {
			t.Fatal("in-flight read not published as Loading")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Snapshot blocked behind the catalog RPC")
	}
	// A same-client Attach renews the read lease: the in-flight read is
	// cancelled and its completion must not publish.
	s.Attach(client, nil)
	select {
	case <-cancelled:
	case <-time.After(3 * time.Second):
		t.Fatal("superseded read was not cancelled")
	}
	held := s.Snapshot()
	time.Sleep(50 * time.Millisecond)
	if got := s.Snapshot(); got != held || len(got.Value.Rows) != 0 {
		t.Fatal("stale read completion was published")
	}
	close(release)
}

func TestScheduleFaultMapping(t *testing.T) {
	s := New()
	s.Attach(scheduleFixture(func(*http.Request, string) (any, error) {
		return nil, errors.New("dial tcp 127.0.0.1:1: connectex: connection refused")
	}), nil)
	s.Refresh(context.Background(), "")
	v := waitSettled(t, s)
	f := v.Value.Err
	if f == nil || f.Cause != errcopy.CauseNetwork {
		t.Fatalf("Err = %+v", f)
	}
	if text := f.SentenceText(); text == "" || strings.Contains(text, "connectex") {
		t.Fatalf("fault renders raw text: %q", text)
	}
}

func TestScheduleEventsOwnerLease(t *testing.T) {
	client := scheduleFixture(func(*http.Request, string) (any, error) { return catalogRecord(), nil })
	s := New()
	s.Attach(client, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	one := s.BeginEvents(ctx, client)
	two := s.BeginEvents(ctx, client)
	s.mu.Lock()
	oneCurrent, twoCurrent := s.eventsOwnerCurrentLocked(one), s.eventsOwnerCurrentLocked(two)
	s.mu.Unlock()
	if oneCurrent || !twoCurrent {
		t.Fatalf("a newer stream must fence the older owner: one=%v two=%v", oneCurrent, twoCurrent)
	}
	s.Detach()
	s.mu.Lock()
	twoCurrent = s.eventsOwnerCurrentLocked(two)
	s.mu.Unlock()
	if twoCurrent || two.lease.Context().Err() == nil {
		t.Fatal("Detach did not retire the events owner")
	}
}

// Update's committed-record merge replaces the row in a fresh slice; the
// version a reader holds keeps its old row.
func TestScheduleUpdateLeavesHeldVersion(t *testing.T) {
	client := scheduleFixture(func(_ *http.Request, method string) (any, error) {
		if method == "edit" {
			return map[string]any{"record": map[string]any{"id": "job", "title": "Evening", "kind": "at", "prompt": "p", "scheduledAt": "2026-12-01T01:00:00Z"}}, nil
		}
		return catalogRecord(), nil
	})
	s := New()
	s.Attach(client, nil)
	s.Refresh(context.Background(), "owner")
	held := waitSettled(t, s)
	title := "Evening"
	if err := s.Update(context.Background(), held.Value.Rows[0], &title, nil, nil); err != nil {
		t.Fatal(err)
	}
	if held.Value.Rows[0].Title != "Morning" {
		t.Fatal("Update edited a published row in place")
	}
}
