package files

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"whaleisle/internal/desktop/workspacefs"
	"whaleisle/internal/dsh"
)

func TestTextReadScopedStatRetiresLateSession(t *testing.T) {
	oldStarted, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	requests := make(chan string, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var env struct {
			RPCID   string `json:"rpcId"`
			Payload struct {
				Args map[string]string `json:"args"`
			} `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
			t.Error(err)
		}
		scope := env.Payload.Args["workspaceFileScopeId"]
		if r.URL.Path != "/api/workspaceFiles/stat" || env.Payload.Args["path"] != "same.txt" || len(env.Payload.Args) != 2 || (scope != "old-session" && scope != "new-session") {
			t.Errorf("wrong queued stat: %+v", env)
		}
		requests <- scope
		if scope == "old-session" {
			close(oldStarted)
			<-release
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": env.RPCID, "result": map[string]any{"ok": true, "value": map[string]any{"version": scope}}})
	}))
	defer server.Close()
	defer releaseOnce.Do(func() { close(release) })
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL}}
	q := &textReadQueue{}
	defer q.stop(true)
	notified := make(chan struct{}, 2)
	q.setNotify(func() { notified <- struct{}{} })
	old := textReadOwner{binding: "host/old-session", root: "owned-root", session: "old-session", path: "same.txt", tab: 1}
	read := func(scope string) func(context.Context) textReadResult {
		return func(context.Context) textReadResult {
			// Force a genuinely late receipt even after the queue cancels its job.
			st, err := workspacefs.StatRemoteScoped(context.Background(), client, scope, "same.txt")
			return textReadResult{stat: st.Version, err: err}
		}
	}
	q.request(old, read(old.session))
	awaitTextRead(t, oldStarted)
	next := old
	next.binding, next.session = "host/new-session", "new-session"
	q.request(next, read(next.session))
	if _, ok := q.take(old); ok {
		t.Fatal("retired Session retained stat authority")
	}
	releaseOnce.Do(func() { close(release) })
	awaitTextRead(t, notified)
	if _, ok := q.take(old); ok {
		t.Fatal("late stat escaped Session fence")
	}
	got, ready := q.take(next)
	if !ready || got.err != nil || got.stat != "new-session" {
		t.Fatalf("replacement stat = %+v, ready=%v", got, ready)
	}
	if first, second := <-requests, <-requests; first != "old-session" || second != "new-session" {
		t.Fatalf("frame scopes changed in flight: %q, %q", first, second)
	}
}
