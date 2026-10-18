package pluginrpc

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type visibilityTransport func(*http.Request) (*http.Response, error)

func (f visibilityTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func visibilityFixture(t *testing.T) DshbotBotCatalog {
	t.Helper()
	v, err := decodeDshbotBotCatalog(json.RawMessage(`{"ns":"dsh-bot","revision":8,"value":{"items":[{"id":"bot-a","kind":"bot","sessionId":"s-a","name":"Alpha","title":"title","hidden":false,"updatedAt":2,"avatar":{"kind":"blob","color":"blue","shape":"circle","dataUrl":"","crop":""},"future":{"nested":[9007199254740993,"保持"]},"notify":true,"inbox":[{"keep":true}]},{"id":"room-b","kind":"room","sessionId":"s-b","name":"Room","hidden":true,"updatedAt":3,"memberBotIds":["bot-a","bot-c"],"holds":{"bot-a":{"keep":true}}}]}}`))
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func TestBotVisibilityCatalogProtocolReceipts(t *testing.T) {
	modes := []string{"hide-bot", "unhide-bot", "hide-room", "unhide-room", "transport", "stale", "reorder", "dropped", "other-field", "target-session", "target-extra", "timestamp", "null-hidden", "readonly"}
	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			base := visibilityFixture(t)
			idx := 0
			if strings.HasSuffix(mode, "room") {
				idx = 1
			}
			before := strings.HasPrefix(mode, "unhide")
			base.Items[idx].Hidden = before
			base.Items[idx].Profile["hidden"], _ = json.Marshal(before)
			cmd := DshbotVisibilityCommand{Expected: base, ID: base.Items[idx].ID, SessionID: base.Items[idx].SessionID, Hidden: !before, UpdatedAt: 100}
			calls := 0
			client := Client{BaseURL: "https://fixture.invalid", HTTPClient: &http.Client{Transport: visibilityTransport(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.URL.Path != "/dshbot/catalog/items" {
					t.Fatal(req.URL.Path)
				}
				var request clientRequest
				decoder := json.NewDecoder(req.Body)
				decoder.UseNumber()
				decoder.Decode(&request)
				raw, _ := json.Marshal(request.Payload)
				var payload struct {
					Revision int                          `json:"revision"`
					Items    []map[string]json.RawMessage `json:"items"`
				}
				json.Unmarshal(raw, &payload)
				var keys map[string]json.RawMessage
				json.Unmarshal(raw, &keys)
				if payload.Revision != 8 || len(keys) != 2 || len(payload.Items) != 2 {
					t.Fatal("not full CAS items")
				}
				for i, row := range base.Items {
					for k, v := range row.Profile {
						if i == idx && (k == "hidden" || k == "updatedAt") {
							continue
						}
						if !sameCapabilityJSON(v, payload.Items[i][k]) {
							t.Fatal("unrelated payload difference", i, k)
						}
					}
				}
				if !sameCapabilityJSON(payload.Items[idx]["hidden"], json.RawMessage(map[bool]string{true: "true", false: "false"}[cmd.Hidden])) || string(payload.Items[idx]["updatedAt"]) != "100" {
					t.Fatal("wrong mutation")
				}
				if mode == "transport" {
					return nil, errors.New("lost reply")
				}
				if mode == "readonly" {
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"result":{"ok":false,"error":{"code":"blocked","message":"Bot catalog is read-only."}}}`))}, nil
				}
				revision := 9
				switch mode {
				case "stale":
					revision = 8
				case "reorder":
					payload.Items[0], payload.Items[1] = payload.Items[1], payload.Items[0]
				case "dropped":
					payload.Items = payload.Items[:1]
				case "other-field":
					payload.Items[1]["holds"] = json.RawMessage(`{}`)
				case "target-session":
					payload.Items[idx]["sessionId"] = json.RawMessage(`"other"`)
				case "target-extra":
					payload.Items[idx]["added"] = json.RawMessage(`true`)
				case "timestamp":
					payload.Items[idx]["updatedAt"] = json.RawMessage(`101`)
				case "null-hidden":
					payload.Items[idx]["hidden"] = json.RawMessage(`null`)
				}
				reply, _ := json.Marshal(map[string]any{"view": map[string]any{"ns": "dsh-bot", "revision": revision, "value": map[string]any{"items": payload.Items}}})
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"result":{"ok":true,"value":` + string(reply) + `}}`))}, nil
			})}}
			out, err := client.DshbotSetVisibility(context.Background(), cmd)
			success := strings.HasPrefix(mode, "hide-") || strings.HasPrefix(mode, "unhide-")
			if (err == nil) != success || calls != 1 || !out.Attempted || out.Rejected != (mode == "readonly") {
				t.Fatal(mode, err, out.Rejected, calls)
			}
			if success && (out.Item.ID != cmd.ID || out.Item.SessionID != cmd.SessionID || out.Item.Hidden != cmd.Hidden) {
				t.Fatal("identity/visibility")
			}
			if !sameCapabilityJSON(base.Items[0].Profile["future"], json.RawMessage(`{"nested":[9007199254740993,"保持"]}`)) {
				t.Fatal("source mutated")
			}
		})
	}
	calls := 0
	client := Client{BaseURL: "https://fixture.invalid", HTTPClient: &http.Client{Transport: visibilityTransport(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("must not send") })}}
	for _, mode := range []string{"missing", "room-session", "kind", "null", "string", "duplicate", "unchanged", "stamp"} {
		base := visibilityFixture(t)
		cmd := DshbotVisibilityCommand{Expected: base, ID: "bot-a", SessionID: "s-a", Hidden: true, UpdatedAt: 100}
		switch mode {
		case "missing":
			delete(base.Items[0].Profile, "hidden")
		case "room-session":
			cmd.SessionID = "s-b"
		case "kind":
			base.Items[0].Profile["kind"] = json.RawMessage(`"future"`)
		case "null":
			base.Items[0].Profile["hidden"] = json.RawMessage(`null`)
		case "string":
			base.Items[0].Profile["hidden"] = json.RawMessage(`"false"`)
		case "duplicate":
			base.Items = append(base.Items, base.Items[0])
			cmd.Expected = base
		case "unchanged":
			cmd.Hidden = false
		case "stamp":
			cmd.UpdatedAt = 0
		}
		if out, err := client.DshbotSetVisibility(context.Background(), cmd); err == nil || out.Attempted {
			t.Fatal("invalid command sent", mode)
		}
	}
	if calls != 0 {
		t.Fatal("preflight bypass")
	}
}
