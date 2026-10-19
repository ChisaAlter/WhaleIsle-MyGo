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

func TestWorkspaceRemovalConversationOwnership(t *testing.T) {
	for _, scenario := range []string{"receipt-first", "follow-first", "unrelated-selection", "newer-selection", "newer-host", "rejected-receipt"} {
		t.Run(scenario, func(t *testing.T) {
			entered, release := make(chan struct{}), make(chan struct{})
			var once sync.Once
			unblock := func() { once.Do(func() { close(release) }) }
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !strings.HasSuffix(r.URL.Path, "/workspace/delete") {
					t.Errorf("unexpected RPC %s", r.URL.Path)
					http.Error(w, "unexpected RPC", 500)
					return
				}
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				close(entered)
				<-release
				rpcOK(w, body["rpcId"], map[string]any{"deleted": scenario != "rejected-receipt"})
			}))
			defer func() { unblock(); srv.Close() }()
			s := New()
			s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
			s.SetWorkspaceCwd("/host/one")
			s.applyWorkspaceFollowFrame(json.RawMessage(`{"type":"baseline","value":{"items":[{"workspaceId":"one","path":"/host/one","title":"One","sessionIds":["owned"]},{"workspaceId":"two","path":"/host/two","title":"Two","sessionIds":["other"]}],"pinnedSessionIds":[],"archivedSessionIds":[],"scratchCwd":"/host/scratch"}}`))
			s.selectedID = "owned"
			s.items = []Item{{SessionID: "owned", Cwd: "/host/one"}, {SessionID: "other", Cwd: "/host/two"}}
			if scenario == "unrelated-selection" {
				s.selectedID = "other"
			}
			s.BeginWorkspaceRemoval("one")
			token := s.Snapshot().WorkspaceNavigation.Token
			s.ConfirmWorkspaceRemoval(context.Background(), token)
			select {
			case <-entered:
			case <-time.After(2 * time.Second):
				t.Fatal("delete request did not arrive")
			}
			if scenario == "newer-selection" {
				s.mu.Lock()
				s.selectSessionLocked("other")
				s.mu.Unlock()
			}
			if scenario == "newer-host" {
				s.mu.Lock()
				s.eventsGeneration++
				s.mu.Unlock()
			}
			if scenario == "receipt-first" || scenario == "unrelated-selection" {
				unblock()
				if s.Snapshot().SelectedID == "" {
					t.Fatal("selection cleared before the authoritative membership removal")
				}
			}
			s.applyWorkspaceFollowFrame(json.RawMessage(`{"type":"remove","workspaceId":"one"}`))
			if scenario == "follow-first" || scenario == "rejected-receipt" {
				if s.Snapshot().SelectedID != "owned" {
					t.Fatal("follow echo alone cleared the conversation")
				}
			}
			unblock()
			deadline := time.Now().Add(2 * time.Second)
			for {
				op := s.Snapshot().WorkspaceNavigation
				if op == nil || (scenario == "rejected-receipt" && op.Error != "") {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("workspace removal did not settle")
				}
				time.Sleep(time.Millisecond)
			}
			want := ""
			if scenario == "unrelated-selection" || scenario == "newer-selection" {
				want = "other"
			}
			if scenario == "newer-host" || scenario == "rejected-receipt" {
				want = "owned"
			}
			view := s.Snapshot()
			if view.SelectedID != want || len(view.Items) != 2 {
				t.Fatalf("selection/session preservation: selected=%q want=%q items=%d", view.SelectedID, want, len(view.Items))
			}
		})
	}
}
