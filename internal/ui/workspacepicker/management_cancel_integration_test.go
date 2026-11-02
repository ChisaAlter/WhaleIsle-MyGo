package workspacepicker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/dsh"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// Only public Store entry points populate this fixture. The Host is an in-process
// loopback server; NewTester handles local events without any native window,
// system input, clipboard, profile, subprocess or filesystem workspace.
func managementCancelStore(t *testing.T) (*session.Store, *atomic.Int32) {
	t.Helper()
	var rpcCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == dsh.MuxPath {
			conn, err := websocket.Accept(w, r, nil)
			if err != nil {
				return
			}
			defer conn.CloseNow()
			for {
				var frame dsh.ClientFrame
				if wsjson.Read(r.Context(), conn, &frame) != nil {
					return
				}
				if frame.Type != dsh.FrameOpen {
					continue
				}
				if frame.Endpoint != "workspace/follow" {
					t.Errorf("unexpected stream %q", frame.Endpoint)
					return
				}
				baseline := json.RawMessage(`{"type":"baseline","value":{"items":[{"workspaceId":"a","path":"/host/a","title":"Workspace A","sessionIds":["session-a"]},{"workspaceId":"b","path":"/host/b","title":"Workspace B","sessionIds":["session-b"]}],"scratchCwd":"/host/scratch","pinnedSessionIds":["session-b"],"archivedSessionIds":[]}}`)
				if wsjson.Write(r.Context(), conn, dsh.ServerFrame{Type: dsh.FrameItem, StreamID: frame.StreamID, Value: baseline}) != nil {
					return
				}
			}
		}
		rpcCalls.Add(1)
		if r.URL.Path != "/api/session/list" {
			t.Errorf("Cancel reached unexpected Host RPC %q", r.URL.Path)
			http.Error(w, "unexpected mutation", http.StatusBadRequest)
			return
		}
		var request struct {
			RPCID string `json:"rpcId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode list request: %v", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"type":"server-response","rpcId":"` + request.RPCID + `","result":{"ok":true,"value":{"items":[{"sessionId":"session-a","cwd":"/host/a","title":"Conversation A","blank":false,"running":false,"updatedAt":200},{"sessionId":"session-b","cwd":"/host/b","title":"Conversation B","blank":false,"running":false,"updatedAt":100}]}}}`))
	}))
	store := session.New()
	// Select before attaching so the fixture does not start unrelated detail reads.
	store.Select("session-a")
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL}, HTTP: server.Client()}
	store.Attach(client, nil)
	store.SetWorkspaceCwd("/host/a")
	mux := &dsh.Mux{Endpoint: client.Endpoint}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		store.Detach()
		_ = mux.Close()
		server.Close()
	})
	store.RefreshList(ctx)
	store.AttachMux(mux)
	store.FollowWorkspace(ctx)
	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	for {
		snap := store.Snapshot()
		if snap.WorkspaceReady && len(snap.Workspaces) == 2 && len(snap.Items) == 2 && !snap.ListLoading {
			if snap.SelectedID != "session-a" || snap.SelectedCwd != "/host/a" || store.WorkspaceID() != "a" {
				t.Fatalf("fixture lost current workspace A: id=%q selected=%q cwd=%q", store.WorkspaceID(), snap.SelectedID, snap.SelectedCwd)
			}
			return store, &rpcCalls
		}
		select {
		case <-deadline.C:
			t.Fatalf("public Store bootstrap did not settle: ready=%v rows=%d sessions=%d loading=%v", snap.WorkspaceReady, len(snap.Workspaces), len(snap.Items), snap.ListLoading)
		case <-tick.C:
		}
	}
}

func TestManagementDialogCancelPreservesActualStore(t *testing.T) {
	previousLocale := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(previousLocale) })
	for _, mode := range []string{"remove", "rename"} {
		t.Run(mode, func(t *testing.T) {
			store, rpcCalls := managementCancelStore(t)
			before := store.Snapshot()
			callsBefore := rpcCalls.Load()
			open := func() {
				if mode == "remove" {
					store.BeginWorkspaceRemoval("b")
				} else {
					store.BeginWorkspaceRename("b")
				}
			}
			open()
			request := store.Snapshot().WorkspaceNavigation
			if request == nil || request.Token == 0 || request.Mode != mode || request.TargetID != "b" || request.TargetTitle != "Workspace B" || request.Busy {
				t.Fatalf("real Store did not open target B management: %+v", request)
			}
			token := request.Token
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Text(c, "Current workspace A")
				View(c, Model{Store: store, ThemeID: "system"})
			}, 640, 480)
			tt.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 1})
			if current := store.Snapshot().WorkspaceNavigation; current == nil || current.Token != token {
				t.Fatalf("render did not retain the actual current navigation token: %+v", current)
			}
			if mode == "rename" {
				if !tt.Focused("Workspace name") {
					t.Fatal("rendered rename editor is not locally focused")
				}
				tt.Type("Uncommitted B title")
				if current := store.Snapshot().WorkspaceNavigation; current == nil || current.Token != token || current.Draft != "Uncommitted B title" {
					t.Fatalf("rename edit did not reach the real token-bound Store: %+v", current)
				}
			}
			if err := tt.Click("Cancel"); err != nil {
				t.Fatal(err)
			}
			tt.Frame()
			if store.Snapshot().WorkspaceNavigation != nil || tt.HasText("Cancel") || !tt.HasText("Current workspace A") {
				t.Fatal("rendered Cancel did not retire the actual Store dialog")
			}
			// A stale token must not dismiss a subsequent operation. The rendered
			// button then cancels that new current token, exercising the real owner.
			open()
			reopened := store.Snapshot().WorkspaceNavigation
			if reopened == nil || reopened.Token <= token {
				t.Fatalf("reopening reused the canceled token: %+v", reopened)
			}
			store.CancelWorkspaceNavigation(token)
			if current := store.Snapshot().WorkspaceNavigation; current == nil || current.Token != reopened.Token {
				t.Fatal("old token canceled the new workspace management operation")
			}
			tt.Frame()
			if err := tt.Click("Cancel"); err != nil {
				t.Fatal(err)
			}
			tt.Frame()
			after := store.Snapshot()
			if after.WorkspaceNavigation != nil || tt.HasText("Cancel") {
				t.Fatal("second Cancel did not dismiss the current-token dialog")
			}
			if store.WorkspaceID() != "a" || store.WorkspaceCwd() != "/host/a" || after.SelectedID != before.SelectedID || after.SelectedCwd != before.SelectedCwd {
				t.Fatal("Cancel changed the current workspace or selected session")
			}
			if !reflect.DeepEqual(after.Workspaces, before.Workspaces) || !reflect.DeepEqual(after.Items, before.Items) || !reflect.DeepEqual(after.WorkspaceMembers, before.WorkspaceMembers) || !reflect.DeepEqual(after.PinnedSessionIDs, before.PinnedSessionIDs) || !reflect.DeepEqual(after.ArchivedSessionIDs, before.ArchivedSessionIDs) {
				t.Fatal("Cancel mutated workspace registrations, session roster or membership")
			}
			if got := rpcCalls.Load(); got != callsBefore || got != 1 {
				t.Fatalf("Cancel sent a Host mutation: RPC count %d -> %d (want only bootstrap session/list)", callsBefore, got)
			}
		})
	}
}
