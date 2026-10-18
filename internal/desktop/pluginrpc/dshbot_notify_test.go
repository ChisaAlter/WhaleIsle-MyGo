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

type notifyTransport func(*http.Request) (*http.Response, error)

func (f notifyTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func notifyFixture() DshbotCapabilityStatus {
	var p map[string]json.RawMessage
	json.Unmarshal([]byte(`{"id":"b","kind":"bot","sessionId":"s","name":"Fixture","title":"title","sectionId":"","description":"Keep description","avatar":{"kind":"blob","shape":"circle","color":"blue","dataUrl":"","crop":""},"workspaceId":"/fixture","model":{"provider":"","model":"","reasoningEffort":""},"pinned":false,"hidden":false,"pinOrder":0,"allowedSenderIds":[],"memberBotIds":[],"objective":"Existing objective","objectiveMaxRounds":16,"memoryReview":false,"notify":false,"capabilities":{"tools":{"mode":"all","names":[]},"skills":{"mode":"selected","names":[]},"mcp":{"mode":"selected","names":[]}},"maxRounds":3,"maxSpeaks":10,"inbox":[],"readInitialized":true,"lastSeenSeq":0,"createdAt":1,"updatedAt":2,"futureField":{"keep":true}}`), &p)
	var policy DshbotCapabilityPolicies
	json.Unmarshal(p["capabilities"], &policy)
	raw, _ := json.Marshal(p)
	var bot DshbotCatalogItem
	json.Unmarshal(raw, &bot)
	return DshbotCapabilityStatus{Bot: bot, Revision: 8, Profile: p, Policies: policy, Catalog: DshbotCapabilityCatalog{Active: true, SkillsAvailable: true, Tools: []DshbotCapabilityEntry{{Name: "read"}}, Skills: []DshbotCapabilityEntry{}, MCP: []DshbotCapabilityEntry{}}}
}
func TestBotNotifyExactBooleanAndCASReceipts(t *testing.T) {
	expected := notifyFixture()
	for _, raw := range []string{"", `null`, `"true"`, `0`, `{}`, `[]`} {
		bad := CloneDshbotCapabilities(expected)
		bad.Profile["notify"] = json.RawMessage(raw)
		if _, err := DshbotNotifyEnabled(bad); err == nil {
			t.Fatal("guessed boolean", raw)
		}
	}
	if ValidateDshbotNotifyCommand(DshbotNotifyCommand{Expected: expected, Enabled: false}) == nil {
		t.Fatal("unchanged save")
	}
	for _, mode := range []string{"enable", "disable", "transport", "item", "session", "unknown-avatar", "unrelated", "missing", "extra"} {
		t.Run(mode, func(t *testing.T) {
			before := notifyFixture()
			desired := mode != "disable"
			if !desired {
				before.Profile["notify"] = json.RawMessage(`true`)
			}
			cmd := DshbotNotifyCommand{Expected: before, Enabled: desired}
			calls := 0
			client := Client{BaseURL: "http://127.0.0.1", HTTPClient: &http.Client{Transport: notifyTransport(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.URL.Path != "/dshbot/bot/update" {
					t.Fatal("unexpected request")
				}
				var request clientRequest
				json.NewDecoder(req.Body).Decode(&request)
				b, _ := json.Marshal(request.Payload)
				var profile map[string]json.RawMessage
				json.Unmarshal(b, &profile)
				if string(profile["revision"]) != "8" || len(profile) != len(before.Profile)+1 {
					t.Fatal("CAS/full profile")
				}
				for k, v := range before.Profile {
					if k != "notify" && !sameCapabilityJSON(v, profile[k]) {
						t.Fatal("unrelated write", k)
					}
				}
				got, err := DshbotNotifyEnabled(DshbotCapabilityStatus{Profile: profile})
				if err != nil || got != desired {
					t.Fatal("wrong notify")
				}
				if mode == "transport" {
					return nil, errors.New("unknown")
				}
				delete(profile, "revision")
				profile["updatedAt"] = json.RawMessage(`4`)
				item := map[string]json.RawMessage{}
				for k, v := range profile {
					item[k] = v
				}
				item["avatar"] = json.RawMessage(`{"kind":"blob","shape":"circle","color":"blue"}`)
				if mode == "item" {
					item["notify"] = json.RawMessage(`false`)
				}
				if mode == "unknown-avatar" {
					item["avatar"] = json.RawMessage(`{"kind":"blob","shape":"circle","color":"blue","future":true}`)
				}
				if mode == "unrelated" {
					profile["hidden"] = json.RawMessage(`true`)
				}
				if mode == "missing" {
					delete(profile, "description")
				}
				if mode == "extra" {
					profile["extra"] = json.RawMessage(`true`)
				}
				session := "s"
				if mode == "session" {
					session = "other"
				}
				response, _ := json.Marshal(map[string]any{"view": map[string]any{"ns": "dsh-bot", "revision": 9, "value": map[string]any{"items": []any{profile}}}, "item": item, "sessionId": session, "warnings": []string{"presentation warning"}})
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"result":{"ok":true,"value":` + string(response) + `}}`))}, nil
			})}}
			out, err := client.DshbotSaveNotify(context.Background(), cmd)
			if (err == nil) != (mode == "enable" || mode == "disable") || !out.Attempted || calls != 1 {
				t.Fatal(mode, err, calls)
			}
			if err == nil && len(out.Warnings) != 1 {
				t.Fatal("warning lost")
			}
		})
	}
	calls := 0
	client := Client{BaseURL: "http://127.0.0.1", HTTPClient: &http.Client{Transport: notifyTransport(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("must not send") })}}
	for _, fields := range []map[string]json.RawMessage{{"notify": json.RawMessage(`null`)}, {"notify": json.RawMessage(`"true"`)}, {"notify": json.RawMessage(`true`), "hidden": json.RawMessage(`true`)}, {"hidden": json.RawMessage(`true`)}} {
		out, err := client.saveDshbotProfileFields(context.Background(), expected, fields)
		if err == nil || out.Attempted {
			t.Fatal("unreviewed field")
		}
	}
	if calls != 0 {
		t.Fatal("rejected payload sent")
	}
}
