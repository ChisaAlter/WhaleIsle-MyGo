package session

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"whaleisle/internal/dsh"
)

func TestProjectLeadExitNavigatesSameWorkspaceAndWaitsReady(t *testing.T) {
	for _, registered := range []bool{true, false} {
		t.Run(map[bool]string{true: "same workspace", false: "scratch"}[registered], func(t *testing.T) {
			followOpened, release := make(chan struct{}), make(chan struct{})
			var once sync.Once
			var mu sync.Mutex
			var createRequest map[string]any
			var mutations []string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
						if frame.Endpoint == "session/follow" {
							once.Do(func() { close(followOpened) })
							select {
							case <-release:
							case <-r.Context().Done():
								return
							}
							raw := json.RawMessage(`{"type":"snapshot","records":[]}`)
							if err = wsjson.Write(r.Context(), conn, dsh.ServerFrame{Type: dsh.FrameItem, StreamID: frame.StreamID, Value: raw}); err != nil {
								return
							}
						}
					}
				}
				var body struct {
					RPCID   string `json:"rpcId"`
					Payload struct {
						Args struct {
							Request map[string]any `json:"request"`
						} `json:"args"`
					} `json:"payload"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					http.Error(w, err.Error(), 400)
					return
				}
				method := strings.TrimPrefix(r.URL.Path, "/api/")
				var value any = map[string]any{}
				mu.Lock()
				switch method {
				case "session/create":
					createRequest = body.Payload.Args.Request
					mutations = append(mutations, method)
					value = map[string]any{"sessionId": "ordinary-session"}
				case "session/blankReuse":
					value = map[string]any{"reusable": false}
				default:
					if strings.Contains(method, "delete") || strings.Contains(method, "archive") || strings.Contains(method, "stop") {
						mutations = append(mutations, method)
					}
				}
				mu.Unlock()
				_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": body.RPCID, "result": map[string]any{"ok": true, "value": value}})
			}))
			defer srv.Close()
			client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
			mux := &dsh.Mux{Endpoint: client.Endpoint}
			defer mux.Close()
			s := New()
			s.Attach(client, nil)
			s.AttachMux(mux)
			baseline := `{"type":"baseline","value":{"scratchCwd":"/scratch","items":[],"pinnedSessionIds":[],"archivedSessionIds":[]}}`
			if registered {
				baseline = `{"type":"baseline","value":{"scratchCwd":"/scratch","items":[{"workspaceId":"owned","path":"/project","sessionIds":["project-lead"]}],"pinnedSessionIds":[],"archivedSessionIds":[]}}`
			}
			s.applyWorkspaceBaseline(json.RawMessage(baseline))
			s.mu.Lock()
			s.selectedID = "project-lead"
			s.cwd = "/project"
			s.items = []Item{{SessionID: "project-lead", Cwd: "/project", Presentation: &SessionPresentation{Owner: "project", WorkingDirectory: "/project"}}}
			s.mu.Unlock()
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- s.LeaveProjectLead(ctx, "project-lead") }()
			select {
			case <-followOpened:
			case <-ctx.Done():
				t.Fatal("ordinary follow never opened")
			}
			select {
			case err := <-done:
				t.Fatalf("disable could proceed before ordinary session ready:%v", err)
			default:
			}
			close(release)
			select {
			case err := <-done:
				if err != nil {
					t.Fatal(err)
				}
			case <-ctx.Done():
				t.Fatal("exit did not await ready")
			}
			view := s.Snapshot()
			if view.SelectedID != "ordinary-session" || !view.HistoryReady || s.ProjectLeadSelected() {
				t.Fatalf("Lead did not exit to ready normal session:%+v", view)
			}
			mu.Lock()
			defer mu.Unlock()
			if registered && createRequest["workspaceId"] != "owned" {
				t.Fatalf("lost same directory workspace:%v", createRequest)
			}
			if !registered && createRequest["cwd"] != "/scratch" {
				t.Fatalf("lost scratch fallback:%v", createRequest)
			}
			if len(mutations) != 1 || mutations[0] != "session/create" {
				t.Fatalf("exit mutated Project work:%v", mutations)
			}
			if len(view.Items) < 2 || view.Items[0].SessionID != "project-lead" {
				t.Fatal("Lead identity/data removed")
			}
		})
	}
}

func TestProjectLeadExitIgnoresOrdinaryAndRejectsUnreadyOwner(t *testing.T) {
	s := New()
	s.items = []Item{{SessionID: "normal"}}
	s.selectedID = "normal"
	if err := s.LeaveProjectLead(context.Background()); err != nil {
		t.Fatal(err)
	}
	s.items[0].Presentation = &SessionPresentation{Owner: "project"}
	if err := s.LeaveProjectLead(context.Background()); !errors.Is(err, ErrNoClient) {
		t.Fatalf("disconnected Lead exit falsely succeeded:%v", err)
	}
	// Waiting is bound to Host and selection, even if an unrelated conversation
	// becomes ready. A replacement must not authorize disabling the old Lead.
	s.mu.Lock()
	s.eventsGeneration = 7
	s.selectedID = "ordinary"
	s.historyReady = false
	s.mu.Unlock()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.waitProjectLeaveReady(ctx, 7, "ordinary"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := s.waitProjectLeaveReady(context.Background(), 6, "ordinary"); err == nil {
		t.Fatal("new Host authorized old Lead exit")
	}
	if err := s.waitProjectLeaveReady(context.Background(), 7, "different"); err == nil {
		t.Fatal("new navigation authorized old Lead exit")
	}
	s.mu.Lock()
	s.followGeneration = 9
	s.historyReady = true
	s.mu.Unlock()
	if err := s.waitProjectLeaveReady(context.Background(), 7, "ordinary", 8); err == nil {
		t.Fatal("away-and-back selection authorized old Lead exit")
	}
}
