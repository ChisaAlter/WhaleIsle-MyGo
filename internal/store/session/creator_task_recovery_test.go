package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"whaleisle/internal/dsh"
)

func TestCreatorRetainedTargetRecoveryDoesNotCreateOrApplyToConversation(t *testing.T) {
	for _, reusable := range []bool{false, true} {
		t.Run(map[bool]string{false: "target-used", true: "blank-target"}[reusable], func(t *testing.T) {
			creates, applies := 0, 0
			var target string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var frame struct {
					RPCID   any `json:"rpcId"`
					Payload struct {
						Args map[string]json.RawMessage `json:"args"`
					} `json:"payload"`
				}
				if err := json.NewDecoder(r.Body).Decode(&frame); err != nil {
					t.Error(err)
					return
				}
				respond := func(value any) {
					_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": frame.RPCID, "result": map[string]any{"ok": true, "value": value}})
				}
				switch strings.TrimPrefix(r.URL.Path, "/api/") {
				case "agentPresets/list":
					respond(map[string]any{"presets": []map[string]any{{"id": "cordis"}}})
				case "session/create":
					creates++
					var request struct {
						SessionID string `json:"sessionId"`
					}
					_ = json.Unmarshal(frame.Payload.Args["request"], &request)
					target = request.SessionID
					respond(map[string]any{"sessionId": target})
				case "agentPresets/select":
					applies++
					if string(frame.Payload.Args["agentId"]) != `"`+target+`"` {
						t.Errorf("preset applied to another conversation: %s", frame.Payload.Args["agentId"])
					}
					if applies == 1 {
						http.Error(w, "lost apply receipt", http.StatusBadGateway)
						return
					}
					respond("cordis")
				case "session/blankReuse":
					var request struct {
						SessionID string `json:"sessionId"`
					}
					_ = json.Unmarshal(frame.Payload.Args["request"], &request)
					if request.SessionID != target {
						t.Errorf("checked another target: %q", request.SessionID)
					}
					respond(map[string]any{"reusable": reusable})
				default:
					t.Errorf("Creator performed forbidden operation %s", r.URL.Path)
					http.Error(w, "unexpected", http.StatusBadRequest)
				}
			}))
			defer server.Close()
			s := New()
			s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL}}, nil)
			s.SetWorkspaceCwd("/fixture")
			s.mu.Lock()
			s.selectSessionLocked("existing-nonempty")
			s.mu.Unlock()
			c := NewCreatorTask(s, nil)
			c.ObserveVisible(true)
			c.Prepare()
			ticket := c.Snapshot().Ticket
			if c.Create(context.Background(), ticket) {
				t.Fatal("lost apply receipt reported ready")
			}
			if c.Snapshot().SessionID != target || !c.Snapshot().Created {
				t.Fatal("created target was not retained")
			}
			if got := c.ApplyToExisting(context.Background(), ticket); got != reusable {
				t.Fatalf("recovery = %v; reusable = %v", got, reusable)
			}
			if creates != 1 || applies != 1+map[bool]int{false: 0, true: 1}[reusable] || s.SelectedSessionID() != "existing-nonempty" {
				t.Fatalf("recovery changed creation/selection: creates=%d applies=%d selected=%q", creates, applies, s.SelectedSessionID())
			}
			if reusable {
				if !c.Open(ticket, true) || s.SelectedSessionID() != target {
					t.Fatal("confirmed result did not open its exact target")
				}
			}
		})
	}
}
