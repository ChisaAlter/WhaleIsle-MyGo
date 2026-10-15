package pluginrpc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDshbotDescribeTypedAndRoster(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/dshbot/describe") {
			http.NotFound(w, r)
			return
		}
		var req clientRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		value := DshbotDescribeResult{
			View: DshbotSettingsView{
				NS:       "dsh-bot",
				Revision: 7,
				Value: DshbotCatalogValue{
					Items: []DshbotCatalogItem{
						{ID: "b2", Kind: "bot", Name: "Beta", SessionID: "sess-b2", Pinned: true},
						{ID: "b1", Kind: "bot", Name: "Alpha", SessionID: "sess-b1"},
						{ID: "r1", Kind: "room", Name: "Room", SessionID: "sess-r1", MemberBotIDs: []string{"b1"}},
						{ID: "h1", Kind: "bot", Name: "Hidden", SessionID: "sess-h", Hidden: true},
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(serverResponse{
			Type:  "server-response",
			RPCID: req.RPCID,
			Result: mustRaw(t, resultEnvelope{
				OK:    true,
				Value: mustRaw(t, value),
			}),
		})
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL}
	res, err := c.DshbotDescribeTyped(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if res.View.Revision != 7 || len(res.View.Value.Items) != 4 {
		t.Fatalf("describe = %+v", res)
	}
	rows := RosterRowsFromDescribe(res, false)
	if len(rows) != 3 {
		t.Fatalf("rows = %d, want 3 (hidden excluded)", len(rows))
	}
	if rows[0].ID != "b2" || !rows[0].Pinned {
		t.Fatalf("pinned first = %+v", rows[0])
	}
}

func TestDshbotGroupRuntimeAndActivity(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/dshbot/group/runtime", func(w http.ResponseWriter, r *http.Request) {
		var req clientRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		payload, _ := json.Marshal(req.Payload)
		if !strings.Contains(string(payload), "room-1") {
			t.Fatalf("payload = %s", payload)
		}
		_ = json.NewEncoder(w).Encode(serverResponse{
			Type:  "server-response",
			RPCID: req.RPCID,
			Result: mustRaw(t, resultEnvelope{
				OK: true,
				Value: mustRaw(t, DshbotGroupRuntimeResult{
					RoomID: "room-1", SessionID: "sess-room",
					Members: []DshbotGroupRuntimeMember{{BotID: "b1", Status: "idle"}},
				}),
			}),
		})
	})
	mux.HandleFunc("/dshbot/bot/activity", func(w http.ResponseWriter, r *http.Request) {
		var req clientRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		_ = json.NewEncoder(w).Encode(serverResponse{
			Type:  "server-response",
			RPCID: req.RPCID,
			Result: mustRaw(t, resultEnvelope{
				OK: true,
				Value: mustRaw(t, DshbotActivityResult{
					Entries: []DshbotActivityEntry{{BotID: "b1", Preview: "hi", Attention: true}},
				}),
			}),
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := &Client{BaseURL: srv.URL}
	rt, err := c.DshbotGroupRuntime(context.Background(), "room-1")
	if err != nil || len(rt.Members) != 1 {
		t.Fatalf("runtime = %+v err=%v", rt, err)
	}
	act, err := c.DshbotActivityTyped(context.Background(), nil)
	if err != nil || len(act.Entries) != 1 || act.Entries[0].Preview != "hi" {
		t.Fatalf("activity = %+v err=%v", act, err)
	}
}

func TestDshbotOpenUsesSelectedIdentity(t *testing.T) {
	for _, mismatched := range []bool{false, true} {
		t.Run(map[bool]string{false: "canonical", true: "changed"}[mismatched], func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req struct {
					RPCID   string         `json:"rpcId"`
					Payload map[string]any `json:"payload"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Error(err)
					http.Error(w, "invalid request", http.StatusBadRequest)
					return
				}
				var value any
				switch r.URL.Path {
				case "/dshbot/describe":
					if req.Payload["botId"] != "chosen" {
						t.Errorf("describe lost botId: %#v", req.Payload)
					}
					value = DshbotDescribeResult{View: DshbotSettingsView{NS: "dsh-bot", Revision: 7}}
				case "/dshbot/bot/open":
					if req.Payload["id"] != "chosen" || req.Payload["revision"] != float64(7) || req.Payload["scratchCwd"] != "/scratch" {
						t.Errorf("open payload: %#v", req.Payload)
					}
					sid := "canonical"
					if mismatched {
						sid = "other-session"
					}
					value = map[string]any{"item": DshbotCatalogItem{ID: "chosen", SessionID: sid}, "sessionId": sid}
				default:
					t.Errorf("unexpected request: %s", r.URL.Path)
					http.NotFound(w, r)
					return
				}
				_ = json.NewEncoder(w).Encode(serverResponse{Type: "server-response", RPCID: req.RPCID,
					Result: mustRaw(t, resultEnvelope{OK: true, Value: mustRaw(t, value)})})
			}))
			defer srv.Close()
			client := &Client{BaseURL: srv.URL}
			if _, err := client.DshbotDescribeTyped(context.Background(), "chosen"); err != nil {
				t.Fatal(err)
			}
			sid, err := client.DshbotOpen(context.Background(), DshbotCatalogItem{ID: "chosen", SessionID: "canonical"}, 7, "/scratch")
			if mismatched {
				if err == nil || sid != "" {
					t.Fatalf("changed session accepted: %q, %v", sid, err)
				}
			} else if err != nil || sid != "canonical" {
				t.Fatalf("open = %q, %v", sid, err)
			}
		})
	}
}
