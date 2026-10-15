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

type objectiveConfigTransport func(*http.Request) (*http.Response, error)

func (f objectiveConfigTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func objectiveConfigFixture() DshbotCapabilityStatus {
	var p map[string]json.RawMessage
	json.Unmarshal([]byte(`{"id":"b","kind":"bot","sessionId":"s","name":"Fixture","title":"title","sectionId":"","description":"Keep description","avatar":{"kind":"blob","shape":"circle","color":"blue","dataUrl":"","crop":""},"workspaceId":"/fixture","model":{"provider":"","model":"","reasoningEffort":""},"pinned":false,"hidden":false,"pinOrder":0,"allowedSenderIds":[],"memberBotIds":[],"objective":"Existing objective","objectiveMaxRounds":16,"memoryReview":false,"notify":false,"capabilities":{"tools":{"mode":"all","names":[]},"skills":{"mode":"selected","names":[]},"mcp":{"mode":"selected","names":[]}},"maxRounds":3,"maxSpeaks":10,"inbox":[],"readInitialized":true,"lastSeenSeq":0,"createdAt":1,"updatedAt":2,"futureField":{"keep":true}}`), &p)
	var policy DshbotCapabilityPolicies
	json.Unmarshal(p["capabilities"], &policy)
	raw, _ := json.Marshal(p)
	var bot DshbotCatalogItem
	json.Unmarshal(raw, &bot)
	return DshbotCapabilityStatus{Bot: bot, Revision: 8, Profile: p, Policies: policy, Catalog: DshbotCapabilityCatalog{Active: true, SkillsAvailable: true, Tools: []DshbotCapabilityEntry{{Name: "read"}}, Skills: []DshbotCapabilityEntry{}, MCP: []DshbotCapabilityEntry{}}}
}

func TestBotObjectiveConfigExactPairAndPreservation(t *testing.T) {
	for _, initial := range []string{"", "Existing objective"} {
		t.Run("from-"+initial, func(t *testing.T) {
			base := objectiveConfigFixture()
			base.Profile["objective"], _ = json.Marshal(initial)
			calls := 0
			c := Client{BaseURL: "https://fixture.invalid", HTTPClient: &http.Client{Transport: objectiveConfigTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				var req clientRequest
				json.NewDecoder(r.Body).Decode(&req)
				if req.Method != "bot/update" || r.URL.Path != "/dshbot/bot/update" {
					t.Fatal(req.Method, r.URL.Path)
				}
				encoded, _ := json.Marshal(req.Payload)
				var p map[string]json.RawMessage
				json.Unmarshal(encoded, &p)
				if string(p["revision"]) != "8" || string(p["objective"]) != `"New complete objective"` || string(p["objectiveMaxRounds"]) != "200" {
					t.Fatal("wrong pair/CAS", p)
				}
				if len(p) != len(base.Profile)+1 {
					t.Fatal("missing full profile")
				}
				for k, v := range base.Profile {
					if k != "objective" && k != "objectiveMaxRounds" && !sameCapabilityJSON(v, p[k]) {
						t.Fatal("changed unrelated", k)
					}
				}
				delete(p, "revision")
				p["updatedAt"] = json.RawMessage(`3`)
				view := map[string]any{"ns": "dsh-bot", "revision": 9, "value": map[string]any{"items": []any{p}}}
				value, _ := json.Marshal(map[string]any{"view": view, "warnings": []string{"fixture sync warning"}})
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"result":{"ok":true,"value":` + string(value) + `}}`))}, nil
			})}}
			out, err := c.DshbotSaveObjectiveConfig(context.Background(), DshbotObjectiveConfigCommand{Expected: base, Desired: DshbotObjectiveConfig{Objective: "New complete objective", MaxRounds: 200}})
			if err != nil || !out.Attempted || calls != 1 || len(out.Warnings) != 1 {
				t.Fatal(err, out, calls)
			}
			if string(base.Profile["objectiveMaxRounds"]) != "16" {
				t.Fatal("mutated reviewed snapshot")
			}
		})
	}
}
func TestBotObjectiveConfigLimitsAndLosslessBoundary(t *testing.T) {
	base := objectiveConfigFixture()
	for _, draft := range []DshbotObjectiveConfigDraft{{Objective: "", MaxRounds: "16"}, {Objective: "x", MaxRounds: "0"}, {Objective: "x", MaxRounds: "201"}, {Objective: "x", MaxRounds: "1.5"}, {Objective: strings.Repeat("😀", 1001), MaxRounds: "16"}, {Objective: string([]byte{0xff}), MaxRounds: "16"}} {
		if _, err := ParseDshbotObjectiveConfigDraft(draft); err == nil {
			t.Fatal("invalid accepted", draft)
		}
	}
	desired, err := ParseDshbotObjectiveConfigDraft(DshbotObjectiveConfigDraft{Objective: "\ufeff " + strings.Repeat("😀", 1000) + " \u00a0", MaxRounds: " 200 "})
	if err != nil || WhaleTextLength(desired.Objective) != 2000 || desired.MaxRounds != 200 {
		t.Fatal(err)
	}
	if ValidateDshbotObjectiveConfigCommand(DshbotObjectiveConfigCommand{Expected: base, Desired: DshbotObjectiveConfig{Objective: "Existing objective", MaxRounds: 16}}) == nil {
		t.Fatal("unchanged accepted")
	}
	for _, field := range []string{"objective", "objectiveMaxRounds"} {
		copy := CloneDshbotCapabilities(base)
		delete(copy.Profile, field)
		if _, err := DshbotObjectiveConfigValue(copy); err == nil {
			t.Fatal("missing defaulted", field)
		}
	}
	copy := CloneDshbotCapabilities(base)
	copy.Profile["model"] = json.RawMessage(`{"provider":"","model":"","reasoningEffort":"","future":true}`)
	if ValidateDshbotObjectiveConfigCommand(DshbotObjectiveConfigCommand{Expected: copy, Desired: desired}) == nil {
		t.Fatal("unknown nested dropped")
	}
	c := Client{BaseURL: "https://fixture.invalid", HTTPClient: &http.Client{Transport: objectiveConfigTransport(func(*http.Request) (*http.Response, error) { t.Fatal("invalid patch sent"); return nil, nil })}}
	for _, fields := range []map[string]json.RawMessage{{"objective": json.RawMessage(`"x"`)}, {"objective": json.RawMessage(`"x"`), "objectiveMaxRounds": json.RawMessage(`16`), "notify": json.RawMessage(`true`)}, {"notify": json.RawMessage(`true`)}} {
		if out, err := c.saveDshbotProfileFields(context.Background(), base, fields); err == nil || out.Attempted {
			t.Fatal("unsupported patch")
		}
	}
}
func TestBotObjectiveConfigUnknownAndMismatchedReceipt(t *testing.T) {
	for _, mode := range []string{"transport", "cap", "unrelated", "unsafeReceipt"} {
		t.Run(mode, func(t *testing.T) {
			base := objectiveConfigFixture()
			if mode == "unsafeReceipt" {
				base.Profile["allowedSenderIds"] = json.RawMessage(`["peer"]`)
				base.PeerIDs = []string{"peer"}
			}
			calls := 0
			c := Client{BaseURL: "https://fixture.invalid", HTTPClient: &http.Client{Transport: objectiveConfigTransport(func(*http.Request) (*http.Response, error) {
				calls++
				if mode == "transport" {
					return nil, errors.New("synthetic uncertain")
				}
				p := CloneDshbotCapabilities(base).Profile
				p["objective"] = json.RawMessage(`"Changed"`)
				p["objectiveMaxRounds"] = json.RawMessage(`20`)
				if mode == "cap" {
					p["objectiveMaxRounds"] = json.RawMessage(`21`)
				} else if mode == "unrelated" {
					p["notify"] = json.RawMessage(`true`)
				}
				value, _ := json.Marshal(map[string]any{"view": map[string]any{"ns": "dsh-bot", "revision": 9, "value": map[string]any{"items": []any{p}}}})
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"result":{"ok":true,"value":` + string(value) + `}}`))}, nil
			})}}
			out, err := c.DshbotSaveObjectiveConfig(context.Background(), DshbotObjectiveConfigCommand{Expected: base, Desired: DshbotObjectiveConfig{Objective: "Changed", MaxRounds: 20}})
			if err == nil || !out.Attempted || calls != 1 {
				t.Fatal("uncertain retried/accepted", err, calls)
			}
		})
	}
}
