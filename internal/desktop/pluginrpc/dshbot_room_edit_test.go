package pluginrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

type roomEditTransport func(*http.Request) (*http.Response, error)

func (f roomEditTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

const roomEditFixtureJSON = `{"id":"room","kind":"room","sessionId":"canonical-room","name":"Fixture group","title":"","description":"preserve full description","workspaceId":"workspace-1","sectionId":"section-1","pinned":false,"hidden":false,"pinOrder":0,"readInitialized":true,"lastSeenSeq":4,"createdAt":1,"updatedAt":2,"objective":"","objectiveMaxRounds":16,"memoryReview":true,"notify":true,"allowedSenderIds":[],"memberBotIds":["a","b"],"maxRounds":3,"maxSpeaks":10,"avatar":{"kind":"blob","shape":"circle","color":"blue","crop":"","dataUrl":""},"model":{"provider":"","model":"","reasoningEffort":""},"capabilities":{"tools":{"mode":"all","names":[],"future":{"guard":true}},"skills":{"mode":"all","names":[]},"mcp":{"mode":"all","names":[]},"extra":{"nested":[1,2]}},"holds":{},"inbox":[],"futureTop":{"retain":{"bytes":"value"}}}`

func roomEditFixtureStatus(t *testing.T) DshbotRoomEditStatus {
	t.Helper()
	var p map[string]json.RawMessage
	json.Unmarshal([]byte(roomEditFixtureJSON), &p)
	raw := roomEditFixtureView(p, 7)
	s, err := decodeDshbotRoomEdit(raw, "room")
	if err != nil || s.WriteError != "" {
		t.Fatal(s.WriteError, err)
	}
	return s
}
func roomEditFixtureView(p map[string]json.RawMessage, revision int) json.RawMessage {
	v, _ := json.Marshal(map[string]any{"ns": "dsh-bot", "revision": revision, "value": map[string]any{"items": []any{map[string]string{"id": "a", "kind": "bot", "name": "Alpha", "sessionId": "sa"}, map[string]string{"id": "b", "kind": "bot", "name": "Beta", "sessionId": "sb"}, map[string]string{"id": "c", "kind": "bot", "name": "Gamma", "sessionId": "sc"}, p}, "sections": []any{map[string]string{"id": "section-1"}}}})
	return v
}
func TestRoomEditExactWirePreservationAndUnknown(t *testing.T) {
	status := roomEditFixtureStatus(t)
	cmd := DshbotRoomEditCommand{Expected: status, Values: DshbotRoomEditValues{MemberBotIDs: []string{"b", "c"}, MaxRounds: 2, MaxSpeaks: 4}}
	calls := 0
	c := Client{BaseURL: "https://fixture.invalid", HTTPClient: &http.Client{Transport: roomEditTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		var req struct {
			RPCID   string                     `json:"rpcId"`
			Method  string                     `json:"method"`
			Payload map[string]json.RawMessage `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req.Method != "bot/update" || r.URL.Path != "/dshbot/bot/update" {
			t.Fatal("wrong endpoint", req.Method)
		}
		want := DshbotRoomEditProfile(cmd)
		want["revision"] = json.RawMessage("7")
		if !reflect.DeepEqual(want, req.Payload) {
			t.Fatal("not exact reviewed payload")
		}
		if req.Payload["requestId"] != nil {
			t.Fatal("invented idempotency")
		}
		p := DshbotRoomEditProfile(cmd)
		p["updatedAt"] = json.RawMessage("3")
		receiptItem := CloneDshbotRoomEdit(cmd.Expected).Profile
		for k, v := range p {
			receiptItem[k] = v
		}
		receiptItem["avatar"] = json.RawMessage(`{"kind":"blob","shape":"circle","color":"blue"}`) // next precedes schema defaults
		body, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": req.RPCID, "result": map[string]any{"ok": true, "value": map[string]any{"view": roomEditFixtureView(p, 8), "item": receiptItem, "sessionId": "canonical-room", "warnings": []string{"title rename warning"}}}})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(body)))}, nil
	})}}
	result, err := c.DshbotSaveRoomEdit(context.Background(), cmd)
	if err != nil || calls != 1 || !result.Attempted || len(result.Warnings) != 1 {
		t.Fatal(result, err, calls)
	}
	if !sameCapabilityJSON(result.Status.Profile["capabilities"], status.Profile["capabilities"]) || !sameCapabilityJSON(result.Status.Profile["futureTop"], status.Profile["futureTop"]) {
		t.Fatal("opaque fields lost")
	}
	c.HTTPClient.Transport = roomEditTransport(func(*http.Request) (*http.Response, error) { calls++; return nil, fmt.Errorf("response lost") })
	result, err = c.DshbotSaveRoomEdit(context.Background(), cmd)
	if err == nil || !result.Attempted || calls != 2 {
		t.Fatal("unknown retried", result, err, calls)
	}
	for _, bad := range []struct{ key, raw string }{{"avatar", `{"kind":"blob","shape":"circle","color":"blue","crop":"","dataUrl":"","future":true}`}, {"model", `{"provider":"","model":"","reasoningEffort":"","future":true}`}, {"notify", `false`}, {"workspaceId", `" workspace-1 "`}, {"inbox", `[{}]`}, {"holdCheckpoint", `{}`}} {
		b := CloneDshbotRoomEditCommand(cmd)
		b.Expected.Profile[bad.key] = json.RawMessage(bad.raw)
		raw, _ := json.Marshal(b.Expected.Profile)
		json.Unmarshal(raw, &b.Expected.Item)
		if ValidateDshbotRoomEdit(b) == nil {
			t.Fatal("unsafe normalization accepted", bad.key)
		}
		out, err := c.DshbotSaveRoomEdit(context.Background(), b)
		if err == nil || out.Attempted || calls != 2 {
			t.Fatal("unsafe update dispatched", bad.key)
		}
	}
	for _, members := range [][]string{{"a"}, {"a", "a"}, {"a", "room"}, {"a", "missing"}} {
		b := CloneDshbotRoomEditCommand(cmd)
		b.Values.MemberBotIDs = members
		if ValidateDshbotRoomEdit(b) == nil {
			t.Fatal("invalid member accepted", members)
		}
	}
	for _, limits := range [][2]int{{0, 1}, {4, 1}, {1, 0}, {1, 11}} {
		b := CloneDshbotRoomEditCommand(cmd)
		b.Values.MaxRounds, b.Values.MaxSpeaks = limits[0], limits[1]
		if ValidateDshbotRoomEdit(b) == nil {
			t.Fatal("invalid limits", limits)
		}
	}
	fresh := CloneDshbotRoomEdit(status)
	fresh.Catalog.Revision++
	if DshbotRoomEditApplied(cmd, fresh) == nil {
		t.Fatal("different readback unlocked")
	}
	fresh.Profile = DshbotRoomEditProfile(cmd)
	raw, _ := json.Marshal(fresh.Profile)
	json.Unmarshal(raw, &fresh.Item)
	if err := DshbotRoomEditApplied(cmd, fresh); err != nil {
		t.Fatal(err)
	}
	fresh.Profile["futureTop"] = json.RawMessage(`{}`)
	if DshbotRoomEditApplied(cmd, fresh) == nil {
		t.Fatal("unknown top-level loss accepted")
	}
}
