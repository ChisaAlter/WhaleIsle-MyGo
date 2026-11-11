package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

func TestComposerCreateKeepsCapturedWorkspaceAccount(t *testing.T) {
	for _, kind := range []string{"api", "gesture", "unregistered"} {
		t.Run(kind, func(t *testing.T) {
			var received api.SessionCreateRequest
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var wire struct {
					RPCID   string `json:"rpcId"`
					Payload struct {
						Args struct {
							Request api.SessionCreateRequest `json:"request"`
						} `json:"args"`
					} `json:"payload"`
				}
				if err := json.NewDecoder(r.Body).Decode(&wire); err != nil {
					t.Error(err)
				}
				received = wire.Payload.Args.Request
				rpcOK(w, wire.RPCID, map[string]any{"sessionId": "created"})
			}))
			defer srv.Close()
			st := New()
			st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
			st.applyWorkspaceFollowFrame(json.RawMessage(`{"type":"baseline","value":{"scratchCwd":"/scratch","items":[{"workspaceId":"original","path":"C:\\Work\\Alpha","sessionIds":[]}],"pinnedSessionIds":[],"archivedSessionIds":[]}}`))
			var target *ComposerTarget
			var err error
			if kind == "gesture" {
				target, err = st.CaptureComposerGestureTarget("", "", st.composerHostGeneration, false, nil)
			} else {
				target, err = st.CaptureComposerTarget("")
			}
			if err != nil {
				t.Fatal(err)
			}
			// A late registry replacement cannot retarget a detached send gesture.
			st.applyWorkspaceFollowFrame(json.RawMessage(`{"type":"replace","value":{"scratchCwd":"/scratch","items":[{"workspaceId":"replacement","path":"C:\\Work\\Alpha","sessionIds":[]}],"pinnedSessionIds":[],"archivedSessionIds":[]}}`))
			cwd := "c:/work/alpha"
			if kind == "unregistered" {
				cwd = "/outside-registry"
			}
			if _, err = target.CreateSession(context.Background(), cwd); err != nil {
				t.Fatal(err)
			}
			if kind == "unregistered" {
				if received.WorkspaceId != nil || received.Cwd == nil || *received.Cwd != cwd {
					t.Fatalf("directory-only contract changed: %+v", received)
				}
			} else if received.WorkspaceId == nil || *received.WorkspaceId != "original" || received.Cwd != nil {
				t.Fatalf("lost captured workspace identity: %+v", received)
			}
		})
	}
}
