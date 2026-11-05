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

type botDuplicateTransport func(*http.Request) (*http.Response, error)

func (f botDuplicateTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	response, err := f(r)
	return fixtureResponseIdentity(r, response, err)
}
func botDuplicateFixture() DshbotBotDuplicateCommand {
	raw := json.RawMessage(`{"id":"source","kind":"bot","sessionId":"source-session","name":"Source","title":"Display title","sectionId":"","description":"Full preserved description","avatar":{"kind":"blob","shape":"circle","color":"blue"},"workspaceId":"workspace-fixture","model":{"provider":"provider","model":"model","reasoningEffort":"high"},"pinned":true,"hidden":false,"pinOrder":7,"allowedSenderIds":[],"memberBotIds":[],"objective":"Complete this entire objective","objectiveMaxRounds":23,"memoryReview":false,"notify":true,"capabilities":{"tools":{"mode":"selected","names":["read"]},"skills":{"mode":"all","names":[]},"mcp":{"mode":"selected","names":[]}},"maxRounds":2,"maxSpeaks":7,"inbox":[{"id":"old-message"}],"readInitialized":true,"lastSeenSeq":8,"createdAt":1,"updatedAt":2}`)
	item, _ := decodeDshbotBotItem(raw)
	return DshbotBotDuplicateCommand{SourceID: "source", Name: "Chosen copy name", ScratchCwd: "/reviewed/cwd", Expected: DshbotBotCatalog{Revision: 7, Items: []DshbotBotItem{item}}, AllowObjectiveExecution: true}
}
func botDuplicateResultItem(cmd DshbotBotDuplicateCommand, id, sid string) DshbotBotItem {
	p := map[string]json.RawMessage{}
	for _, key := range DshbotBotDuplicateConfigKeys {
		p[key] = cmd.Expected.Items[0].Profile[key]
	}
	for key, v := range map[string]any{"id": id, "sessionId": sid, "kind": "bot", "name": cmd.Name, "hidden": false, "pinned": false, "pinOrder": 0, "inbox": []any{}, "memberBotIds": []string{}, "readInitialized": true, "lastSeenSeq": 0, "createdAt": 3, "updatedAt": 3, "maxRounds": 3, "maxSpeaks": 10} {
		p[key], _ = json.Marshal(v)
	}
	raw, _ := json.Marshal(p)
	item, _ := decodeDshbotBotItem(raw)
	return item
}
func TestBotDuplicateContractWarningsAndUnknownCandidates(t *testing.T) {
	cmd := botDuplicateFixture()
	calls := 0
	lost := false
	badReceipt := false
	client := &Client{BaseURL: "http://127.0.0.1", HTTPClient: &http.Client{Transport: botDuplicateTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		var req struct {
			RPCID, Method string
			Payload       map[string]json.RawMessage
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req.Method != "bot/duplicate" || r.URL.Path != "/dshbot/bot/duplicate" || len(req.Payload) != 4 {
			t.Fatal("unexpected action/payload", req.Method, req.Payload)
		}
		for k, want := range map[string]string{"id": `"source"`, "name": `"Chosen copy name"`, "revision": "7", "scratchCwd": `"/reviewed/cwd"`} {
			if string(req.Payload[k]) != want {
				t.Fatal(k, string(req.Payload[k]))
			}
		}
		if lost {
			return nil, fmt.Errorf("response lost")
		}
		item := botDuplicateResultItem(cmd, "server-copy", "server-session")
		sid := item.SessionID
		if badReceipt {
			sid = "wrong"
		}
		value := map[string]any{"view": map[string]any{"ns": "dsh-bot", "revision": 8, "value": map[string]any{"items": []any{cmd.Expected.Items[0].Profile, item.Profile}}}, "item": item.Profile, "sessionId": sid, "warnings": []string{"Bot was duplicated, but memory copy failed: fixture warning in full"}}
		b, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": req.RPCID, "result": map[string]any{"ok": true, "value": value}})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(b)))}, nil
	})}}
	unconfirmed := cmd
	unconfirmed.AllowObjectiveExecution = false
	out, err := client.DshbotDuplicateBot(context.Background(), unconfirmed)
	if err == nil || out.Attempted || calls != 0 {
		t.Fatal("objective execution unconfirmed")
	}
	out, err = client.DshbotDuplicateBot(context.Background(), cmd)
	if err != nil || out.Item.ID != "server-copy" || len(out.Warnings) != 1 || !strings.HasSuffix(out.Warnings[0], "in full") || calls != 1 {
		t.Fatal(out, err, calls)
	}
	lost = true
	out, err = client.DshbotDuplicateBot(context.Background(), cmd)
	if err == nil || out.Rejected || !out.Attempted || calls != 2 {
		t.Fatal("unknown repeated/unlocked", out, err)
	}
	lost = false
	badReceipt = true
	out, err = client.DshbotDuplicateBot(context.Background(), cmd)
	if err == nil || out.Rejected || calls != 3 {
		t.Fatal("wrong receipt accepted")
	}
	view := CloneDshbotBotCatalog(cmd.Expected)
	view.Revision++
	view.Items = append(view.Items, botDuplicateResultItem(cmd, "candidate-a", "session-a"), botDuplicateResultItem(cmd, "candidate-b", "session-b"))
	candidates := DshbotBotDuplicateCandidates(cmd, view)
	if len(candidates) != 2 {
		t.Fatal("candidates incorrectly treated as one causal record", len(candidates))
	}
	beforeOpen := CloneDshbotBotItem(view.Items[1])
	view.Items[1].Profile["objective"] = json.RawMessage(`"changed"`)
	if SameDshbotBotDuplicateOpenTarget(beforeOpen, view.Items[1]) {
		t.Fatal("changed objective opened")
	}
	if len(DshbotBotDuplicateCandidates(cmd, view)) != 1 {
		t.Fatal("configuration drift accepted")
	}
	bad := cmd
	bad.Expected = CloneDshbotBotCatalog(cmd.Expected)
	bad.Expected.Items[0].Profile["futureConfig"] = json.RawMessage(`{"unknown":true}`)
	if ValidateDshbotBotDuplicate(bad) == nil {
		t.Fatal("unknown source configuration silently dropped")
	}
}
