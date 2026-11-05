package session

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"

	"whaleisle/internal/dsh"
)

func parityWorkspaceClient(t *testing.T, run func(string) (any, error)) *dsh.Client {
	t.Helper()
	return &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://workspace-parity.invalid"}, HTTP: &http.Client{Transport: lifecycleAdmissionTransport(func(r *http.Request) (*http.Response, error) {
		var body struct {
			RPCID string `json:"rpcId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			return nil, err
		}
		value, err := run(strings.TrimPrefix(r.URL.Path, "/api/"))
		if err != nil {
			return nil, err
		}
		raw, err := json.Marshal(map[string]any{"type": "server-response", "rpcId": body.RPCID, "result": map[string]any{"ok": true, "value": value}})
		if err != nil {
			return nil, err
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
	})}}
}

func TestGlobalWorkspaceCreationUsesHostMembershipAndFencesNavigation(t *testing.T) {
	for _, navigate := range []bool{false, true} {
		synctest.Test(t, func(t *testing.T) {
			h := newWorkspaceCreationHost(t)
			h.store.mu.Lock()
			h.store.selectedID = "two-blank"
			h.store.items = []Item{{SessionID: "two-blank", Cwd: "/same", Blank: true}}
			h.store.mu.Unlock()
			h.handler = func(call workspaceCreationCall) (map[string]any, *dsh.Error) {
				if call.Method != "session/create" || call.Request["workspaceId"] != "two" || call.Request["cwd"] != nil || call.Request["sessionId"] != nil {
					t.Fatalf("global create did not preserve membership or excluded draft identities: %+v", call)
				}
				if navigate {
					h.store.Select("newer")
					h.store.SetLastAction("newer action")
				}
				return map[string]any{"sessionId": "global-created"}, nil
			}
			h.store.CreateSession(context.Background())
			synctest.Wait()
			snap := h.store.Snapshot()
			want := "global-created"
			if navigate {
				want = "newer"
			}
			if snap.SelectedID != want || !snap.WorkspaceMembers["global-created"] {
				t.Fatalf("selected=%q membership=%v", snap.SelectedID, snap.WorkspaceMembers)
			}
			if navigate && snap.LastAction != "newer action" {
				t.Fatalf("late failure overwrote current action: %q", snap.LastAction)
			}
		})
	}
}

func TestNewSessionWorkspaceResolutionKeepsScratchAndRecencySeparate(t *testing.T) {
	h := newWorkspaceCreationHost(t)
	h.store.mu.Lock()
	h.store.items = []Item{{SessionID: "scratch", Cwd: "/scratch"}, {SessionID: "two-blank", UpdatedAt: 42}}
	h.store.selectedID, h.store.cwd, h.store.workspaceID = "scratch", "/same", "one"
	h.store.mu.Unlock()
	if got := h.store.NewSessionWorkspaceID(); got != "::scratch" {
		t.Fatalf("scratch inherited directory workspace %q", got)
	}
	h.store.mu.Lock()
	h.store.selectedID, h.store.cwd, h.store.workspaceID = "", "/missing", ""
	h.store.mu.Unlock()
	if got := h.store.NewSessionWorkspaceID(); got != "two" {
		t.Fatalf("recent target=%q", got)
	}
}

func TestWorkspaceRestoreDoesNotInitializeOrReplaceRegisteredDirectory(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := New()
		initCalls := 0
		client := parityWorkspaceClient(t, func(method string) (any, error) {
			if method == "workspace/initializeDefault" {
				initCalls++
				return map[string]any{"workspace": map[string]any{"workspaceId": "default", "path": "/default"}}, nil
			}
			return map[string]any{"items": []any{}}, nil
		})
		s.Attach(client, nil)
		s.SetWorkspaceCwd("/saved")
		s.applyWorkspaceBaseline(json.RawMessage(`{"type":"baseline","value":{"items":[{"workspaceId":"saved","path":"/saved","sessionIds":[]}],"scratchCwd":"/scratch","pinnedSessionIds":[],"archivedSessionIds":[]}}`))
		s.InitializeDefaultWorkspace(context.Background())
		synctest.Wait()
		if initCalls != 0 || s.WorkspaceCwd() != "/saved" || s.WorkspaceID() != "saved" || s.SelectedSessionID() != "" {
			t.Fatalf("restored directory was replaced: calls=%d cwd=%q id=%q", initCalls, s.WorkspaceCwd(), s.WorkspaceID())
		}
	})
}

func TestLateDefaultReplyCannotOverwriteSavedDirectoryBeforeBaseline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := New()
		client := parityWorkspaceClient(t, func(method string) (any, error) {
			if method == "workspace/initializeDefault" {
				return map[string]any{"workspace": map[string]any{"workspaceId": "default", "path": "/default"}}, nil
			}
			return map[string]any{"items": []any{}}, nil
		})
		s.Attach(client, nil)
		s.SetWorkspaceCwd("/saved")
		s.InitializeDefaultWorkspace(context.Background())
		synctest.Wait()
		if s.WorkspaceCwd() != "/saved" {
			t.Fatalf("default reply replaced saved cwd with %q", s.WorkspaceCwd())
		}
	})
}

func TestForkReplyCannotSelectAcrossNewNavigation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := New()
		client := parityWorkspaceClient(t, func(method string) (any, error) {
			if method == "session/fork" {
				s.Select("newer")
				s.SetLastAction("newer action")
				return map[string]any{"sessionId": "fork-result"}, nil
			}
			return map[string]any{}, nil
		})
		s.Attach(client, nil)
		s.Select("source")
		s.ForkSession(context.Background(), "source")
		synctest.Wait()
		if s.SelectedSessionID() != "newer" || s.Snapshot().LastAction != "newer action" {
			t.Fatal("fork reply retargeted newer navigation")
		}
	})
}

func TestDeleteFailureKeepsExactTargetForRetry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := New()
		calls := 0
		client := parityWorkspaceClient(t, func(method string) (any, error) {
			if method == "session/delete" {
				calls++
				if calls == 1 {
					return nil, errors.New("delete refused")
				}
				return map[string]any{"deletedSessionIds": []string{"archived"}, "archivedSessionIds": []string{}}, nil
			}
			return map[string]any{"items": []any{}}, nil
		})
		s.Attach(client, nil)
		s.items = []Item{{SessionID: "archived", Title: "Archived"}}
		s.archivedSessionIDs = []string{"archived"}
		s.DeleteSession(context.Background(), "archived")
		synctest.Wait()
		failed := s.SessionDeletion()
		if failed.SessionID != "archived" || failed.Title != "Archived" || failed.Busy || failed.Error == "" || len(s.Snapshot().ArchivedSessionIDs) != 1 {
			t.Fatalf("missing retry target: %+v", failed)
		}
		s.DeleteSession(context.Background(), failed.SessionID)
		synctest.Wait()
		if calls != 2 || s.SessionDeletion().SessionID != "" || len(s.Snapshot().ArchivedSessionIDs) != 0 {
			t.Fatal("retry did not consume successful Host deletion")
		}
	})
}
