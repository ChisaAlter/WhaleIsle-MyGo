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

type capabilityTransport func(*http.Request) (*http.Response, error)

func (f capabilityTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func capabilityProfileFixture() map[string]json.RawMessage {
	var p map[string]json.RawMessage
	json.Unmarshal([]byte(`{"id":"b","kind":"bot","sessionId":"s","name":"Fixture","title":"title","sectionId":"","description":"keep this description","avatar":{"kind":"blob","shape":"circle","color":"blue"},"workspaceId":"/fixture","model":{"provider":"","model":"","reasoningEffort":""},"pinned":false,"hidden":false,"pinOrder":0,"allowedSenderIds":[],"memberBotIds":[],"objective":"Keep existing objective","objectiveMaxRounds":16,"memoryReview":false,"notify":false,"capabilities":{"tools":{"mode":"all","names":[]},"skills":{"mode":"selected","names":[]},"mcp":{"mode":"selected","names":[]}},"maxRounds":3,"maxSpeaks":10,"inbox":[],"readInitialized":true,"lastSeenSeq":0,"createdAt":1,"updatedAt":2,"futureField":{"opaque":[1,"keep"]}}`), &p)
	return p
}
func capabilityFixture() DshbotCapabilityStatus {
	p := capabilityProfileFixture()
	var b DshbotCatalogItem
	raw, _ := json.Marshal(p)
	json.Unmarshal(raw, &b)
	var policy DshbotCapabilityPolicies
	json.Unmarshal(p["capabilities"], &policy)
	return DshbotCapabilityStatus{Bot: b, Revision: 8, Profile: p, Policies: policy, Catalog: DshbotCapabilityCatalog{Active: true, SkillsAvailable: true, Tools: []DshbotCapabilityEntry{{Name: "read", Description: "read fixture"}}, Skills: []DshbotCapabilityEntry{}, MCP: []DshbotCapabilityEntry{}}}
}
func capabilityView(p map[string]json.RawMessage, revision int) string {
	raw, _ := json.Marshal(map[string]any{"ns": "dsh-bot", "revision": revision, "value": map[string]any{"items": []any{p}}})
	return string(raw)
}
func TestBotCapabilitiesWirePreservesFullProfile(t *testing.T) {
	base := capabilityFixture()
	calls := 0
	c := Client{BaseURL: "https://fixture.invalid", HTTPClient: &http.Client{Transport: capabilityTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		var req clientRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		payload := req.Payload.(map[string]any)
		response := ""
		switch req.Method {
		case "describe":
			if !reflect.DeepEqual(payload, map[string]any{"botId": "b"}) {
				t.Fatal(payload)
			}
			response = `{"view":` + capabilityView(base.Profile, 8) + `,"capabilities":{"active":true,"skillsAvailable":true,"tools":[{"name":"read","description":"read fixture"}],"skills":[],"mcp":[]}}`
		case "bot/update":
			if payload["revision"] != float64(8) || payload["id"] != "b" {
				t.Fatal(payload)
			}
			delete(payload, "revision")
			praw, _ := json.Marshal(payload)
			var got map[string]json.RawMessage
			json.Unmarshal(praw, &got)
			for key, value := range base.Profile {
				if key != "capabilities" && !sameCapabilityJSON(value, got[key]) {
					t.Fatal("unrelated profile changed", key)
				}
			}
			var policies DshbotCapabilityPolicies
			json.Unmarshal(got["capabilities"], &policies)
			if policies["tools"].Mode != "selected" || len(policies["tools"].Names) != 1 {
				t.Fatal(policies)
			}
			got["updatedAt"] = json.RawMessage(`3`)
			response = `{"view":` + capabilityView(got, 9) + `,"warnings":["fixture rename warning"]}`
		default:
			t.Fatal("unexpected method", req.Method)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"result":{"ok":true,"value":` + response + `}}`))}, nil
	})}}
	read, err := c.DshbotCapabilities(context.Background(), "b")
	if err != nil || read.WriteError != "" {
		t.Fatal(err, read.WriteError)
	}
	desired := CloneDshbotCapabilityPolicies(read.Policies)
	desired["tools"] = DshbotCapabilityPolicy{Mode: "selected", Names: []string{"read"}}
	out, err := c.DshbotSaveCapabilities(context.Background(), DshbotCapabilityCommand{Expected: read, Desired: desired})
	if err != nil || !out.Attempted || len(out.Warnings) != 1 || calls != 2 {
		t.Fatal(err, out, calls)
	}
}
func TestBotCapabilitiesRejectLossAndUnknownSelection(t *testing.T) {
	base := capabilityFixture()
	if err := ValidateDshbotCapabilityProfile(base); err != nil {
		t.Fatal(err)
	}
	for _, mutation := range []func(map[string]json.RawMessage){func(p map[string]json.RawMessage) { delete(p, "description") }, func(p map[string]json.RawMessage) {
		p["model"] = json.RawMessage(`{"provider":"","model":"","reasoningEffort":"","future":1}`)
	}, func(p map[string]json.RawMessage) {
		p["avatar"] = json.RawMessage(`{"kind":"blob","shape":"circle","color":"blue","future":1}`)
	}, func(p map[string]json.RawMessage) {
		p["capabilities"] = json.RawMessage(`{"tools":{"mode":"all","names":[],"future":1},"skills":{"mode":"all","names":[]},"mcp":{"mode":"all","names":[]}}`)
	}} {
		changed := CloneDshbotCapabilities(base)
		mutation(changed.Profile)
		if ValidateDshbotCapabilityProfile(changed) == nil {
			t.Fatal("lossy profile accepted")
		}
	}
	if _, err := capabilityNames(json.RawMessage(`["\ud800"]`)); err == nil {
		t.Fatal("unpaired surrogate would be silently rewritten")
	}
	if _, err := capabilityNames(json.RawMessage(`["\ud83d\udc33"]`)); err != nil {
		t.Fatal("valid surrogate pair rejected", err)
	}
	desired := CloneDshbotCapabilityPolicies(base.Policies)
	desired["tools"] = DshbotCapabilityPolicy{Mode: "selected", Names: []string{"invented"}}
	if ValidateDshbotCapabilityCommand(DshbotCapabilityCommand{Expected: base, Desired: desired}) == nil {
		t.Fatal("invented name accepted")
	}
	desired["tools"] = DshbotCapabilityPolicy{Mode: "selected", Names: []string{}}
	if ValidateDshbotCapabilityCommand(DshbotCapabilityCommand{Expected: base, Desired: desired}) != nil {
		t.Fatal("original selected-empty denied")
	}
	base.Catalog.Active = false
	if ValidateDshbotCapabilityCommand(DshbotCapabilityCommand{Expected: base, Desired: desired}) == nil {
		t.Fatal("inactive catalog accepted")
	}
}
func TestBotCapabilitiesCASIdentityAndUnknownNoRetry(t *testing.T) {
	base := capabilityFixture()
	fresh := CloneDshbotCapabilities(base)
	fresh.Revision++
	if SameDshbotCapabilities(base, fresh) {
		t.Fatal("revision ignored")
	}
	fresh = CloneDshbotCapabilities(base)
	fresh.Profile["futureField"] = json.RawMessage(`{"new":true}`)
	if SameDshbotCapabilities(base, fresh) {
		t.Fatal("unknown current field ignored")
	}
	desired := CloneDshbotCapabilityPolicies(base.Policies)
	desired["tools"] = DshbotCapabilityPolicy{Mode: "selected", Names: []string{}}
	calls := 0
	c := Client{BaseURL: "https://fixture.invalid", HTTPClient: &http.Client{Transport: capabilityTransport(func(*http.Request) (*http.Response, error) {
		calls++
		return nil, errors.New("synthetic uncertain receipt")
	})}}
	out, err := c.DshbotSaveCapabilities(context.Background(), DshbotCapabilityCommand{Expected: base, Desired: desired})
	if err == nil || !out.Attempted || calls != 1 {
		t.Fatal("unknown outcome retried", out, err, calls)
	}
}
