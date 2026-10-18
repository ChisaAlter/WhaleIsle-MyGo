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

type botCreateTransport func(*http.Request) (*http.Response, error)

func (f botCreateTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func botCreateFixtureCommand() DshbotBotCreateCommand {
	return DshbotBotCreateCommand{ID: "bot-fixture", SessionID: "session-bot-fixture", Expected: DshbotBotCatalog{Revision: 7, Items: []DshbotBotItem{}}, Draft: DshbotBotDraft{Name: "Fixture", Title: "Title", Description: "Description", ScratchCwd: "/fixture", Provider: "provider", Model: "model", ReasoningEffort: "high", MemoryReview: true, Notify: true}}
}
func botCreateFixtureItem(cmd DshbotBotCreateCommand) DshbotBotItem {
	p := dshbotBotCreateProfile(cmd)
	raw, _ := json.Marshal(p)
	var item DshbotCatalogItem
	json.Unmarshal(raw, &item)
	return DshbotBotItem{DshbotCatalogItem: item, Profile: p}
}
func botCreateFixtureView(cmd DshbotBotCreateCommand) DshbotBotCatalog {
	return DshbotBotCatalog{Revision: 8, Items: []DshbotBotItem{botCreateFixtureItem(cmd)}}
}
func TestBotCreateContractBoundary(t *testing.T) {
	cmd := botCreateFixtureCommand()
	calls := 0
	responseError := ""
	lost := false
	inconsistent := false
	client := &Client{BaseURL: "https://fixture.invalid", HTTPClient: &http.Client{Transport: botCreateTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		var req struct {
			RPCID, Method string
			Payload       map[string]json.RawMessage
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req.Method != "bot/create" || r.URL.Path != "/dshbot/bot/create" {
			t.Fatal("unexpected mutation", req.Method)
		}
		for k, v := range map[string]string{"kind": `"bot"`, "id": `"bot-fixture"`, "sessionId": `"session-bot-fixture"`, "revision": "7", "title": `"Title"`, "objective": `""`, "objectiveMaxRounds": "16", "memoryReview": "true", "notify": "true", "allowedSenderIds": "[]"} {
			if string(req.Payload[k]) != v {
				t.Fatalf("%s=%s", k, req.Payload[k])
			}
		}
		for _, k := range []string{"requestId", "prompt", "agentPreset", "memberBotIds", "maxRounds", "maxSpeaks"} {
			if _, ok := req.Payload[k]; ok {
				t.Fatal("invented or room-only payload", k)
			}
		}
		if lost {
			return nil, fmt.Errorf("response lost")
		}
		result := map[string]any{"ok": false, "error": map[string]string{"code": "error", "message": responseError}}
		if responseError == "" {
			p := dshbotBotCreateProfile(cmd)
			sid := cmd.SessionID
			if inconsistent {
				sid = "unrelated"
			}
			result = map[string]any{"ok": true, "value": map[string]any{"view": map[string]any{"ns": "dsh-bot", "revision": 8, "value": map[string]any{"items": []any{p}}}, "item": p, "sessionId": sid}}
		}
		body, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": req.RPCID, "result": result})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(body)))}, nil
	})}}
	out, err := client.DshbotCreateBot(context.Background(), cmd)
	if err != nil || out.Item.ID != cmd.ID || calls != 1 {
		t.Fatal(out, err, calls)
	}
	lost = true
	out, err = client.DshbotCreateBot(context.Background(), cmd)
	if err == nil || out.Rejected || !out.Attempted || calls != 2 {
		t.Fatal("lost response retried or unlocked", out, err, calls)
	}
	lost = false
	inconsistent = true
	out, err = client.DshbotCreateBot(context.Background(), cmd)
	if err == nil || out.Rejected || calls != 3 {
		t.Fatal("wrong canonical receipt accepted", out, err, calls)
	}
	inconsistent = false
	for _, tc := range []struct {
		message  string
		rejected bool
	}{{"Bot catalog changed. Refresh and try again.", true}, {"No application default model is configured for this Bot. Select a Bot model first.", true}, {"Bot Session creation failed and catalog rollback also failed: Bot catalog changed. Refresh and try again.", false}, {"Session failed: Model provider and model must be selected together.", false}} {
		responseError = tc.message
		out, err = client.DshbotCreateBot(context.Background(), cmd)
		if err == nil || out.Rejected != tc.rejected {
			t.Fatal("unsafe prewrite classification", tc, out, err)
		}
	}
	before := calls
	bad := cmd
	bad.Draft.Model = ""
	out, err = client.DshbotCreateBot(context.Background(), bad)
	if err == nil || out.Attempted || !out.Rejected || calls != before {
		t.Fatal("invalid model dispatched")
	}
	for _, raw := range []string{`{"ns":"dsh-bot","value":{"items":[]}}`, `{"ns":"dsh-bot","revision":1,"value":{}}`, `{"ns":"wrong","revision":1,"value":{"items":[]}}`} {
		if _, err := decodeDshbotBotCatalog(json.RawMessage(raw)); err == nil {
			t.Fatal("guessed catalog", raw)
		}
	}
	good := botCreateFixtureView(cmd)
	if _, err := DshbotCreatedBot(cmd, good); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"title", "description", "model", "avatar", "capabilities", "allowedSenderIds", "objective", "objectiveMaxRounds", "memoryReview", "notify", "sectionId"} {
		v := CloneDshbotBotCatalog(good)
		delete(v.Items[0].Profile, key)
		if _, err := DshbotCreatedBot(cmd, v); err == nil {
			t.Fatal("incomplete record adopted", key)
		}
	}
	v := CloneDshbotBotCatalog(good)
	v.Items[0].Profile["model"] = json.RawMessage(`{"provider":"other","model":"model","reasoningEffort":"high"}`)
	if _, err := DshbotCreatedBot(cmd, v); err == nil {
		t.Fatal("wrong model adopted")
	}
	if SameDshbotBotCatalog(v, good) {
		t.Fatal("profile drift ignored")
	}
}
