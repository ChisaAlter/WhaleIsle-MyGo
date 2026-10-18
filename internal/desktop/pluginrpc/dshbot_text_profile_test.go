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

type textProfileTransport func(*http.Request) (*http.Response, error)

func (f textProfileTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func textProfileFixture() DshbotCapabilityStatus {
	var p map[string]json.RawMessage
	json.Unmarshal([]byte(`{"id":"b","kind":"bot","sessionId":"s","name":"Fixture","title":"title","sectionId":"","description":"Keep description","avatar":{"kind":"blob","shape":"circle","color":"blue","dataUrl":"","crop":""},"workspaceId":"/fixture","model":{"provider":"","model":"","reasoningEffort":""},"pinned":false,"hidden":false,"pinOrder":0,"allowedSenderIds":[],"memberBotIds":[],"objective":"Existing objective","objectiveMaxRounds":16,"memoryReview":false,"notify":false,"capabilities":{"tools":{"mode":"all","names":[]},"skills":{"mode":"selected","names":[]},"mcp":{"mode":"selected","names":[]}},"maxRounds":3,"maxSpeaks":10,"inbox":[],"readInitialized":true,"lastSeenSeq":0,"createdAt":1,"updatedAt":2,"futureField":{"keep":true}}`), &p)
	var policy DshbotCapabilityPolicies
	json.Unmarshal(p["capabilities"], &policy)
	raw, _ := json.Marshal(p)
	var bot DshbotCatalogItem
	json.Unmarshal(raw, &bot)
	return DshbotCapabilityStatus{Bot: bot, Revision: 8, Profile: p, Policies: policy, Catalog: DshbotCapabilityCatalog{Active: true, SkillsAvailable: true, Tools: []DshbotCapabilityEntry{{Name: "read"}}, Skills: []DshbotCapabilityEntry{}, MCP: []DshbotCapabilityEntry{}}}
}

func TestBotTextProfileNormalizationAndNarrowFields(t *testing.T) {
	v, err := ParseDshbotTextProfile(DshbotTextProfileValues{Name: "\ufeff New \u00a0", Title: "  Display  ", Description: "  Persona\nsecond line  "})
	if err != nil || v.Name != "New" || v.Title != "Display" || v.Description != "Persona\nsecond line" {
		t.Fatal(v, err)
	}
	for _, v := range []DshbotTextProfileValues{{Name: " "}, {Name: strings.Repeat("😀", 61)}, {Name: "name", Title: strings.Repeat("a", 121)}, {Name: "name", Description: strings.Repeat("😀", 6001)}, {Name: "name", Description: string([]byte{255})}} {
		if _, err := ParseDshbotTextProfile(v); err == nil {
			t.Fatal("invalid text accepted")
		}
	}
	max := DshbotTextProfileValues{Name: strings.Repeat("😀", 60), Title: strings.Repeat("😀", 60), Description: strings.Repeat("😀", 6000)}
	if _, err := ParseDshbotTextProfile(max); err != nil {
		t.Fatal(err)
	}
	s := textProfileFixture()
	before, _ := DshbotTextProfileValue(s)
	if ValidateDshbotTextProfileCommand(DshbotTextProfileCommand{Expected: s, Values: before}) == nil {
		t.Fatal("unchanged")
	}
	cmd := DshbotTextProfileCommand{Expected: s, Values: DshbotTextProfileValues{Name: "New", Title: "", Description: ""}}
	if err := ValidateDshbotTextProfileCommand(cmd); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"title", "model"} {
		copy := CloneDshbotTextProfileCommand(cmd)
		if bad == "title" {
			delete(copy.Expected.Profile, "title")
		} else {
			copy.Expected.Profile["model"] = json.RawMessage(`{"provider":"","model":"","reasoningEffort":"","future":true}`)
		}
		if ValidateDshbotTextProfileCommand(copy) == nil {
			t.Fatal("lossy profile accepted")
		}
	}
	c := Client{BaseURL: "http://127.0.0.1", HTTPClient: &http.Client{Transport: textProfileTransport(func(*http.Request) (*http.Response, error) { t.Fatal("unsupported fields sent"); return nil, nil })}}
	for _, fields := range []map[string]json.RawMessage{{"title": json.RawMessage(`"x"`)}, {"name": json.RawMessage(`"x"`), "title": json.RawMessage(`""`)}, {"name": json.RawMessage(`"x"`), "title": json.RawMessage(`""`), "description": json.RawMessage(`""`), "notify": json.RawMessage(`true`)}} {
		out, err := c.saveDshbotProfileFields(context.Background(), s, fields)
		if err == nil || out.Attempted {
			t.Fatal("bad field set accepted")
		}
	}
}
func TestBotTextProfileExactSaveAndUncertainReceipts(t *testing.T) {
	for _, mode := range []string{"success", "transport", "session", "item", "unrelated", "extra"} {
		t.Run(mode, func(t *testing.T) {
			s := textProfileFixture()
			cmd := DshbotTextProfileCommand{Expected: s, Values: DshbotTextProfileValues{Name: "new-address", Title: "Visible title", Description: "Complete persona\nSecond line"}}
			calls := 0
			c := Client{BaseURL: "http://127.0.0.1", HTTPClient: &http.Client{Transport: textProfileTransport(func(r *http.Request) (*http.Response, error) {
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
				for k, v := range s.Profile {
					if k != "name" && k != "title" && k != "description" && !sameCapabilityJSON(v, p[k]) {
						t.Fatal("unrelated field", k)
					}
				}
				if string(p["name"]) != `"new-address"` || string(p["title"]) != `"Visible title"` {
					t.Fatal("wrong text")
				}
				if mode == "transport" {
					return nil, errors.New("unknown")
				}
				delete(p, "revision")
				p["updatedAt"] = json.RawMessage(`3`)
				sid := "s"
				if mode == "session" {
					sid = "other"
					p["sessionId"] = json.RawMessage(`"other"`)
				}
				if mode == "unrelated" {
					p["notify"] = json.RawMessage(`true`)
				}
				if mode == "extra" {
					p["unreviewed"] = json.RawMessage(`true`)
				}
				item := map[string]json.RawMessage{}
				for k, v := range p {
					item[k] = v
				}
				if mode == "item" {
					item["title"] = json.RawMessage(`"other"`)
				}
				v, _ := json.Marshal(map[string]any{"view": map[string]any{"ns": "dsh-bot", "revision": 9, "value": map[string]any{"items": []any{p}}}, "item": item, "sessionId": sid, "warnings": []string{"rename warning"}})
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"result":{"ok":true,"value":` + string(v) + `}}`))}, nil
			})}}
			out, err := c.DshbotSaveTextProfile(context.Background(), cmd)
			if (err == nil) != (mode == "success") || !out.Attempted || calls != 1 {
				t.Fatal(mode, err, calls)
			}
			if mode == "success" && len(out.Warnings) != 1 {
				t.Fatal("warning lost")
			}
		})
	}
}
