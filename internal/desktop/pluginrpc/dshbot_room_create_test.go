package pluginrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roomCreateTransport func(*http.Request) (*http.Response, error)

func (f roomCreateTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func roomFixtureCommand() DshbotRoomCreateCommand {
	return DshbotRoomCreateCommand{ID: "room-fixture", SessionID: "session-room-fixture", Expected: DshbotRoomCatalog{Revision: 7, Items: []DshbotRoomItem{{DshbotCatalogItem: DshbotCatalogItem{ID: "a", Kind: "bot", Name: "A", SessionID: "sa"}}, {DshbotCatalogItem: DshbotCatalogItem{ID: "b", Kind: "bot", Name: "B", SessionID: "sb"}}}}, Draft: DshbotRoomDraft{Name: "Fixture room", Description: "Full description", ScratchCwd: "/fixture", MemberBotIDs: []string{"a", "b"}, MaxRounds: 3, MaxSpeaks: 10}}
}
func roomFixtureCreated(cmd DshbotRoomCreateCommand) DshbotRoomItem {
	return DshbotRoomItem{DshbotCatalogItem: DshbotCatalogItem{ID: cmd.ID, SessionID: cmd.SessionID, Kind: "room", Name: cmd.Draft.Name, WorkspaceID: cmd.Draft.WorkspaceID, MemberBotIDs: append([]string(nil), cmd.Draft.MemberBotIDs...)}, Description: cmd.Draft.Description, MaxRounds: cmd.Draft.MaxRounds, MaxSpeaks: cmd.Draft.MaxSpeaks}
}
func TestRoomCreateWireAndUnknown(t *testing.T) {
	cmd := roomFixtureCommand()
	calls := 0
	c := Client{BaseURL: "http://127.0.0.1", HTTPClient: &http.Client{Transport: roomCreateTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		var req struct {
			RPCID   string                     `json:"rpcId"`
			Method  string                     `json:"method"`
			Payload map[string]json.RawMessage `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req.Method != "bot/create" || r.URL.Path != "/dshbot/bot/create" {
			t.Fatal("wrong mutation", req.Method)
		}
		for k, v := range map[string]string{"kind": `"room"`, "id": `"room-fixture"`, "sessionId": `"session-room-fixture"`, "revision": "7", "maxRounds": "3", "maxSpeaks": "10", "memberBotIds": `["a","b"]`} {
			if string(req.Payload[k]) != v {
				t.Fatalf("%s: %s", k, req.Payload[k])
			}
		}
		if _, ok := req.Payload["requestId"]; ok {
			t.Fatal("invented Host idempotency")
		}
		item := roomFixtureCreated(cmd)
		items := append(cmd.Expected.Items, item)
		body, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": req.RPCID, "result": map[string]any{"ok": true, "value": map[string]any{"view": map[string]any{"ns": "dsh-bot", "revision": 8, "value": map[string]any{"items": items}}, "item": item, "sessionId": item.SessionID}}})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(body)))}, nil
	})}}
	result, err := c.DshbotCreateRoom(context.Background(), cmd)
	if err != nil || result.Item.ID != cmd.ID || calls != 1 {
		t.Fatal(result, err, calls)
	}
	c.HTTPClient.Transport = roomCreateTransport(func(*http.Request) (*http.Response, error) { calls++; return nil, fmt.Errorf("response lost") })
	out, err := c.DshbotCreateRoom(context.Background(), cmd)
	if err == nil || out.Rejected || !out.Attempted || calls != 2 {
		t.Fatal("unknown creation must not retry", out, err, calls)
	}
	bad := cmd
	bad.Draft.MemberBotIDs = []string{"a", "room-id"}
	out, err = c.DshbotCreateRoom(context.Background(), bad)
	if err == nil || !out.Rejected || out.Attempted || calls != 2 {
		t.Fatal("invalid member dispatched", out, err)
	}
	for _, raw := range []string{`{"ns":"dsh-bot","value":{"items":[]}}`, `{"ns":"other","revision":0,"value":{"items":[]}}`, `{"ns":"dsh-bot","revision":0,"value":{}}`} {
		if _, err := decodeDshbotRoomCatalog(json.RawMessage(raw)); err == nil {
			t.Fatal("guessed missing catalog", raw)
		}
	}
}
func TestRoomCreateValidationAndExactReadback(t *testing.T) {
	cmd := roomFixtureCommand()
	empty := CloneDshbotRoomCatalog(cmd.Expected)
	empty.Items[0].MemberBotIDs = []string{}
	if !SameDshbotRoomCatalog(empty, cmd.Expected) {
		t.Fatal("empty member lists incorrectly changed catalog")
	}
	good := roomFixtureCreated(cmd)
	view := cmd.Expected
	view.Revision++
	view.Items = append(view.Items, good)
	if _, err := DshbotCreatedRoom(cmd, view); err != nil {
		t.Fatal(err)
	}
	view.Items[len(view.Items)-1].SessionID = "other"
	if _, err := DshbotCreatedRoom(cmd, view); err == nil {
		t.Fatal("wrong canonical session accepted")
	}
	for _, limits := range [][2]int{{0, 1}, {4, 1}, {1, 0}, {1, 11}} {
		bad := roomFixtureCommand()
		bad.Draft.MaxRounds = limits[0]
		bad.Draft.MaxSpeaks = limits[1]
		if ValidateDshbotRoomCreate(bad) == nil {
			t.Fatal("bad limits accepted", limits)
		}
	}
	bad := roomFixtureCommand()
	bad.Draft.MemberBotIDs = []string{"a", "a"}
	if ValidateDshbotRoomCreate(bad) == nil {
		t.Fatal("duplicate member accepted")
	}
	bad = roomFixtureCommand()
	bad.Expected.Items[1].Kind = "room"
	if ValidateDshbotRoomCreate(bad) == nil {
		t.Fatal("room as member accepted")
	}
	bad = roomFixtureCommand()
	bad.Draft.ScratchCwd = ""
	if ValidateDshbotRoomCreate(bad) == nil {
		t.Fatal("missing cwd accepted")
	}
}

func TestRoomCreateFinalRejectionBoundary(t *testing.T) {
	for _, test := range []struct {
		message  string
		rejected bool
	}{
		{"Bot catalog changed. Refresh and try again.", true},
		{"Session creation failed: Bot catalog changed. Refresh and try again.", false},
		{"Bot Session creation failed and catalog rollback also failed: Group limits are invalid.", false},
	} {
		t.Run(test.message, func(t *testing.T) {
			calls := 0
			c := Client{BaseURL: "http://127.0.0.1", HTTPClient: &http.Client{Transport: roomCreateTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				var request struct {
					RPCID string `json:"rpcId"`
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Fatal(err)
				}
				body, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": request.RPCID, "result": map[string]any{"ok": false, "error": map[string]string{"code": "error", "message": test.message}}})
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(body)))}, nil
			})}}
			result, err := c.DshbotCreateRoom(context.Background(), roomFixtureCommand())
			if err == nil || result.Rejected != test.rejected || !result.Attempted || calls != 1 {
				t.Fatal("unsafe rejection boundary", result, err, calls)
			}
		})
	}
}
