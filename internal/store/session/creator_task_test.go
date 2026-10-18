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

// One synthetic fixture invocation covers only Creator admission. Every RPC is
// explicitly allowlisted; no live Host, model, settings write, or user data.
func TestCreatorTaskEntryFixture(t *testing.T) {
	for _, mode := range []string{"success", "missing", "broken", "hidden", "create-unknown", "wrong-receipt", "apply-failed", "leave-during-create", "selection-roundtrip", "host-switch", "same-client-account", "stale-confirm"} {
		t.Run(mode, func(t *testing.T) {
			s := New()
			var c *CreatorTask
			creates, applies, lists := 0, 0, 0
			var requested string
			var srv *httptest.Server
			srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					RPCID   any `json:"rpcId"`
					Payload struct {
						Args map[string]json.RawMessage `json:"args"`
					} `json:"payload"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
					return
				}
				respond := func(v any) {
					_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": body.RPCID, "result": map[string]any{"ok": true, "value": v}})
				}
				switch strings.TrimPrefix(r.URL.Path, "/api/") {
				case "agentPresets/list":
					rows := []map[string]any{{"id": "standard", "isDefault": true}}
					if mode != "missing" {
						row := map[string]any{"id": "cordis"}
						if mode == "broken" {
							row["broken"] = "missing package"
						}
						if mode == "hidden" {
							row["hidden"] = true
						}
						rows = append(rows, row)
					}
					respond(map[string]any{"presets": rows})
				case "session/create":
					creates++
					var req map[string]any
					_ = json.Unmarshal(body.Payload.Args["request"], &req)
					requested, _ = req["sessionId"].(string)
					if len(requested) != 44 || !strings.HasPrefix(requested, "session-") || req["cwd"] != "/fixture-workspace" || len(req) != 2 {
						t.Errorf("wrong create contract: %+v", req)
					}
					switch mode {
					case "create-unknown":
						http.Error(w, "fixture lost receipt", 502)
						return
					case "wrong-receipt":
						respond(map[string]any{"sessionId": "old-session"})
						return
					case "leave-during-create":
						c.ObserveVisible(false)
					case "selection-roundtrip":
						s.mu.Lock()
						s.selectSessionLocked("other")
						s.selectSessionLocked("old-session")
						s.mu.Unlock()
					case "host-switch":
						s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
					case "same-client-account":
						s.mu.Lock()
						s.client.Endpoint.Cookie = "new-fixture-account"
						s.mu.Unlock()
					}
					respond(map[string]any{"sessionId": requested})
				case "agentPresets/select":
					applies++
					if string(body.Payload.Args["agentId"]) != `"`+requested+`"` || string(body.Payload.Args["agentPreset"]) != `"cordis"` || len(body.Payload.Args) != 2 {
						t.Errorf("wrong exact target: %+v", body.Payload.Args)
					}
					if mode == "apply-failed" {
						http.Error(w, "fixture apply refusal", 409)
						return
					}
					respond("cordis")
				case "session/list":
					lists++
					respond(map[string]any{"items": []map[string]any{{"sessionId": requested, "cwd": "/fixture-workspace", "blank": true, "agentAvailable": true}}})
				default:
					t.Errorf("forbidden RPC %s", r.URL.Path)
					http.Error(w, "unexpected", 500)
				}
			}))
			defer srv.Close()
			s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
			s.SetWorkspaceCwd("/fixture-workspace")
			s.mu.Lock()
			s.selectSessionLocked("old-session")
			s.selectedAgentPresetLabel = "Standard"
			s.mu.Unlock()
			c = NewCreatorTask(s, nil)
			c.ObserveVisible(true)
			c.Prepare()
			before := c.Snapshot()
			if !before.Review || before.SessionID == "" {
				t.Fatalf("review missing: %+v", before)
			}
			if mode == "stale-confirm" {
				c.ObserveVisible(false)
				c.ObserveVisible(true)
			}
			ok := c.Create(context.Background(), before.Ticket)
			after := c.Snapshot()
			c.Create(context.Background(), before.Ticket) // Replayed rendered confirmation cannot create twice.
			if s.Snapshot().SelectedID != "old-session" || s.Snapshot().SelectedAgentPreset != "Standard" {
				t.Fatal("asynchronous admission changed the active conversation")
			}
			switch mode {
			case "success":
				if !ok || !after.Created || !after.Applied || creates != 1 || applies != 1 {
					t.Fatalf("success state: %+v %d/%d", after, creates, applies)
				}
				if !c.Open(after.Ticket, true) || s.Snapshot().SelectedID != requested || s.Snapshot().SelectedAgentPreset != "cordis" {
					t.Fatal("did not open exact confirmed Creator target")
				}
			case "missing", "broken", "hidden", "stale-confirm":
				if ok || creates != 0 || applies != 0 {
					t.Fatalf("invalid admission wrote: %+v", after)
				}
			case "create-unknown", "wrong-receipt":
				if ok || after.Created || after.SessionID != requested || creates != 1 || applies != 0 {
					t.Fatalf("lost attempted identity: %+v", after)
				}
				c.Prepare()
				if c.Snapshot().SessionID != requested {
					t.Fatal("unknown outcome allowed reconstruction")
				}
				c.Locate(context.Background(), c.Snapshot().Ticket)
				found := c.Snapshot()
				if !found.Created || found.Applied || lists != 1 || creates != 1 || applies != 0 {
					t.Fatalf("readback recreated/applied: %+v", found)
				}
				if !c.Open(found.Ticket, false) || s.Snapshot().SelectedID != requested {
					t.Fatal("could not open located existing target")
				}
			case "apply-failed":
				if ok || !after.Created || after.Applied || creates != 1 || applies != 1 || after.Error == "" {
					t.Fatalf("lost failed apply receipt: %+v", after)
				}
				if c.Open(after.Ticket, true) {
					t.Fatal("automatic open hid failed apply")
				}
				if !c.Open(after.Ticket, false) {
					t.Fatal("explicit existing-target recovery failed")
				}
			default:
				if ok || !after.Created || after.Applied || creates != 1 || applies != 0 {
					t.Fatalf("stale continuation applied: %+v", after)
				}
				if c.Open(after.Ticket, true) {
					t.Fatal("late result stole navigation")
				}
				if mode == "host-switch" || mode == "same-client-account" {
					if c.Open(after.Ticket, false) {
						t.Fatal("old Host target opened on new owner")
					}
				}
			}
		})
	}
}
