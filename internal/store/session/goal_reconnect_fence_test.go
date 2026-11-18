package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"whaleisle/internal/dsh"
)

// A goal read started by one events owner must not publish after a reconnect has
// replaced that owner. Today the continuation is fenced by followGeneration,
// which a reconnect does not advance, so a late goal from the previous session
// lands in the store and the person sees a goal that belongs to a Host
// generation that is gone.
func TestGoalReadStartedBeforeReconnectDoesNotPublish(t *testing.T) {
	goalRequested := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]json.RawMessage
		_ = json.NewDecoder(r.Body).Decode(&body)
		if strings.HasSuffix(r.URL.Path, "/goals/get") {
			once.Do(func() { close(goalRequested) })
			<-release
			mu.Lock()
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type":  "server-response",
				"rpcId": body["rpcId"],
				"result": map[string]any{"ok": true, "value": map[string]any{
					"id":    "stale-goal",
					"title": "Goal from the previous Host",
				}},
			})
			mu.Unlock()
			return
		}
		mu.Lock()
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": body["rpcId"], "result": map[string]any{"ok": true, "value": map[string]any{}}})
		mu.Unlock()
	}))
	defer srv.Close()
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()

	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	st := New()
	st.Attach(client, nil)
	owner := st.BeginEvents(context.Background(), client)
	if owner == nil {
		t.Fatal("events generation did not start")
	}
	st.mu.Lock()
	st.items = []Item{{SessionID: "A", Running: true}}
	st.selectedID = "A"
	st.mu.Unlock()

	st.OnEmit(owner, dsh.EventsEmit{
		Type:  dsh.EventEmit,
		Event: "api-session/status",
		Args:  []json.RawMessage{json.RawMessage(`"A"`), json.RawMessage(`false`)},
	})
	select {
	case <-goalRequested:
	case <-time.After(5 * time.Second):
		t.Fatal("goal read never started")
	}

	// A reconnect replaces the events owner while the goal read is in flight.
	if reconnect := st.BeginEvents(context.Background(), client); reconnect == nil {
		t.Fatal("reconnect did not acquire a generation")
	}
	before := st.Snapshot()

	close(release)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if st.Snapshot().PlanBoard.Goal != nil {
			mu.Lock()
			mu.Unlock()
			t.Fatalf("goal from the previous events owner published after reconnect: %+v", st.Snapshot().PlanBoard.Goal)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got := st.Snapshot(); got.PlanBoard.Goal != before.PlanBoard.Goal {
		t.Fatalf("store changed after reconnect: %+v", got.PlanBoard)
	}
}
