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

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"whaleisle/internal/dsh"
)

// These are the selected-session callers which previously could start follow
// without a control reader. The socket fixture proves actual reader admission;
// the packaged-Host timed flow separately proves the real projection contract.
func TestSelectedSessionEntriesAdmitOneControlReader(t *testing.T) {
	for _, entry := range []string{"initial-list", "create", "fork", "same-select", "same-workspace-select"} {
		t.Run(entry, func(t *testing.T) {
			var controls, follows atomic.Int32
			notify := make(chan struct{}, 64)
			s := New()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == dsh.MuxPath {
					conn, e := websocket.Accept(w, r, nil)
					if e != nil {
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
						switch frame.Endpoint {
						case "session/follow":
							follows.Add(1)
						case "session/control":
							controls.Add(1)
							baseline := json.RawMessage(`{"type":"baseline","value":{"projections":{"target":{"asOfSeq":10,"values":{"userQuestions":{"active":[{"callId":"qa-call","state":"continued","questions":[{"id":"q","question":"Answer"}]}],"settled":[]},"inbox":{"next-turn":[],"next-step":[]}}}}}}`)
							if wsjson.Write(r.Context(), conn, dsh.ServerFrame{Type: dsh.FrameItem, StreamID: frame.StreamID, Value: baseline}) != nil {
								return
							}
						}
					}
				}
				var request struct {
					RPCID string `json:"rpcId"`
				}
				if json.NewDecoder(r.Body).Decode(&request) != nil {
					http.Error(w, "bad fixture request", 400)
					return
				}
				var value any = map[string]any{}
				switch strings.TrimPrefix(r.URL.Path, "/api/") {
				case "session/list":
					value = map[string]any{"items": []any{map[string]any{"sessionId": "target", "blank": false, "running": false, "agentAvailable": true, "updatedAt": 1}}}
				case "session/create", "session/fork":
					value = map[string]any{"sessionId": "target"}
				case "session/projections":
					value = map[string]any{"asOfSeq": 2, "values": map[string]any{}}
					// Other selection-owned readers are not the subject of this fixture.
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": request.RPCID, "result": map[string]any{"ok": true, "value": value}})
			}))
			client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
			mux := &dsh.Mux{Endpoint: client.Endpoint}
			t.Cleanup(func() { s.Detach(); mux.Close(); srv.Close() })
			s.Attach(client, func() {
				select {
				case notify <- struct{}{}:
				default:
				}
			})
			s.AttachMux(mux)
			ctx := context.Background()
			switch entry {
			case "initial-list":
				s.RefreshList(ctx)
			case "create", "fork":
				s.mu.Lock()
				s.selectedID = "source"
				s.mu.Unlock()
				if entry == "create" {
					s.runCreateSession(ctx, client, s.eventsGeneration, "/owned", "")
				} else {
					s.runForkSession(ctx, client, s.eventsGeneration, "source")
				}
			case "same-select":
				s.mu.Lock()
				s.selectedID = "target"
				s.historyReady = true
				s.mu.Unlock()
				s.Select("target")
			case "same-workspace-select":
				s.applyWorkspaceBaseline(json.RawMessage(`{"type":"baseline","value":{"scratchCwd":"/scratch","items":[{"workspaceId":"project","path":"/project","title":"Project","sessionIds":["target"]}],"pinnedSessionIds":[],"archivedSessionIds":[]}}`))
				s.mu.Lock()
				s.selectedID = "target"
				s.mu.Unlock()
				target, e := s.CaptureWorkspaceSessionTarget(s.eventsGeneration, "project", "target")
				if e != nil {
					t.Fatal(e)
				}
				id, e := target.Create(ctx)
				if e != nil || !target.Select(id) {
					t.Fatalf("workspace select: %s %v", id, e)
				}
			}
			deadline := time.NewTimer(3 * time.Second)
			defer deadline.Stop()
			for {
				v := s.Snapshot()
				if v.SelectedID == "target" && v.ControlNote == "baseline" && strings.Contains(string(v.QuestionProjection.Questions), "qa-call") {
					break
				}
				select {
				case <-notify:
				case <-deadline.C:
					t.Fatalf("%s omitted control: selected=%s note=%s questions=%s", entry, v.SelectedID, v.ControlNote, v.QuestionProjection.Questions)
				}
			}
			s.mu.Lock()
			owner := s.controlWatchStreamCtx
			followGeneration := s.followGeneration
			note := s.controlNote
			s.mu.Unlock()
			s.Select("target")
			s.WatchControl(ctx)
			s.mu.Lock()
			same := s.controlWatchStreamCtx == owner && s.controlNote == note && s.followGeneration == followGeneration
			s.mu.Unlock()
			if !same || controls.Load() != 1 {
				t.Fatalf("same selection restarted/cleared control: ownerSame=%v opens=%d", same, controls.Load())
			}
			if strings.HasPrefix(entry, "same-") && follows.Load() != 0 {
				t.Fatal("same-ID control admission restarted follow")
			}
		})
	}
}

func TestControlAdmissionRetiresPriorContextAndHostGeneration(t *testing.T) {
	s := New()
	defer s.Detach()
	client, mux := &dsh.Client{}, &dsh.Mux{}
	s.client, s.mux, s.selectedID = client, mux, "target"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.mu.Lock()
	first, _ := s.beginControlWatchLocked(ctx)
	old := s.controlWatchStreamCtx
	s.controlNote = "baseline"
	duplicate, changed := s.beginControlWatchLocked(ctx)
	same := s.controlWatchStreamCtx == old && s.controlNote == "baseline"
	s.eventsGeneration++
	replacement, cleared := s.beginControlWatchLocked(ctx)
	fresh := s.controlWatchStreamCtx
	s.mu.Unlock()
	if first == nil || duplicate != nil || changed || !same || replacement == nil || !cleared || old.Err() == nil || fresh == old {
		t.Fatal("control admission did not preserve/detach the exact owner")
	}
	cancel()
	if fresh.Err() == nil {
		t.Fatal("parent cancellation failed to revoke control")
	}
}
