package pluginrpc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roomRuntimeTransport func(*http.Request) (*http.Response, error)

func (f roomRuntimeTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	response, err := f(r)
	return fixtureResponseIdentity(r, response, err)
}
func roomRuntimeFixtureCatalog() DshbotRoomCatalog {
	return DshbotRoomCatalog{Revision: 9, Items: []DshbotRoomItem{{DshbotCatalogItem: DshbotCatalogItem{ID: "room", Kind: "room", Name: "Group", SessionID: "room-session", MemberBotIDs: []string{"a", "b"}}}, {DshbotCatalogItem: DshbotCatalogItem{ID: "a", Kind: "bot", Name: "Alpha", SessionID: "personal-a"}}, {DshbotCatalogItem: DshbotCatalogItem{ID: "b", Kind: "bot", Name: "Beta", SessionID: "personal-b"}}}}
}
func TestRoomRuntimeReadIdentityAndFullStatusFixture(t *testing.T) {
	sid := "dshbot-group-member-v3-" + base64.RawURLEncoding.EncodeToString([]byte(`["room-session","a"]`))
	raw := json.RawMessage(fmt.Sprintf(`{"roomId":"room","sessionId":"room-session","members":[{"roomSessionId":"room-session","botId":"a","sessionId":%q,"status":"pending","lastError":{"code":"LATE_RESULT","message":"full failure text","details":{"future":true}},"pending":{"kind":"approval","requestId":"pending-1"},"future":{"retain":[1,2,3]}}],"extra":{"notLost":true}}`, sid))
	catalog := roomRuntimeFixtureCatalog()
	calls := []string{}
	failRuntime := false
	c := Client{BaseURL: "http://127.0.0.1", HTTPClient: &http.Client{Transport: roomRuntimeTransport(func(r *http.Request) (*http.Response, error) {
		var req struct {
			RPCID   string                     `json:"rpcId"`
			Method  string                     `json:"method"`
			Payload map[string]json.RawMessage `json:"payload"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		calls = append(calls, req.Method)
		var value any
		switch req.Method {
		case "describe":
			value = map[string]any{"view": map[string]any{"ns": "dsh-bot", "revision": 9, "value": map[string]any{"items": catalog.Items}}}
		case "group/runtime":
			if string(req.Payload["roomId"]) != `"room"` {
				t.Fatal("wrong room")
			}
			if failRuntime {
				return nil, fmt.Errorf("runtime unavailable")
			}
			value = raw
		default:
			t.Fatal("refresh attempted activation", req.Method)
		}
		body, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": req.RPCID, "result": map[string]any{"ok": true, "value": value}})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(body)))}, nil
	})}}
	v, err := c.DshbotRoomRuntime(context.Background(), "room")
	if err != nil || len(calls) != 2 {
		t.Fatal(err, calls)
	}
	m, err := DshbotRoomRuntimeTarget(v, sid)
	if err != nil || m.Name != "Alpha" || !DshbotRoomRuntimeAttention(m) || !strings.Contains(string(m.Raw), "future") || !strings.Contains(string(v.Raw), "notLost") {
		t.Fatal("incomplete runtime", m, err)
	}
	clone := CloneDshbotRoomRuntime(v)
	clone.Members[0].Raw[0] = ' '
	if v.Members[0].Raw[0] != '{' {
		t.Fatal("raw alias")
	}
	for _, bad := range []string{`{"roomId":"other","sessionId":"room-session","members":[]}`, `{"roomId":"room","sessionId":"other","members":[]}`, `{"roomId":"room","sessionId":"room-session"}`, `{"roomId":"room","sessionId":"room-session","members":[null]}`} {
		if _, err := decodeDshbotRoomRuntime(catalog, "room", json.RawMessage(bad)); err == nil {
			t.Fatal("bad runtime accepted", bad)
		}
	}
	for _, badSID := range []string{"personal-a", "room-session", "dshbot-group-member-v3-" + base64.RawURLEncoding.EncodeToString([]byte(`["other-room","a"]`))} {
		bad := strings.ReplaceAll(string(raw), sid, badSID)
		view, err := decodeDshbotRoomRuntime(catalog, "room", json.RawMessage(bad))
		if err != nil {
			t.Fatal(err)
		}
		if _, err = DshbotRoomRuntimeTarget(view, badSID); err == nil {
			t.Fatal("foreign member session opened", badSID)
		}
	}
	gone := CloneDshbotRoomRuntime(v)
	gone.Room.MemberBotIDs = []string{"b"}
	if _, err = DshbotRoomRuntimeTarget(gone, sid); err == nil {
		t.Fatal("removed member opened")
	}
	failRuntime = true
	if _, err = c.DshbotRoomRuntime(context.Background(), "room"); err == nil || len(calls) != 4 {
		t.Fatal("runtime failure hidden or retried", err, calls)
	}
}
