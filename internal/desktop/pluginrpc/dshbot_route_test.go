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

type routeTransport func(*http.Request) (*http.Response, error)

func (f routeTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func routeProfileFixture() DshbotCapabilityStatus {
	var p map[string]json.RawMessage
	json.Unmarshal([]byte(`{"id":"b","kind":"bot","sessionId":"s","name":"Fixture","title":"title","sectionId":"","description":"Keep description","avatar":{"kind":"blob","shape":"circle","color":"blue","dataUrl":"","crop":""},"workspaceId":"/fixture","model":{"provider":"","model":"","reasoningEffort":""},"pinned":false,"hidden":false,"pinOrder":0,"allowedSenderIds":[],"memberBotIds":[],"objective":"Existing objective","objectiveMaxRounds":16,"memoryReview":false,"notify":false,"capabilities":{"tools":{"mode":"all","names":[]},"skills":{"mode":"selected","names":[]},"mcp":{"mode":"selected","names":[]}},"maxRounds":3,"maxSpeaks":10,"inbox":[],"readInitialized":true,"lastSeenSeq":0,"createdAt":1,"updatedAt":2,"futureField":{"keep":true}}`), &p)
	var policy DshbotCapabilityPolicies
	json.Unmarshal(p["capabilities"], &policy)
	raw, _ := json.Marshal(p)
	var bot DshbotCatalogItem
	json.Unmarshal(raw, &bot)
	return DshbotCapabilityStatus{Bot: bot, Revision: 8, Profile: p, Policies: policy, Catalog: DshbotCapabilityCatalog{Active: true, SkillsAvailable: true, Tools: []DshbotCapabilityEntry{{Name: "read"}}, Skills: []DshbotCapabilityEntry{}, MCP: []DshbotCapabilityEntry{}}}
}

func routeFixture() DshbotRouteStatus {
	return DshbotRouteStatus{DshbotCapabilityStatus: routeProfileFixture(), Options: DshbotRouteOptions{ModelsKnown: true, WorkspacesKnown: true, SessionKnown: true, Default: DshbotRouteModel{Provider: "p", Model: "m"}, Models: []DshbotRouteModelOption{{Provider: "p", Model: "m", Efforts: []string{"high"}}}, Workspaces: []DshbotRouteWorkspace{{ID: "/fixture", Path: "/fixture"}, {ID: "w2", Path: "/host/two"}}, ScratchCwd: "/host/scratch"}}
}
func TestBotRouteExactProfileSessionReceiptAndUnknown(t *testing.T) {
	for _, mode := range []string{"model", "workspace", "scratch", "transport", "session", "item", "unrelated"} {
		t.Run(mode, func(t *testing.T) {
			s := routeFixture()
			v, _ := DshbotRouteValue(s)
			v.Model = DshbotRouteModel{Provider: "p", Model: "m", ReasoningEffort: "high"}
			if mode != "model" {
				v.WorkspaceID = "w2"
			}
			if mode == "scratch" {
				v.WorkspaceID = ""
			}
			cmd := DshbotRouteCommand{Expected: s, Values: v}
			calls := 0
			c := Client{BaseURL: "http://127.0.0.1", HTTPClient: &http.Client{Transport: routeTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.URL.Path != "/dshbot/bot/update" {
					t.Fatal(r.URL.Path)
				}
				var req clientRequest
				json.NewDecoder(r.Body).Decode(&req)
				raw, _ := json.Marshal(req.Payload)
				var p map[string]json.RawMessage
				json.Unmarshal(raw, &p)
				if string(p["revision"]) != "8" {
					t.Fatal("CAS")
				}
				for k, before := range s.Profile {
					if k != "model" && k != "workspaceId" && !sameCapabilityJSON(before, p[k]) {
						t.Fatal("unrelated changed", k)
					}
				}
				if mode == "scratch" && string(p["scratchCwd"]) != `"/host/scratch"` {
					t.Fatal("scratch missing")
				}
				if mode == "transport" {
					return nil, errors.New("unknown")
				}
				delete(p, "revision")
				delete(p, "scratchCwd")
				sid := s.Bot.SessionID
				if mode != "model" {
					sid = "session-01234567-89ab-4cde-8f01-23456789abcd"
					p["sessionId"], _ = json.Marshal(sid)
				}
				if mode == "session" {
					sid = "wrong"
				}
				if mode == "unrelated" {
					p["notify"] = json.RawMessage(`true`)
				}
				item := map[string]json.RawMessage{}
				for k, v := range p {
					item[k] = v
				}
				if mode == "item" {
					item["name"] = json.RawMessage(`"wrong"`)
				}
				value, _ := json.Marshal(map[string]any{"view": map[string]any{"ns": "dsh-bot", "revision": 9, "value": map[string]any{"items": []any{p}}}, "item": item, "sessionId": sid, "warnings": []string{"fixture warning"}})
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"result":{"ok":true,"value":` + string(value) + `}}`))}, nil
			})}}
			out, err := c.DshbotSaveRoute(context.Background(), cmd)
			good := mode == "model" || mode == "workspace" || mode == "scratch"
			if (err == nil) != good || !out.Attempted || calls != 1 {
				t.Fatal(mode, err, calls)
			}
			if good && len(out.Warnings) != 1 {
				t.Fatal("warnings lost")
			}
		})
	}
}
func TestBotRouteDiscoveryAndWriteGuards(t *testing.T) {
	s := routeFixture()
	v, _ := DshbotRouteValue(s)
	v.WorkspaceID = "w2"
	cmd := DshbotRouteCommand{Expected: s, Values: v}
	if err := ValidateDshbotRouteCommand(cmd); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"locked", "unknown-session", "unknown-workspace", "missing-workspace", "bad-effort", "half-route", "nested", "unsupported-unicode"} {
		t.Run(mode, func(t *testing.T) {
			bad := CloneDshbotRouteCommand(cmd)
			switch mode {
			case "locked":
				bad.Expected.Options.WorkspaceLocked = true
			case "unknown-session":
				bad.Expected.Options.SessionKnown = false
			case "unknown-workspace":
				bad.Expected.Options.WorkspacesKnown = false
			case "missing-workspace":
				bad.Values.WorkspaceID = "arbitrary-local-path"
			case "bad-effort":
				bad.Values.Model = DshbotRouteModel{Provider: "p", Model: "m", ReasoningEffort: "unsupported"}
			case "half-route":
				bad.Values.Model.Provider = "p"
			case "nested":
				bad.Expected.Profile["model"] = json.RawMessage(`{"provider":"","model":"","reasoningEffort":"","future":true}`)
			case "unsupported-unicode":
				bad.Values.Model.Model = string([]byte{255})
			}
			if ValidateDshbotRouteCommand(bad) == nil {
				t.Fatal("accepted", mode)
			}
		})
	}
	raw := json.RawMessage(`{"groups":[{"id":"p","models":[{"id":"m","name":"Model","reasoning":{"efforts":[{"id":"high"}],"defaultEffort":"high"}}]}],"default":{"provider":"p","model":"m"}}`)
	var opts DshbotRouteOptions
	if err := DecodeDshbotRouteModels(raw, &opts); err != nil || !opts.ModelsKnown || len(opts.Models) != 1 {
		t.Fatal(err)
	}
	if err := DecodeDshbotRouteWorkspaces(json.RawMessage(`{"type":"baseline","value":{"scratchCwd":"/host/scratch","items":[{"workspaceId":"w","path":"/host/w","title":"W"}]}}`), &opts); err != nil || len(opts.Workspaces) != 1 {
		t.Fatal(err)
	}
	if err := DecodeDshbotRouteSession(json.RawMessage(`{"items":[{"sessionId":"s","blank":true}]}`), "s", &opts); err != nil || opts.WorkspaceLocked {
		t.Fatal(err)
	}
	DecodeDshbotRouteSession(json.RawMessage(`{"items":[{"sessionId":"s"}]}`), "s", &opts)
	if opts.SessionKnown || !opts.WorkspaceLocked {
		t.Fatal("unknown blank unlocked")
	}
	c := Client{BaseURL: "http://127.0.0.1", HTTPClient: &http.Client{Transport: routeTransport(func(*http.Request) (*http.Response, error) { t.Fatal("unsupported patch sent"); return nil, nil })}}
	_, err := c.saveDshbotProfileFields(context.Background(), s.DshbotCapabilityStatus, map[string]json.RawMessage{"model": json.RawMessage(`{}`), "workspaceId": json.RawMessage(`"w"`)})
	if err == nil {
		t.Fatal("unreviewed route accepted")
	}
}

// A route command may never smuggle route transport/receipt semantics into an
// unrelated editor's otherwise valid narrow allowlist branch.
func TestBotRouteForeignWriterArgumentsRejected(t *testing.T) {
	s := routeFixture()
	v, _ := DshbotRouteValue(s)
	v.WorkspaceID = "w2"
	cmd := DshbotRouteCommand{Expected: s, Values: v}
	c := Client{BaseURL: "http://127.0.0.1", HTTPClient: &http.Client{Transport: routeTransport(func(*http.Request) (*http.Response, error) {
		t.Fatal("foreign route arguments reached transport")
		return nil, nil
	})}}
	for _, fields := range []map[string]json.RawMessage{{"memoryReview": json.RawMessage(`true`)}, {"capabilities": s.Profile["capabilities"]}, {"objective": json.RawMessage(`"changed"`), "objectiveMaxRounds": json.RawMessage(`20`)}} {
		out, err := c.saveDshbotProfileFields(context.Background(), s.DshbotCapabilityStatus, fields, cmd)
		if err == nil || out.Attempted {
			t.Fatal("foreign route arguments accepted")
		}
	}
}
