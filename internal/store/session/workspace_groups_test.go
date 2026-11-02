package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"whaleisle/internal/dsh"
)

func TestWorkspaceGroupsUseDirectAccountsAndRealScratch(t *testing.T) {
	snap := ViewState{SelectedID: "blank", ScratchCwd: "/scratch", Workspaces: []WorkspaceRow{
		{WorkspaceID: "one", SessionIDs: []string{"parent", "blank", "old-blank", "child"}},
		{WorkspaceID: "two", SessionIDs: []string{"fork"}},
	}, Items: []Item{
		{SessionID: "parent", Cwd: "/one"},
		{SessionID: "fork", ParentID: "parent", Cwd: "/one"},
		{SessionID: "unaccounted", ParentID: "parent", Cwd: "/one"},
		{SessionID: "scratch", ParentID: "parent", Cwd: "/scratch"},
		{SessionID: "foreign", Cwd: "/missing"},
		{SessionID: "blank", Blank: true, Cwd: "/one"},
		{SessionID: "old-blank", Blank: true, Cwd: "/one"},
		{SessionID: "child", Origin: "subagent", Cwd: "/one"},
	}}
	for _, mode := range []string{"workspace", "workspace-tree", "flat"} {
		snap.WorkspaceLayout.GroupBy = mode
		groups := BuildWorkspaceGroups(snap, snap.Items)
		owners := map[string]string{}
		for _, group := range groups {
			for _, item := range group.Items {
				owners[item.SessionID] = group.Key
			}
		}
		want := map[string]string{"parent": "one", "fork": "two", "blank": "one", "scratch": "::scratch"}
		if mode == "flat" {
			want = map[string]string{"parent": "::flat", "fork": "::flat", "blank": "::flat", "scratch": "::flat"}
		}
		if !reflect.DeepEqual(owners, want) {
			t.Fatalf("%s parent lineage or cwd inferred account: %v want %v", mode, owners, want)
		}
	}
	if len(snap.Workspaces[0].SessionIDs) != 4 || snap.Workspaces[1].SessionIDs[0] != "fork" {
		t.Fatal("render projection rewrote Host membership")
	}
}

func TestWorkspaceAccountOrderPreservesArchivesAndPlacesNewForks(t *testing.T) {
	snap := ViewState{SelectedID: "blank", Workspaces: []WorkspaceRow{{WorkspaceID: "one", SessionIDs: []string{"parent", "saved-fork", "new-fork", "new-grandchild", "other", "pin-a", "pin-b", "blank", "archived", "missing"}}},
		PinnedSessionIDs: []string{"pin-a", "pin-b"}, ArchivedSessionIDs: []string{"archived"},
		WorkspaceLayout: WorkspaceLayoutState{OrderBy: "manual", SessionOrders: map[string][]string{"one": {"other", "archived", "missing", "parent", "saved-fork", "pin-b", "pin-a"}}},
		Items: []Item{
			{SessionID: "parent", UpdatedAt: 50}, {SessionID: "saved-fork", ParentID: "parent", UpdatedAt: 10},
			{SessionID: "new-fork", ParentID: "parent", UpdatedAt: 30}, {SessionID: "new-grandchild", ParentID: "new-fork", UpdatedAt: 40},
			{SessionID: "other", UpdatedAt: 60}, {SessionID: "pin-a", UpdatedAt: 80}, {SessionID: "pin-b", UpdatedAt: 70},
			{SessionID: "blank", Blank: true}, {SessionID: "archived", UpdatedAt: 100},
		}}
	visible := append([]Item(nil), snap.Items[:len(snap.Items)-1]...)
	group := BuildWorkspaceGroups(snap, visible)[0]
	wantOrder := []string{"blank", "other", "archived", "missing", "new-grandchild", "new-fork", "parent", "saved-fork", "pin-b", "pin-a"}
	if !reflect.DeepEqual(group.Workspace.SessionIDs, wantOrder) {
		t.Fatalf("reconciled order %v want %v", group.Workspace.SessionIDs, wantOrder)
	}
	rows := OrderedWorkspaceSessionRows(group.Items, group.Workspace.SessionIDs, snap.PinnedSessionIDs, "manual", false)
	var ids []string
	for _, row := range rows {
		ids = append(ids, row.Item.SessionID)
		if row.Depth != 0 {
			t.Fatal("ordinary fork gained lineage indentation")
		}
	}
	wantVisible := []string{"blank", "pin-b", "pin-a", "other", "new-grandchild", "new-fork", "parent", "saved-fork"}
	if !reflect.DeepEqual(ids, wantVisible) {
		t.Fatalf("blank/pins/baseorder: %v want %v", ids, wantVisible)
	}
	group = BuildWorkspaceGroups(snap, snap.Items)[0]
	if !reflect.DeepEqual(group.Workspace.SessionIDs, wantOrder) {
		t.Fatal("restoring hidden archive changed account slots")
	}
	snap.WorkspaceLayout.OrderBy = "updated"
	group = BuildWorkspaceGroups(snap, visible)[0]
	rows = OrderedWorkspaceSessionRows(group.Items, group.Workspace.SessionIDs, snap.PinnedSessionIDs, "updated", false)
	ids = nil
	for _, row := range rows {
		ids = append(ids, row.Item.SessionID)
	}
	if !reflect.DeepEqual(ids, []string{"blank", "pin-a", "pin-b", "other", "parent", "new-grandchild", "new-fork", "saved-fork"}) {
		t.Fatalf("updated row order %v", ids)
	}
	if snap.Workspaces[0].SessionIDs[0] != "parent" || snap.WorkspaceLayout.SessionOrders["one"][0] != "other" {
		t.Fatal("source order inputs mutated")
	}
}

func TestWorkspaceGroupingFixture(t *testing.T) {
	rows := []WorkspaceRow{
		{WorkspaceID: "root", Path: "/srv/repo", Title: "Root", SessionIDs: []string{"second", "first", "sub"}},
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
	if len(byID["root"].Items) != 3 || len(byID["::scratch"].Items) != 1 || byID["::scratch"].Items[0].SessionID != "scratch" {
		t.Fatal("membership or actual scratch ownership was inferred from lineage/path")
	}
	for _, group := range groups {
		for _, item := range group.Items {
			if item.SessionID == "orphan" {
				t.Fatal("unregistered foreign cwd entered a sidebar group")
			}
		}
	}
	ordered := OrderedWorkspaceSessionRows(byID["root"].Items, rows[0].SessionIDs, nil, "manual", false)
	if len(ordered) != 3 || ordered[0].Item.SessionID != "second" || ordered[1].Item.SessionID != "first" || ordered[2].Item.SessionID != "sub" || ordered[2].Depth != 0 {
		t.Fatalf("manual/fork flat rows %#v", ordered)
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
	if len(flat) != 1 || len(flat[0].Items) != len(items)-1 {
		t.Fatal("flat view did not retain all accounted/scratch sessions while excluding foreign cwd")
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
	s.ToggleWorkspaceGroup("one") // source default is folded; first toggle expands
	s.ToggleWorkspaceGroup("one") // explicitly fold before testing snapshot isolation
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
	s.mu.Lock()
	s.items = []Item{{SessionID: "a", UpdatedAt: 20}, {SessionID: "b", UpdatedAt: 10}}
	s.mu.Unlock()
	s.InsertSessionBefore(context.Background(), "one", "b", "a")
	s.InsertSessionBefore(context.Background(), "one", "b", "a")
	if calls.Load() != 1 {
		t.Fatal("browser-local session drag sent a Host membership RPC")
	}
	got := s.Snapshot()
	if got.WorkspaceLayout.OrderBy != "manual" || !reflect.DeepEqual(got.WorkspaceLayout.SessionOrders["one"], []string{"b", "a"}) {
		t.Fatal("session drag did not freeze local manual order", got.WorkspaceLayout)
	}
	for _, ws := range got.Workspaces {
		if ws.WorkspaceID == "one" && !reflect.DeepEqual(ws.SessionIDs, []string{"a", "b"}) {
			t.Fatal("browser-local ordering changed Host membership order")
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
