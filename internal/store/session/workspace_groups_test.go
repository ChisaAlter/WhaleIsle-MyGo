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

func TestWorkspaceGroupingFixture(t *testing.T) {
	rows := []WorkspaceRow{
		{WorkspaceID: "root", Path: "/srv/repo", Title: "Root", SessionIDs: []string{"second", "first"}},
		{WorkspaceID: "child", Path: "/srv/repo/nested", Title: "Child", SessionIDs: []string{"child-session"}},
		{WorkspaceID: "neighbor", Path: "/srv/repository", Title: "Neighbor"},
		{WorkspaceID: "win", Path: `C:\Code`, Title: "Windows"},
		{WorkspaceID: "winchild", Path: `c:\code\Nested`, Title: "Windows child"},
	}
	items := []Item{{SessionID: "first", Cwd: "/srv/repo", UpdatedAt: 20}, {SessionID: "second", Cwd: "/srv/repo", UpdatedAt: 10}, {SessionID: "sub", ParentID: "first", Cwd: "/srv/repo", UpdatedAt: 30}, {SessionID: "child-session", Cwd: "/srv/repo/nested"}, {SessionID: "scratch", Cwd: "/scratch"}, {SessionID: "orphan", Cwd: "/srv/repo"}}
	snap := ViewState{Workspaces: rows, WorkspaceLayout: WorkspaceLayoutState{GroupBy: "workspace-tree", OrderBy: "manual"}, ScratchCwd: "/scratch", WorkspaceMembers: map[string]bool{"first": true, "second": true, "child-session": true}}
	groups := BuildWorkspaceGroups(snap, items)
	byID := map[string]WorkspaceGroup{}
	for _, g := range groups {
		byID[g.Key] = g
	}
	if byID["child"].ParentID != "root" || byID["child"].Depth != 1 || byID["neighbor"].ParentID != "" || byID["winchild"].ParentID != "win" || byID["winchild"].Depth != 1 {
		t.Fatalf("Host path tree mismatch %#v", groups)
	}
	if len(byID["root"].Items) != 3 || len(byID["::scratch"].Items) != 2 {
		t.Fatal("membership guessed from path or unfiled sessions split")
	}
	foundOrphan := false
	for _, item := range byID["::scratch"].Items {
		if item.SessionID == "orphan" {
			foundOrphan = true
		}
	}
	if !foundOrphan {
		t.Fatal("a session outside every registered workspace was not filed under 无工作目录")
	}
	ordered := OrderedWorkspaceSessionRows(byID["root"].Items, rows[0].SessionIDs, nil, "manual", false)
	if len(ordered) != 3 || ordered[0].Item.SessionID != "second" || ordered[1].Item.SessionID != "first" || ordered[2].Depth != 1 {
		t.Fatalf("manual/session tree %#v", ordered)
	}
	ordered = OrderedWorkspaceSessionRows(byID["root"].Items, rows[0].SessionIDs, nil, "updated", true)
	if ordered[0].Item.SessionID != "sub" || ordered[1].Item.SessionID != "first" || ordered[0].Depth != 0 {
		t.Fatal("updated/flat order")
	}
	ordered = OrderedWorkspaceSessionRows(byID["root"].Items, rows[0].SessionIDs, []string{"first"}, "manual", true)
	if ordered[0].Item.SessionID != "first" {
		t.Fatal("pin priority lost")
	}
	snap.WorkspaceLayout.GroupBy = "workspace"
	for _, g := range BuildWorkspaceGroups(snap, items) {
		if g.Depth != 0 || g.ParentID != "" {
			t.Fatal("group mode retained tree nesting")
		}
	}
	snap.WorkspaceLayout.GroupBy = "flat"
	flat := BuildWorkspaceGroups(snap, items)
	if len(flat) != 1 || len(flat[0].Items) != len(items) {
		t.Fatal("flat view omitted sessions")
	}
	if before, ok := WorkspaceMoveAnchor(groups, "root", 1); !ok || before != "win" {
		t.Fatalf("move down anchor %q %v", before, ok)
	}
	if _, ok := WorkspaceMoveAnchor(groups, "child", -1); ok {
		t.Fatal("tree moved a child across parent groups")
	}
	if before, ok := WorkspaceMoveAnchor(groups, "win", -1); !ok || before != "neighbor" {
		t.Fatal("move up anchor")
	}
	cycle := OrderedWorkspaceSessionRows([]Item{{SessionID: "a", ParentID: "b"}, {SessionID: "b", ParentID: "a"}}, nil, nil, "manual", false)
	if len(cycle) != 2 {
		t.Fatal("cyclic source lost or repeated rows")
	}

	var calls atomic.Int32
	var fail atomic.Bool
	requests := make(chan map[string]any, 8)
	var block atomic.Pointer[chan struct{}]
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		calls.Add(1)
		args := body["payload"].(map[string]any)["args"].(map[string]any)
		requests <- args["request"].(map[string]any)
		if gate := block.Load(); gate != nil {
			<-*gate
		}
		if fail.Load() {
			http.Error(w, "Host refused order", 503)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/workspace/insertSessionBefore") {
			rpcOK(w, body["rpcId"], map[string]any{"workspace": map[string]any{"workspaceId": "one", "path": "/one", "title": "One", "sessionIds": []string{"b", "a"}}})
			return
		}
		if !strings.HasSuffix(r.URL.Path, "/workspace/insertBefore") {
			t.Errorf("fabricated reorder RPC %s", r.URL.Path)
		}
		rpcOK(w, body["rpcId"], map[string]any{"workspaceIds": []string{"two", "one"}})
	}))
	defer srv.Close()
	fresh := func() *Store {
		s := New()
		s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
		s.applyWorkspaceFollowFrame(json.RawMessage(`{"type":"baseline","value":{"items":[{"workspaceId":"one","path":"/one","sessionIds":[]},{"workspaceId":"two","path":"/two","sessionIds":[]}],"pinnedSessionIds":[],"archivedSessionIds":[],"scratchCwd":"/scratch"}}`))
		return s
	}
	wait := func(done func() bool) {
		t.Helper()
		until := time.Now().Add(time.Second)
		for !done() {
			if time.Now().After(until) {
				t.Fatal("fake reorder did not settle")
			}
			time.Sleep(time.Millisecond)
		}
	}
	s := fresh()
	s.SetWorkspaceGrouping("workspace-tree")
	s.SetWorkspaceOrdering("updated")
	s.ToggleWorkspaceGroup("one")
	layout := s.Snapshot().WorkspaceLayout
	layout.Collapsed["one"] = false
	if !s.Snapshot().WorkspaceLayout.Collapsed["one"] {
		t.Fatal("snapshot aliases collapse state")
	}
	gate := make(chan struct{})
	block.Store(&gate)
	s.InsertWorkspaceBefore(context.Background(), "two", "one")
	s.InsertWorkspaceBefore(context.Background(), "two", "")
	req := <-requests
	if req["workspaceId"] != "two" || req["beforeWorkspaceId"] != "one" {
		t.Fatal("wrong reorder request", req)
	}
	close(gate)
	block.Store(nil)
	wait(func() bool { return !s.Snapshot().WorkspaceLayout.Reordering })
	if calls.Load() != 1 || s.Snapshot().Workspaces[0].WorkspaceID != "two" {
		t.Fatal("duplicate reorder or receipt not projected")
	}
	s.applyWorkspaceFollowFrame(json.RawMessage(`{"type":"upsert","workspace":{"workspaceId":"one","path":"/one","sessionIds":["a","b"]}}`))
	gate = make(chan struct{})
	block.Store(&gate)
	s.InsertSessionBefore(context.Background(), "one", "b", "a")
	s.InsertSessionBefore(context.Background(), "one", "b", "a")
	req = <-requests
	if req["workspaceId"] != "one" || req["sessionId"] != "b" || req["beforeSessionId"] != "a" {
		t.Fatal("session move used wrong workspace or anchor", req)
	}
	close(gate)
	block.Store(nil)
	wait(func() bool { return !s.Snapshot().WorkspaceLayout.Reordering })
	if calls.Load() != 2 {
		t.Fatal("duplicate session move was sent")
	}
	got := s.Snapshot()
	for _, ws := range got.Workspaces {
		if ws.WorkspaceID == "one" && ws.SessionIDs[0] != "b" {
			t.Fatal("session manual order receipt was ignored")
		}
	}
	fail.Store(true)
	s.InsertWorkspaceBefore(context.Background(), "one", "two")
	<-requests
	wait(func() bool { return !s.Snapshot().WorkspaceLayout.Reordering })
	if s.Snapshot().WorkspaceLayout.Error == "" || s.Snapshot().Workspaces[0].WorkspaceID != "two" {
		t.Fatal("failure hid error or optimistically reordered")
	}
	fail.Store(false)
	gate = make(chan struct{})
	block.Store(&gate)
	s.InsertWorkspaceBefore(context.Background(), "one", "")
	req = <-requests
	if _, exists := req["beforeWorkspaceId"]; exists {
		t.Fatal("append encoded a nonexistent anchor")
	}
	s.Detach()
	close(gate)
	block.Store(nil)
	if s.Snapshot().WorkspaceLayout.Reordering || len(s.Snapshot().Workspaces) != 0 || len(s.Snapshot().WorkspaceLayout.Collapsed) != 0 {
		t.Fatal("Host reset retained old registry state")
	}
}
