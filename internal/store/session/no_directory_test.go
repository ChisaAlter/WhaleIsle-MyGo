package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"whaleisle/internal/dsh"
)

func TestNoDirectoryHostAuthority(t *testing.T) {
	var creates, checks atomic.Int32
	var refuse, fail atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		switch {
		case strings.HasSuffix(r.URL.Path, "/session/create"):
			creates.Add(1)
			args := body["payload"].(map[string]any)["args"].(map[string]any)
			if args["request"].(map[string]any)["cwd"] != "/host/scratch" {
				t.Errorf("non-authoritative cwd: %#v", args)
			}
			rpcOK(w, body["rpcId"], map[string]any{"sessionId": "new"})
		case strings.HasSuffix(r.URL.Path, "/session/blankReuse"):
			checks.Add(1)
			if fail.Load() {
				http.Error(w, "unavailable", http.StatusServiceUnavailable)
				return
			}
			rpcOK(w, body["rpcId"], map[string]any{"reusable": !refuse.Load()})
		default:
			t.Errorf("unexpected mutation/request %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	newStore := func() *Store {
		s := New()
		s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
		return s
	}
	baseline := func(s *Store) {
		s.applyWorkspaceFollowFrame(json.RawMessage(`{"type":"baseline","value":{"items":[{"workspaceId":"ws","sessionIds":["member"]}],"archivedSessionIds":["archived"],"pinnedSessionIds":[],"scratchCwd":"/host/scratch"}}`))
	}
	s := newStore()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		id, err := s.CreateNoDirectorySession(ctx)
		if err == nil && id != "new" {
			t.Errorf("id %q", id)
		}
		done <- err
	}()
	select {
	case err := <-done:
		t.Fatalf("did not await baseline: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	baseline(s)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if creates.Load() != 1 || s.Snapshot().SelectedID != "" {
		t.Fatal("creation or selection ownership changed")
	}
	s.items = []Item{{SessionID: "member", Cwd: "/host/scratch", Blank: true}, {SessionID: "archived", Cwd: "/host/scratch", Blank: true}, {SessionID: "child", Cwd: "/host/scratch", Blank: true, Origin: "subagent"}, {SessionID: "other", Cwd: "/deleted", Blank: true}, {SessionID: "used", Cwd: "/host/scratch"}, {SessionID: "blank", Cwd: "/host/scratch", Blank: true}}
	id, err := s.CreateNoDirectorySession(ctx)
	if err != nil || id != "blank" || checks.Load() != 1 || creates.Load() != 1 {
		t.Fatalf("unsafe reuse %s %v checks=%d creates=%d", id, err, checks.Load(), creates.Load())
	}
	snap := s.Snapshot()
	if snap.IsNoDirectory(s.items[0]) || snap.IsNoDirectory(s.items[3]) || !snap.IsNoDirectory(s.items[5]) {
		t.Fatal("scratch membership classification")
	}
	fail.Store(true)
	if _, err = s.CreateNoDirectorySession(ctx); err == nil || creates.Load() != 1 {
		t.Fatal("reuse read failure created or reused")
	}
	fail.Store(false)
	refuse.Store(true)
	if _, err = s.CreateNoDirectorySession(ctx); err != nil || creates.Load() != 2 {
		t.Fatal("explicitly ineligible candidate did not create", err)
	}
	missing := newStore()
	missing.applyWorkspaceFollowFrame(json.RawMessage(`{"type":"baseline","value":{"items":[],"archivedSessionIds":[],"pinnedSessionIds":[]}}`))
	if _, err = missing.CreateNoDirectorySession(ctx); err == nil {
		t.Fatal("invented a missing scratch cwd")
	}
	stopped := newStore()
	go func() { _, err := stopped.CreateNoDirectorySession(ctx); done <- err }()
	stopped.Detach()
	if err = <-done; err == nil {
		t.Fatal("detach did not abort wait")
	}
}
