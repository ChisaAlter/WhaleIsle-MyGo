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
