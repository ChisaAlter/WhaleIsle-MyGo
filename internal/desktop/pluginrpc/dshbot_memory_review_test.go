package pluginrpc

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

type memoryReviewTransport func(*http.Request) (*http.Response, error)

func (f memoryReviewTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func memoryReviewFixture() DshbotCapabilityStatus {
	var p map[string]json.RawMessage
	json.Unmarshal([]byte(`{"id":"b","kind":"bot","sessionId":"s","name":"Fixture","title":"title","sectionId":"","description":"Keep description","avatar":{"kind":"blob","shape":"circle","color":"blue","dataUrl":"","crop":""},"workspaceId":"/fixture","model":{"provider":"","model":"","reasoningEffort":""},"pinned":false,"hidden":false,"pinOrder":0,"allowedSenderIds":[],"memberBotIds":[],"objective":"Existing objective","objectiveMaxRounds":16,"memoryReview":false,"notify":false,"capabilities":{"tools":{"mode":"all","names":[]},"skills":{"mode":"selected","names":[]},"mcp":{"mode":"selected","names":[]}},"maxRounds":3,"maxSpeaks":10,"inbox":[],"readInitialized":true,"lastSeenSeq":0,"createdAt":1,"updatedAt":2,"futureField":{"keep":true}}`), &p)
	var policy DshbotCapabilityPolicies
	json.Unmarshal(p["capabilities"], &policy)
	raw, _ := json.Marshal(p)
	var bot DshbotCatalogItem
	json.Unmarshal(raw, &bot)
	return DshbotCapabilityStatus{Bot: bot, Revision: 8, Profile: p, Policies: policy, Catalog: DshbotCapabilityCatalog{Active: true, SkillsAvailable: true, Tools: []DshbotCapabilityEntry{{Name: "read"}}, Skills: []DshbotCapabilityEntry{}, MCP: []DshbotCapabilityEntry{}}}
}
func TestBotMemoryReviewSharedProfileWriterExactFields(t *testing.T) {
	for _, field := range []string{"memoryReview", "capabilities"} {
		t.Run(field, func(t *testing.T) {
			base := memoryReviewFixture()
			calls := 0
			c := Client{BaseURL: "http://127.0.0.1", HTTPClient: &http.Client{Transport: memoryReviewTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				var req clientRequest
				json.NewDecoder(r.Body).Decode(&req)
				if req.Method != "bot/update" || r.URL.Path != "/dshbot/bot/update" {
					t.Fatal(req.Method, r.URL.Path)
				}
				encoded, _ := json.Marshal(req.Payload)
				var p map[string]json.RawMessage
				json.Unmarshal(encoded, &p)
				if string(p["revision"]) != "8" {
					t.Fatal("missing CAS")
				}
				delete(p, "revision")
				for key, value := range base.Profile {
					if key != field && !sameCapabilityJSON(value, p[key]) {
						t.Fatal("unrelated field changed", key)
					}
				}
				if field == "memoryReview" && string(p[field]) != "true" {
					t.Fatal("wrong flag")
				}
				p["updatedAt"] = json.RawMessage(`3`)
				view := map[string]any{"ns": "dsh-bot", "revision": 9, "value": map[string]any{"items": []any{p}}}
				value, _ := json.Marshal(map[string]any{"view": view, "warnings": []string{"fixture rename warning"}})
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"result":{"ok":true,"value":` + string(value) + `}}`))}, nil
			})}}
			var receipt DshbotCapabilityReceipt
			var err error
			if field == "memoryReview" {
				receipt, err = c.DshbotSaveMemoryReview(context.Background(), DshbotMemoryReviewCommand{Expected: base, Enabled: true})
			} else {
				desired := CloneDshbotCapabilityPolicies(base.Policies)
				desired["tools"] = DshbotCapabilityPolicy{Mode: "selected", Names: []string{"read"}}
				receipt, err = c.DshbotSaveCapabilities(context.Background(), DshbotCapabilityCommand{Expected: base, Desired: desired})
			}
			if err != nil || !receipt.Attempted || calls != 1 || len(receipt.Warnings) != 1 {
				t.Fatal(err, receipt, calls)
			}
		})
	}
}
func TestBotMemoryReviewRejectMissingUnchangedAndForeignFields(t *testing.T) {
	base := memoryReviewFixture()
	if err := ValidateDshbotMemoryReviewCommand(DshbotMemoryReviewCommand{Expected: base, Enabled: false}); err == nil {
		t.Fatal("unchanged allowed")
	}
	missing := CloneDshbotCapabilities(base)
	delete(missing.Profile, "memoryReview")
	if err := ValidateDshbotMemoryReviewCommand(DshbotMemoryReviewCommand{Expected: missing, Enabled: true}); err == nil {
		t.Fatal("missing flag defaulted")
	}
	c := Client{BaseURL: "http://127.0.0.1", HTTPClient: &http.Client{Transport: memoryReviewTransport(func(*http.Request) (*http.Response, error) { t.Fatal("unexpected mutation"); return nil, nil })}}
	if _, err := c.saveDshbotProfileField(context.Background(), base, "hidden", json.RawMessage(`true`)); err == nil {
		t.Fatal("invented editor field")
	}
	copy := CloneDshbotCapabilities(base)
	copy.Profile["model"] = json.RawMessage(`{"provider":"","model":"","reasoningEffort":"","unknown":true}`)
	if ValidateDshbotMemoryReviewCommand(DshbotMemoryReviewCommand{Expected: copy, Enabled: true}) == nil {
		t.Fatal("lossy profile allowed")
	}
}
func TestBotMemoryReviewUnknownOutcomeDoesNotRetry(t *testing.T) {
	base := memoryReviewFixture()
	calls := 0
	c := Client{BaseURL: "http://127.0.0.1", HTTPClient: &http.Client{Transport: memoryReviewTransport(func(*http.Request) (*http.Response, error) {
		calls++
		return nil, errors.New("synthetic missing receipt")
	})}}
	out, err := c.DshbotSaveMemoryReview(context.Background(), DshbotMemoryReviewCommand{Expected: base, Enabled: true})
	if err == nil || !out.Attempted || calls != 1 {
		t.Fatal("unknown retried", calls, err)
	}
	if !reflect.DeepEqual(base.Policies, memoryReviewFixture().Policies) {
		t.Fatal("policy mutated")
	}
}
