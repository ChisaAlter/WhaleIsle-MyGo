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

type senderTransport func(*http.Request) (*http.Response, error)

func (f senderTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func senderProfileFixture() DshbotCapabilityStatus {
	var p map[string]json.RawMessage
	json.Unmarshal([]byte(`{"id":"b","kind":"bot","sessionId":"s","name":"Fixture","title":"title","sectionId":"","description":"Keep description","avatar":{"kind":"blob","shape":"circle","color":"blue","dataUrl":"","crop":""},"workspaceId":"/fixture","model":{"provider":"","model":"","reasoningEffort":""},"pinned":false,"hidden":false,"pinOrder":0,"allowedSenderIds":[],"memberBotIds":[],"objective":"Existing objective","objectiveMaxRounds":16,"memoryReview":false,"notify":false,"capabilities":{"tools":{"mode":"all","names":[]},"skills":{"mode":"selected","names":[]},"mcp":{"mode":"selected","names":[]}},"maxRounds":3,"maxSpeaks":10,"inbox":[],"readInitialized":true,"lastSeenSeq":0,"createdAt":1,"updatedAt":2,"futureField":{"keep":true}}`), &p)
	var policy DshbotCapabilityPolicies
	json.Unmarshal(p["capabilities"], &policy)
	raw, _ := json.Marshal(p)
	var bot DshbotCatalogItem
	json.Unmarshal(raw, &bot)
	return DshbotCapabilityStatus{PeerIDs: []string{"peer-a", "peer-b"}, Bot: bot, Revision: 8, Profile: p, Policies: policy, Catalog: DshbotCapabilityCatalog{Active: true, SkillsAvailable: true, Tools: []DshbotCapabilityEntry{{Name: "read"}}, Skills: []DshbotCapabilityEntry{}, MCP: []DshbotCapabilityEntry{}}}
}

func TestBotSenderPolicyWriterAcceptance(t *testing.T) {
	s := senderProfileFixture()
	calls := 0
	c := Client{BaseURL: "https://fixture.invalid", HTTPClient: &http.Client{Transport: senderTransport(func(r *http.Request) (*http.Response, error) {
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
			t.Fatal("wrong revision")
		}
		delete(p, "revision")
		value, _ := json.Marshal(map[string]any{"view": map[string]any{"ns": "dsh-bot", "revision": 9, "value": map[string]any{"items": []any{p, map[string]any{"id": "peer-a", "kind": "bot"}, map[string]any{"id": "peer-b", "kind": "bot"}}}}, "item": p, "sessionId": "s"})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"result":{"ok":true,"value":` + string(value) + `}}`))}, nil
	})}}
	out, err := c.saveDshbotProfileField(context.Background(), s, "allowedSenderIds", json.RawMessage(`["peer-a"]`))
	if err != nil || !out.Attempted || calls != 1 {
		t.Fatalf("sender policy unavailable: %v calls=%d", err, calls)
	}
}

func senderStatusFixture() DshbotSenderPolicyStatus {
	p := senderProfileFixture()
	raw, _ := json.Marshal(p.Profile)
	s := DshbotSenderPolicyStatus{DshbotCapabilityStatus: p, CatalogItems: []json.RawMessage{raw, json.RawMessage(`{"id":"peer-a","kind":"bot","name":"Duplicate label","sessionId":"a"}`), json.RawMessage(`{"id":"peer-b","kind":"bot","name":"Duplicate label","sessionId":"b-session","hidden":true}`), json.RawMessage(`{"id":"room","kind":"room","memberBotIds":["b","peer-a"]}`)}}
	if err := decodeSenderCatalog(&s); err != nil {
		panic(err)
	}
	return s
}
func TestBotSenderPolicyIdentitySemanticsAndGuards(t *testing.T) {
	s := senderStatusFixture()
	old, _ := DshbotSenderPolicyValue(s)
	messages, tasks := DshbotSenderPolicyAccess(s, old)
	if len(messages) != 2 || len(tasks) != 1 || tasks[0] != "peer-a" || !s.Peers[1].Hidden {
		t.Fatal("empty-list or hidden identity semantics", messages, tasks)
	}
	c := DshbotSenderPolicyCommand{Expected: s, Values: DshbotSenderPolicyValues{Selected: true, IDs: []string{"peer-b"}}}
	if err := ValidateDshbotSenderPolicyCommand(c); err != nil || !DshbotSenderPolicyExpands(c) {
		t.Fatal("non-room direct delegation expansion not detected", err)
	}
	c.Values.IDs = []string{"peer-a"}
	if DshbotSenderPolicyExpands(c) {
		t.Fatal("restricting to existing shared-room source expands access")
	}
	for _, ids := range [][]string{{"b"}, {"room"}, {"*"}, {"external@example.invalid"}, {"peer-a", "peer-a"}, {""}, {}} {
		bad := CloneDshbotSenderPolicyCommand(c)
		bad.Values.IDs = ids
		if ValidateDshbotSenderPolicyCommand(bad) == nil {
			t.Fatal("accepted invalid selected identities", ids)
		}
	}
	bad := CloneDshbotSenderPolicyCommand(c)
	bad.Expected.Profile["model"] = json.RawMessage(`{"provider":"","model":"","reasoningEffort":"","future":true}`)
	if ValidateDshbotSenderPolicyCommand(bad) == nil {
		t.Fatal("unknown nested shape lost")
	}
	fresh := CloneDshbotSenderPolicy(s)
	fresh.CatalogItems[3] = json.RawMessage(`{"id":"room","kind":"room","memberBotIds":["b","peer-b"]}`)
	if SameDshbotSenderPolicy(s, fresh) {
		t.Fatal("same revision room drift accepted")
	}
	fresh = CloneDshbotSenderPolicy(s)
	fresh.CatalogItems = append(fresh.CatalogItems, fresh.CatalogItems[1])
	if decodeSenderCatalog(&fresh) == nil {
		t.Fatal("duplicate identity")
	}
	s.Profile["allowedSenderIds"] = json.RawMessage(`["peer-a","peer-b"]`)
	c.Expected = s
	c.Values = DshbotSenderPolicyValues{IDs: []string{}}
	if !DshbotSenderPolicyExpands(c) {
		t.Fatal("default future-peer wildcard expansion hidden")
	}
}
func TestBotSenderPolicyFullReceiptUncertaintyAndReadOnlyDiscovery(t *testing.T) {
	for _, mode := range []string{"ok", "transport", "session", "item", "extra", "missing", "unrelated", "revision", "scope"} {
		t.Run(mode, func(t *testing.T) {
			s := senderStatusFixture()
			cmd := DshbotSenderPolicyCommand{Expected: s, Values: DshbotSenderPolicyValues{Selected: true, IDs: []string{"peer-b"}}}
			calls := 0
			c := Client{BaseURL: "https://fixture.invalid", HTTPClient: &http.Client{Transport: senderTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.URL.Path != "/dshbot/bot/update" {
					t.Fatal(r.URL.Path)
				}
				var req clientRequest
				json.NewDecoder(r.Body).Decode(&req)
				raw, _ := json.Marshal(req.Payload)
				var p map[string]json.RawMessage
				json.Unmarshal(raw, &p)
				if len(p) != len(s.Profile)+1 || string(p["revision"]) != "8" {
					t.Fatal("not exact full profile CAS")
				}
				for k, v := range s.Profile {
					if k != "allowedSenderIds" && !sameCapabilityJSON(v, p[k]) {
						t.Fatal("unrelated write", k)
					}
				}
				if mode == "transport" {
					return nil, fmt.Errorf("synthetic lost receipt")
				}
				delete(p, "revision")
				sid := "s"
				rev := 9
				switch mode {
				case "session":
					sid = "wrong"
				case "extra":
					p["newField"] = json.RawMessage(`true`)
				case "missing":
					delete(p, "futureField")
				case "unrelated":
					p["description"] = json.RawMessage(`"changed"`)
				case "revision":
					rev = 8
				case "scope":
					p["id"] = json.RawMessage(`"other"`)
				}
				item := map[string]json.RawMessage{}
				for k, v := range p {
					item[k] = v
				}
				if mode == "item" {
					item["name"] = json.RawMessage(`"wrong"`)
				}
				value, _ := json.Marshal(map[string]any{"view": map[string]any{"ns": "dsh-bot", "revision": rev, "value": map[string]any{"items": []any{p, map[string]any{"id": "peer-a", "kind": "bot"}, map[string]any{"id": "peer-b", "kind": "bot"}}}}, "sessionId": sid, "item": item, "warnings": []string{"fixture warning"}})
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"result":{"ok":true,"value":` + string(value) + `}}`))}, nil
			})}}
			out, err := c.DshbotSaveSenderPolicy(context.Background(), cmd)
			if (err == nil) != (mode == "ok") || !out.Attempted || calls != 1 {
				t.Fatal(mode, err, calls)
			}
			if mode == "ok" && len(out.Warnings) != 1 {
				t.Fatal("warning lost")
			}
		})
	}
	s := senderStatusFixture()
	reads := 0
	c := Client{BaseURL: "https://fixture.invalid", HTTPClient: &http.Client{Transport: senderTransport(func(r *http.Request) (*http.Response, error) {
		reads++
		if r.URL.Path != "/dshbot/describe" {
			t.Fatal("discovery changed state", r.URL.Path)
		}
		value, _ := json.Marshal(map[string]any{"view": map[string]any{"ns": "dsh-bot", "revision": s.Revision, "value": map[string]any{"items": s.CatalogItems}}})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"result":{"ok":true,"value":` + string(value) + `}}`))}, nil
	})}}
	out, err := c.DshbotSenderPolicy(context.Background(), "b")
	if err != nil || reads != 1 || out.WriteError != "" || len(out.Peers) != 2 || len(out.SharedRoomPeerIDs) != 1 {
		t.Fatal(err, out)
	}
	if _, err := c.saveDshbotProfileFields(context.Background(), s.DshbotCapabilityStatus, map[string]json.RawMessage{"allowedSenderIds": json.RawMessage(`[]`), "notify": json.RawMessage(`true`)}); err == nil || reads != 1 {
		t.Fatal("mixed writer accepted")
	}
	if _, err := c.saveDshbotProfileFields(context.Background(), s.DshbotCapabilityStatus, map[string]json.RawMessage{"allowedSenderIds": json.RawMessage(`[]`)}, DshbotRouteCommand{}); err == nil || reads != 1 {
		t.Fatal("foreign route argument accepted")
	}
}
