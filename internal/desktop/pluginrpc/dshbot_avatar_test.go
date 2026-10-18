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

type avatarTransport func(*http.Request) (*http.Response, error)

func (f avatarTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func avatarFixture() DshbotCapabilityStatus {
	var p map[string]json.RawMessage
	json.Unmarshal([]byte(`{"id":"b","kind":"bot","sessionId":"s","name":"Fixture","title":"title","sectionId":"","description":"Keep description","avatar":{"kind":"blob","shape":"circle","color":"blue","dataUrl":"","crop":""},"workspaceId":"/fixture","model":{"provider":"","model":"","reasoningEffort":""},"pinned":false,"hidden":false,"pinOrder":0,"allowedSenderIds":[],"memberBotIds":[],"objective":"Existing objective","objectiveMaxRounds":16,"memoryReview":false,"notify":false,"capabilities":{"tools":{"mode":"all","names":[]},"skills":{"mode":"selected","names":[]},"mcp":{"mode":"selected","names":[]}},"maxRounds":3,"maxSpeaks":10,"inbox":[],"readInitialized":true,"lastSeenSeq":0,"createdAt":1,"updatedAt":2,"futureField":{"keep":true}}`), &p)
	var policy DshbotCapabilityPolicies
	json.Unmarshal(p["capabilities"], &policy)
	raw, _ := json.Marshal(p)
	var bot DshbotCatalogItem
	json.Unmarshal(raw, &bot)
	return DshbotCapabilityStatus{Bot: bot, Revision: 8, Profile: p, Policies: policy, Catalog: DshbotCapabilityCatalog{Active: true, SkillsAvailable: true, Tools: []DshbotCapabilityEntry{{Name: "read"}}, Skills: []DshbotCapabilityEntry{}, MCP: []DshbotCapabilityEntry{}}}
}
func TestBotAvatarVocabularyAndNarrowField(t *testing.T) {
	for _, shape := range DshbotAvatarShapes() {
		for _, color := range DshbotAvatarColors() {
			v := DshbotAvatarValues{Kind: "blob", Shape: shape, Color: color}
			if got, err := ParseDshbotAvatar(v); err != nil || got != v {
				t.Fatal(v, err)
			}
		}
	}
	for _, raw := range []string{`{"kind":"blob","shape":"circle","color":"blue","future":1}`, `{"kind":"blob","shape":"unknown","color":"blue"}`, `{"kind":"blob","shape":"circle","color":"blue","dataUrl":"no"}`, `{"kind":"image","dataUrl":"https://fixture.invalid/p.png","crop":"circle"}`, `{"kind":"blob","shape":"circle","color":null}`} {
		if _, err := DshbotAvatarValue(DshbotCapabilityStatus{Profile: map[string]json.RawMessage{"avatar": json.RawMessage(raw)}}); err == nil {
			t.Fatal("invalid record accepted", raw)
		}
	}
	s := avatarFixture()
	before, _ := DshbotAvatarValue(s)
	if ValidateDshbotAvatarCommand(DshbotAvatarCommand{Expected: s, Values: before}) == nil {
		t.Fatal("unchanged")
	}
	image := DshbotAvatarValues{Kind: "image", DataURL: "data:image/png;base64,AA==", Crop: "circle"}
	if ValidateDshbotAvatarCommand(DshbotAvatarCommand{Expected: s, Values: image}) == nil {
		t.Fatal("image writes exposed")
	}
	old := CloneDshbotCapabilities(s)
	old.Profile["avatar"], _ = json.Marshal(image)
	if v, err := DshbotAvatarValue(old); err != nil || v != image {
		t.Fatal("image coerced")
	}
	chosen := DshbotAvatarValues{Kind: "blob", Shape: "drop", Color: "pink"}
	if err := ValidateDshbotAvatarCommand(DshbotAvatarCommand{Expected: old, Values: chosen}); err != nil {
		t.Fatal("explicit replacement", err)
	}
	unknown := CloneDshbotCapabilities(s)
	unknown.Profile["avatar"] = json.RawMessage(`{"kind":"blob","shape":"circle","color":"blue","unknown":true}`)
	if ValidateDshbotAvatarCommand(DshbotAvatarCommand{Expected: unknown, Values: chosen}) == nil {
		t.Fatal("unknown data lost")
	}
	calls := 0
	c := Client{BaseURL: "https://fixture.invalid", HTTPClient: &http.Client{Transport: avatarTransport(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("must not send") })}}
	value, _ := json.Marshal(chosen)
	for _, fields := range []map[string]json.RawMessage{{"avatar": value, "notify": json.RawMessage(`true`)}, {"shape": json.RawMessage(`"drop"`)}} {
		out, err := c.saveDshbotProfileFields(context.Background(), s, fields)
		if err == nil || out.Attempted {
			t.Fatal("mixed fields")
		}
	}
	out, err := c.saveDshbotProfileFields(context.Background(), s, map[string]json.RawMessage{"avatar": value}, DshbotRouteCommand{})
	if err == nil || out.Attempted || calls != 0 {
		t.Fatal("foreign route args")
	}
}
func TestBotAvatarExactCASAndReceipt(t *testing.T) {
	for _, mode := range []string{"success", "transport", "session", "item", "unknown-avatar", "unrelated", "extra"} {
		t.Run(mode, func(t *testing.T) {
			s := avatarFixture()
			v := DshbotAvatarValues{Kind: "blob", Shape: "triangle", Color: "orange"}
			cmd := DshbotAvatarCommand{Expected: s, Values: v}
			calls := 0
			c := Client{BaseURL: "https://fixture.invalid", HTTPClient: &http.Client{Transport: avatarTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.URL.Path != "/dshbot/bot/update" {
					t.Fatal(r.URL.Path)
				}
				var req clientRequest
				json.NewDecoder(r.Body).Decode(&req)
				raw, _ := json.Marshal(req.Payload)
				var p map[string]json.RawMessage
				json.Unmarshal(raw, &p)
				if string(p["revision"]) != "8" || len(p) != len(s.Profile)+1 {
					t.Fatal("CAS/full profile")
				}
				for key, before := range s.Profile {
					if key != "avatar" && !sameCapabilityJSON(before, p[key]) {
						t.Fatal("unrelated changed", key)
					}
				}
				avatar, err := DshbotAvatarValue(DshbotCapabilityStatus{Profile: p})
				if err != nil || avatar != v {
					t.Fatal("wrong avatar")
				}
				if mode == "transport" {
					return nil, errors.New("unknown")
				}
				delete(p, "revision")
				p["updatedAt"] = json.RawMessage(`3`)
				item := map[string]json.RawMessage{}
				for k, b := range p {
					item[k] = b
				}
				// Host item is normalized three-field next; catalog view adds schema defaults.
				p["avatar"] = json.RawMessage(`{"kind":"blob","shape":"triangle","color":"orange","dataUrl":"","crop":""}`)
				sid := "s"
				if mode == "session" {
					sid = "other"
					p["sessionId"] = json.RawMessage(`"other"`)
				}
				if mode == "item" {
					item["avatar"] = json.RawMessage(`{"kind":"blob","shape":"hex","color":"blue"}`)
				}
				if mode == "unknown-avatar" {
					item["avatar"] = json.RawMessage(`{"kind":"blob","shape":"triangle","color":"orange","future":true}`)
				}
				if mode == "unrelated" {
					p["notify"] = json.RawMessage(`true`)
				}
				if mode == "extra" {
					p["unreviewed"] = json.RawMessage(`true`)
				}
				result, _ := json.Marshal(map[string]any{"view": map[string]any{"ns": "dsh-bot", "revision": 9, "value": map[string]any{"items": []any{p}}}, "item": item, "sessionId": sid, "warnings": []string{"presentation warning"}})
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"result":{"ok":true,"value":` + string(result) + `}}`))}, nil
			})}}
			out, err := c.DshbotSaveAvatar(context.Background(), cmd)
			if (err == nil) != (mode == "success") || !out.Attempted || calls != 1 {
				t.Fatal(mode, err, calls)
			}
			if mode == "success" && len(out.Warnings) != 1 {
				t.Fatal("warning lost")
			}
		})
	}
}
