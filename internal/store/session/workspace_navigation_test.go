package session

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"whaleisle/internal/dsh"
)

// One focused fake-Host fixture exercises registration, selection, confirmation,
// retries and stale outcomes without opening any real user directory or GUI.
func TestWorkspaceNavigationFixture(t *testing.T) {
	var calls atomic.Int32
	var fail atomic.Bool
	var responseGate atomic.Pointer[chan struct{}]
	requests := make(chan string, 32)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		method := strings.TrimPrefix(r.URL.Path, "/api/")
		calls.Add(1)
		requests <- method
		if gate := responseGate.Load(); gate != nil {
			<-*gate
		}
		if fail.Load() {
			http.Error(w, "Host refused", 503)
			return
		}
		row := map[string]any{"workspaceId": "new", "path": "/host/canonical", "title": "New", "sessionIds": []string{}}
		var value any
		switch method {
		case "workspace/create":
			value = map[string]any{"workspace": row, "created": true}
		case "workspace/rename":
			row["workspaceId"] = "one"
			row["path"] = "/host/one"
			row["title"] = "Renamed"
			value = map[string]any{"workspace": row}
		case "workspace/delete":
			value = map[string]any{"deleted": true}
		case "directoryPicker/list":
			value = map[string]any{"path": "/host", "entries": []any{}, "crumbs": []any{}}
		default:
			t.Errorf("unexpected RPC %s", method)
		}
		rpcOK(w, body["rpcId"], value)
	}))
	defer srv.Close()
	fresh := func() *Store {
		s := New()
		s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
		s.SetWorkspaceCwd("/host/one")
		s.applyWorkspaceFollowFrame(json.RawMessage(`{"type":"baseline","value":{"items":[{"workspaceId":"one","path":"/host/one","title":"One","sessionIds":["s"]},{"workspaceId":"two","path":"/host/two","title":"Two","sessionIds":[]}],"pinnedSessionIds":[],"archivedSessionIds":[],"scratchCwd":"/host/scratch"}}`))
		s.selectedID = "s"
		s.items = []Item{{SessionID: "s", Cwd: "/host/one"}}
		return s
	}
	wait := func(done func() bool) {
		t.Helper()
		deadline := time.Now().Add(2 * time.Second)
		for !done() {
			if time.Now().After(deadline) {
				t.Fatal("fake Host result did not settle")
			}
			time.Sleep(time.Millisecond)
		}
	}
	var saved string
	persist := func(path string) error { saved = path; return nil }
	t.Run("registered rows and existing choice", func(t *testing.T) {
		s := fresh()
		snap := s.Snapshot()
		if len(snap.Workspaces) != 2 || snap.Workspaces[0].Path != "/host/one" {
			t.Fatal("registry rows discarded")
		}
		snap.Workspaces[0].SessionIDs[0] = "mutated"
		if s.Snapshot().Workspaces[0].SessionIDs[0] != "s" {
			t.Fatal("snapshot aliases membership")
		}
		s.OpenWorkspacePicker()
		op := s.Snapshot().WorkspaceNavigation
		n := calls.Load()
		saved = ""
		if !s.ChooseRegisteredWorkspace(op.Token, "two", persist) || saved != "/host/two" || s.WorkspaceID() != "two" || s.Snapshot().SelectedID != "" || calls.Load() != n {
			t.Fatal("existing selection did not atomically commit without RPC")
		}
	})
	t.Run("failure retry and canonical result", func(t *testing.T) {
		s := fresh()
		s.OpenWorkspacePicker()
		token := s.Snapshot().WorkspaceNavigation.Token
		saved = ""
		fail.Store(true)
		s.RegisterWorkspace(context.Background(), token, "/remote/not-a-local-path", persist, nil)
		wait(func() bool { op := s.Snapshot().WorkspaceNavigation; return op != nil && !op.Busy && op.Error != "" })
		op := s.Snapshot().WorkspaceNavigation
		if op.Draft != "/remote/not-a-local-path" || s.WorkspaceCwd() != "/host/one" || s.Snapshot().SelectedID != "s" || saved != "" {
			t.Fatal("failure lost draft or changed selection/config")
		}
		fail.Store(false)
		s.RegisterWorkspace(context.Background(), token, op.Draft, persist, nil)
		wait(func() bool { return s.Snapshot().WorkspaceNavigation == nil })
		if saved != "/host/canonical" || s.WorkspaceCwd() != saved || s.WorkspaceID() != "new" || s.Snapshot().SelectedID != "" {
			t.Fatal("registration did not commit Host row")
		}
	})
	t.Run("config failure preserves selection", func(t *testing.T) {
		s := fresh()
		s.OpenWorkspacePicker()
		token := s.Snapshot().WorkspaceNavigation.Token
		s.RegisterWorkspace(context.Background(), token, "/draft", func(string) error { return errors.New("disk full") }, nil)
		wait(func() bool {
			op := s.Snapshot().WorkspaceNavigation
			return op != nil && !op.Busy && op.Error == "disk full"
		})
		if s.WorkspaceCwd() != "/host/one" || s.Snapshot().SelectedID != "s" {
			t.Fatal("failed persistence partially selected")
		}
	})
	for _, change := range []string{"workspace", "Host", "selection", "cancel"} {
		t.Run("late "+change, func(t *testing.T) {
			for len(requests) > 0 {
				<-requests
			}
			s := fresh()
			s.OpenWorkspacePicker()
			token := s.Snapshot().WorkspaceNavigation.Token
			gate := make(chan struct{})
			responseGate.Store(&gate)
			var writes atomic.Int32
			cb := func(string) error { writes.Add(1); return nil }
			s.RegisterWorkspace(context.Background(), token, "/draft", cb, nil)
			s.RegisterWorkspace(context.Background(), token, "/duplicate", cb, nil)
			select {
			case method := <-requests:
				if method != "workspace/create" {
					t.Fatal(method)
				}
			case <-time.After(time.Second):
				t.Fatal("request not sent")
			}
			switch change {
			case "workspace":
				s.SetWorkspaceCwd("/newer")
			case "Host":
				s.Detach()
			case "selection":
				s.mu.Lock()
				s.selectedID = "newer"
				s.mu.Unlock()
			case "cancel":
				s.CancelWorkspaceNavigation(token)
			}
			close(gate)
			responseGate.Store(nil)
			wait(func() bool { return s.Snapshot().WorkspaceNavigation == nil })
			// The canceled HTTP request may finish after dialog invalidation; none can persist.
			if writes.Load() != 0 {
				t.Fatal("stale result persisted config")
			}
			if len(requests) > 0 {
				t.Fatal("duplicate registration was sent")
			}
		})
	}
	t.Run("rename duplicate error and retry", func(t *testing.T) {
		s := fresh()
		s.BeginWorkspaceRename("one")
		token := s.Snapshot().WorkspaceNavigation.Token
		n := calls.Load()
		s.ConfirmWorkspaceRename(context.Background(), token, "Two")
		if s.Snapshot().WorkspaceNavigation.Error == "" || calls.Load() != n {
			t.Fatal("duplicate name reached Host")
		}
		fail.Store(true)
		s.ConfirmWorkspaceRename(context.Background(), token, "Renamed")
		wait(func() bool { op := s.Snapshot().WorkspaceNavigation; return op != nil && !op.Busy && op.Error != "" })
		if s.Snapshot().WorkspaceNavigation.Draft != "Renamed" {
			t.Fatal("rename draft lost")
		}
		fail.Store(false)
		s.ConfirmWorkspaceRename(context.Background(), token, "Renamed")
		wait(func() bool { return s.Snapshot().WorkspaceNavigation == nil })
		if s.Snapshot().Workspaces[0].Title != "Renamed" {
			t.Fatal("rename projection missing")
		}
	})
	t.Run("remove explicit confirmation cancel and atomic projection", func(t *testing.T) {
		s := fresh()
		n := calls.Load()
		s.DeleteWorkspace(context.Background(), "one")
		op := s.Snapshot().WorkspaceNavigation
		if op == nil || op.Mode != "remove" || calls.Load() != n {
			t.Fatal("remove bypassed confirmation")
		}
		s.CancelWorkspaceNavigation(op.Token)
		if calls.Load() != n {
			t.Fatal("cancel sent mutation")
		}
		s.BeginWorkspaceRemoval("one")
		token := s.Snapshot().WorkspaceNavigation.Token
		s.ConfirmWorkspaceRemoval(context.Background(), token)
		s.ConfirmWorkspaceRemoval(context.Background(), token)
		wait(func() bool { return calls.Load() == n+1 })
		if s.Snapshot().WorkspaceNavigation == nil || len(s.Snapshot().Workspaces) != 2 {
			t.Fatal("closed before Host registry projection")
		}
		s.applyWorkspaceFollowFrame(json.RawMessage(`{"type":"remove","workspaceId":"one"}`))
		wait(func() bool { return s.Snapshot().WorkspaceNavigation == nil })
		if calls.Load() != n+1 || len(s.Snapshot().Workspaces) != 1 || s.Snapshot().Workspaces[0].WorkspaceID != "two" || s.WorkspaceCwd() != "/host/one" || s.Snapshot().SelectedID != "s" {
			t.Fatal("remove projection/selection/duplicate invariant")
		}
	})
	t.Run("follow order and old generation guard", func(t *testing.T) {
		s := fresh()
		s.applyWorkspaceFollowFrame(json.RawMessage(`{"type":"order","workspaceIds":["two","one"]}`))
		if s.Snapshot().Workspaces[0].WorkspaceID != "two" {
			t.Fatal("order frame ignored")
		}
		s.mu.Lock()
		generation, host := s.workspaceFollowGeneration, s.eventsGeneration
		s.workspaceFollowGeneration++
		if s.workspaceFollowCurrentLocked(context.Background(), nil, generation, host) {
			t.Fatal("late follow accepted")
		}
		s.mu.Unlock()
	})
}
