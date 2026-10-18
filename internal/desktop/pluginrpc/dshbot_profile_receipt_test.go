package pluginrpc

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type schemaReceiptTransport func(*http.Request) (*http.Response, error)

func (f schemaReceiptTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func schemaReceiptFixture() DshbotCapabilityStatus {
	var p map[string]json.RawMessage
	json.Unmarshal([]byte(`{"id":"b","kind":"bot","sessionId":"s","name":"Fixture","title":"title","sectionId":"","description":"Keep description","avatar":{"kind":"blob","shape":"circle","color":"blue","dataUrl":"","crop":""},"workspaceId":"/fixture","model":{"provider":"","model":"","reasoningEffort":""},"pinned":false,"hidden":false,"pinOrder":0,"allowedSenderIds":[],"memberBotIds":[],"objective":"Existing objective","objectiveMaxRounds":16,"memoryReview":false,"notify":false,"capabilities":{"tools":{"mode":"all","names":[]},"skills":{"mode":"selected","names":[]},"mcp":{"mode":"selected","names":[]}},"maxRounds":3,"maxSpeaks":10,"inbox":[],"readInitialized":true,"lastSeenSeq":0,"createdAt":1,"updatedAt":2,"futureField":{"keep":true}}`), &p)
	var policy DshbotCapabilityPolicies
	json.Unmarshal(p["capabilities"], &policy)
	raw, _ := json.Marshal(p)
	var bot DshbotCatalogItem
	json.Unmarshal(raw, &bot)
	return DshbotCapabilityStatus{Bot: bot, Revision: 8, Profile: p, Policies: policy, Catalog: DshbotCapabilityCatalog{Active: true, SkillsAvailable: true, Tools: []DshbotCapabilityEntry{{Name: "read"}}, Skills: []DshbotCapabilityEntry{}, MCP: []DshbotCapabilityEntry{}}}
}

const schemaReceiptRoomJSON = `{"id":"room","kind":"room","sessionId":"canonical-room","name":"Fixture group","title":"","description":"preserve full description","workspaceId":"workspace-1","sectionId":"section-1","pinned":false,"hidden":false,"pinOrder":0,"readInitialized":true,"lastSeenSeq":4,"createdAt":1,"updatedAt":2,"objective":"","objectiveMaxRounds":16,"memoryReview":true,"notify":true,"allowedSenderIds":[],"memberBotIds":["a","b"],"maxRounds":3,"maxSpeaks":10,"avatar":{"kind":"blob","shape":"circle","color":"blue","crop":"","dataUrl":""},"model":{"provider":"","model":"","reasoningEffort":""},"capabilities":{"tools":{"mode":"all","names":[],"future":{"guard":true}},"skills":{"mode":"all","names":[]},"mcp":{"mode":"all","names":[]},"extra":{"nested":[1,2]}},"holds":{},"inbox":[],"futureTop":{"retain":{"bytes":"value"}}}`

func schemaReceiptRoomStatus(t *testing.T) DshbotRoomEditStatus {
	t.Helper()
	var p map[string]json.RawMessage
	json.Unmarshal([]byte(schemaReceiptRoomJSON), &p)
	raw := schemaReceiptRoomView(p, 7)
	s, err := decodeDshbotRoomEdit(raw, "room")
	if err != nil || s.WriteError != "" {
		t.Fatal(s.WriteError, err)
	}
	return s
}
func schemaReceiptRoomView(p map[string]json.RawMessage, revision int) json.RawMessage {
	v, _ := json.Marshal(map[string]any{"ns": "dsh-bot", "revision": revision, "value": map[string]any{"items": []any{map[string]string{"id": "a", "kind": "bot", "name": "Alpha", "sessionId": "sa"}, map[string]string{"id": "b", "kind": "bot", "name": "Beta", "sessionId": "sb"}, map[string]string{"id": "c", "kind": "bot", "name": "Gamma", "sessionId": "sc"}, p}, "sections": []any{map[string]string{"id": "section-1"}}}})
	return v
}
func schemaReceiptRoomCreateCommand() DshbotRoomCreateCommand {
	return DshbotRoomCreateCommand{ID: "room-fixture", SessionID: "session-room-fixture", Expected: DshbotRoomCatalog{Revision: 7, Items: []DshbotRoomItem{{DshbotCatalogItem: DshbotCatalogItem{ID: "a", Kind: "bot", Name: "A", SessionID: "sa"}}, {DshbotCatalogItem: DshbotCatalogItem{ID: "b", Kind: "bot", Name: "B", SessionID: "sb"}}}}, Draft: DshbotRoomDraft{Name: "Fixture room", Description: "Full description", ScratchCwd: "/fixture", MemberBotIDs: []string{"a", "b"}, MaxRounds: 3, MaxSpeaks: 10}}
}
func schemaReceiptRoomCreateCreated(cmd DshbotRoomCreateCommand) DshbotRoomItem {
	return DshbotRoomItem{DshbotCatalogItem: DshbotCatalogItem{ID: cmd.ID, SessionID: cmd.SessionID, Kind: "room", Name: cmd.Draft.Name, WorkspaceID: cmd.Draft.WorkspaceID, MemberBotIDs: append([]string(nil), cmd.Draft.MemberBotIDs...)}, Description: cmd.Draft.Description, MaxRounds: cmd.Draft.MaxRounds, MaxSpeaks: cmd.Draft.MaxSpeaks}
}
func schemaReceiptDuplicateItem(cmd DshbotBotDuplicateCommand, id, sid string) DshbotBotItem {
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
func schemaReceiptRaw(v any) json.RawMessage { raw, _ := json.Marshal(v); return raw }
func schemaReceiptProfile(p map[string]json.RawMessage) map[string]json.RawMessage {
	r := map[string]json.RawMessage{}
	for k, v := range p {
		r[k] = append(json.RawMessage(nil), v...)
	}
	return r
}
func schemaReceiptReply(items []any, item any, sid string, revision int) (*http.Response, error) {
	raw := schemaReceiptRaw(map[string]any{"result": map[string]any{"ok": true, "value": map[string]any{"view": map[string]any{"ns": "dsh-bot", "revision": revision, "value": map[string]any{"items": items}}, "item": item, "sessionId": sid, "warnings": []string{"fixture warning"}}}})
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
}

const schemaReceiptBlob3 = `{"kind":"blob","shape":"circle","color":"blue"}`
const schemaReceiptBlob5 = `{"kind":"blob","shape":"circle","color":"blue","dataUrl":"","crop":""}`

func TestBotAvatarSchemaReceiptAcrossConsumers(t *testing.T) {
	for _, consumer := range []string{"text", "sender", "route"} {
		for _, mode := range []string{"success", "nonempty-default", "unknown-avatar", "null-default", "unrelated", "extra", "missing-holds"} {
			t.Run(consumer+"/"+mode, func(t *testing.T) {
				s := schemaReceiptFixture()
				s.Profile["holds"] = json.RawMessage(`{}`)
				s.PeerIDs = []string{"peer"}
				peer := map[string]string{"id": "peer", "kind": "bot", "sessionId": "peer-session", "name": "Peer"}
				calls := 0
				c := Client{BaseURL: "https://fixture.invalid", HTTPClient: &http.Client{Transport: schemaReceiptTransport(func(r *http.Request) (*http.Response, error) {
					calls++
					if r.URL.Path != "/dshbot/bot/update" {
						t.Fatal(r.URL.Path)
					}
					var req struct{ Payload map[string]json.RawMessage }
					json.NewDecoder(r.Body).Decode(&req)
					p := req.Payload
					if string(p["revision"]) != "8" {
						t.Fatal("CAS")
					}
					delete(p, "revision")
					p["updatedAt"] = json.RawMessage(`3`)
					item := schemaReceiptProfile(p)
					item["avatar"] = json.RawMessage(schemaReceiptBlob3)
					switch mode {
					case "nonempty-default":
						item["avatar"] = json.RawMessage(`{"kind":"blob","shape":"circle","color":"blue","crop":"circle"}`)
					case "unknown-avatar":
						item["avatar"] = json.RawMessage(`{"kind":"blob","shape":"circle","color":"blue","future":true}`)
					case "null-default":
						item["avatar"] = json.RawMessage(`{"kind":"blob","shape":"circle","color":"blue","crop":null}`)
					case "unrelated":
						item["description"] = json.RawMessage(`"unreviewed"`)
					case "extra":
						item["future-key"] = json.RawMessage(`true`)
					case "missing-holds":
						delete(item, "holds")
					}
					return schemaReceiptReply([]any{p, peer}, item, "s", 9)
				})}}
				var out DshbotCapabilityReceipt
				var err error
				switch consumer {
				case "text":
					out, err = c.DshbotSaveTextProfile(context.Background(), DshbotTextProfileCommand{Expected: s, Values: DshbotTextProfileValues{Name: "Changed", Title: "title", Description: "Keep description"}})
				case "sender":
					status := DshbotSenderPolicyStatus{DshbotCapabilityStatus: s, CatalogItems: []json.RawMessage{schemaReceiptRaw(s.Profile), schemaReceiptRaw(peer)}}
					out, err = c.DshbotSaveSenderPolicy(context.Background(), DshbotSenderPolicyCommand{Expected: status, Values: DshbotSenderPolicyValues{Selected: true, IDs: []string{"peer"}}})
				case "route":
					status := DshbotRouteStatus{DshbotCapabilityStatus: s, Options: DshbotRouteOptions{ModelsKnown: true, WorkspacesKnown: true, SessionKnown: true, Default: DshbotRouteModel{Provider: "p", Model: "m"}, Models: []DshbotRouteModelOption{{Provider: "p", Model: "m", Efforts: []string{"high"}}}, Workspaces: []DshbotRouteWorkspace{{ID: "/fixture", Path: "/fixture"}}}}
					out, err = c.DshbotSaveRoute(context.Background(), DshbotRouteCommand{Expected: status, Values: DshbotRouteValues{Model: DshbotRouteModel{Provider: "p", Model: "m", ReasoningEffort: "high"}, WorkspaceID: "/fixture"}})
				}
				if (err == nil) != (mode == "success") || !out.Attempted || calls != 1 {
					t.Fatal(consumer, mode, err, calls)
				}
			})
		}
	}
	for _, consumer := range []string{"create", "duplicate"} {
		for _, mode := range []string{"success", "holds-null", "holds-nonempty", "holds-reversed", "unrelated", "extra", "avatar-unknown", "avatar-nonempty", "expected-avatar"} {
			t.Run(consumer+"/"+mode, func(t *testing.T) {
				create := DshbotBotCreateCommand{ID: "new", SessionID: "new-session", Expected: DshbotBotCatalog{Revision: 8, Items: []DshbotBotItem{}}, Draft: DshbotBotDraft{Name: "New", WorkspaceID: "workspace", Provider: "p", Model: "m", MemoryReview: true, Notify: true}}
				source := schemaReceiptFixture()
				delete(source.Profile, "futureField")
				source.Profile["holds"] = json.RawMessage(`{}`)
				source.Profile["objective"] = json.RawMessage(`""`)
				sourceItem, _ := decodeDshbotBotItem(schemaReceiptRaw(source.Profile))
				duplicate := DshbotBotDuplicateCommand{SourceID: sourceItem.ID, Name: "New copy", Expected: DshbotBotCatalog{Revision: 8, Items: []DshbotBotItem{sourceItem}}}
				calls := 0
				c := Client{BaseURL: "https://fixture.invalid", HTTPClient: &http.Client{Transport: schemaReceiptTransport(func(r *http.Request) (*http.Response, error) {
					calls++
					var p map[string]json.RawMessage
					if consumer == "create" {
						if r.URL.Path != "/dshbot/bot/create" {
							t.Fatal(r.URL.Path)
						}
						p = dshbotBotCreateProfile(create)
						for k, v := range map[string]any{"pinOrder": 0, "inbox": []any{}, "readInitialized": true, "lastSeenSeq": 0, "createdAt": 3, "updatedAt": 3} {
							p[k] = schemaReceiptRaw(v)
						}
					} else {
						if r.URL.Path != "/dshbot/bot/duplicate" {
							t.Fatal(r.URL.Path)
						}
						p = schemaReceiptDuplicateItem(duplicate, "new", "new-session").Profile
					}
					p["avatar"] = json.RawMessage(schemaReceiptBlob5)
					p["holds"] = json.RawMessage(`{}`)
					item := schemaReceiptProfile(p)
					item["avatar"] = json.RawMessage(schemaReceiptBlob3)
					delete(item, "holds")
					switch mode {
					case "holds-null":
						p["holds"] = json.RawMessage(`null`)
					case "holds-nonempty":
						p["holds"] = json.RawMessage(`{"x":true}`)
					case "holds-reversed":
						delete(p, "holds")
						item["holds"] = json.RawMessage(`{}`)
					case "unrelated":
						item["title"] = json.RawMessage(`"unreviewed"`)
					case "extra":
						p["unknown-top"] = json.RawMessage(`true`)
					case "avatar-unknown":
						item["avatar"] = json.RawMessage(`{"kind":"blob","shape":"circle","color":"blue","future":true}`)
					case "avatar-nonempty":
						item["avatar"] = json.RawMessage(`{"kind":"blob","shape":"circle","color":"blue","crop":"circle"}`)
					case "expected-avatar":
						p["avatar"] = json.RawMessage(`{"kind":"blob","shape":"hex","color":"blue","crop":"","dataUrl":""}`)
						item["avatar"] = p["avatar"]
					}
					return schemaReceiptReply([]any{source.Profile, p}, item, "new-session", 9)
				})}}
				var err error
				attempted := false
				if consumer == "create" {
					var out DshbotBotCreateResult
					out, err = c.DshbotCreateBot(context.Background(), create)
					attempted = out.Attempted
				} else {
					var out DshbotBotDuplicateResult
					out, err = c.DshbotDuplicateBot(context.Background(), duplicate)
					attempted = out.Attempted
				}
				if (err == nil) != (mode == "success") || !attempted || calls != 1 {
					t.Fatal(consumer, mode, err, calls)
				}
			})
		}
	}
	t.Run("snapshots-remain-strict", func(t *testing.T) {
		a := schemaReceiptFixture()
		b := CloneDshbotCapabilities(a)
		b.Profile["avatar"] = json.RawMessage(schemaReceiptBlob3)
		if SameDshbotCapabilities(a, b) {
			t.Fatal("CAS normalized")
		}
		ai, _ := decodeDshbotBotItem(schemaReceiptRaw(a.Profile))
		bi, _ := decodeDshbotBotItem(schemaReceiptRaw(b.Profile))
		if sameDshbotBotItem(ai, bi) || SameDshbotBotCatalog(DshbotBotCatalog{Revision: 8, Items: []DshbotBotItem{ai}}, DshbotBotCatalog{Revision: 8, Items: []DshbotBotItem{bi}}) {
			t.Fatal("catalog CAS normalized")
		}
	})
	t.Run("rooms-unaffected-real-calls", func(t *testing.T) {
		create := schemaReceiptRoomCreateCommand()
		calls := 0
		c := Client{BaseURL: "https://fixture.invalid", HTTPClient: &http.Client{Transport: schemaReceiptTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			p := map[string]json.RawMessage{}
			json.Unmarshal(schemaReceiptRaw(schemaReceiptRoomCreateCreated(create)), &p)
			item := schemaReceiptProfile(p)
			item["avatar"] = json.RawMessage(schemaReceiptBlob3)
			p["avatar"] = json.RawMessage(schemaReceiptBlob5)
			p["holds"] = json.RawMessage(`{}`)
			return schemaReceiptReply([]any{create.Expected.Items[0], create.Expected.Items[1], p}, item, create.SessionID, 8)
		})}}
		if out, err := c.DshbotCreateRoom(context.Background(), create); err != nil || !out.Attempted {
			t.Fatal(out, err)
		}
		edit := DshbotRoomEditCommand{Expected: schemaReceiptRoomStatus(t), Values: DshbotRoomEditValues{MemberBotIDs: []string{"b", "c"}, MaxRounds: 2, MaxSpeaks: 4}}
		c.HTTPClient.Transport = schemaReceiptTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			p := DshbotRoomEditProfile(edit)
			p["updatedAt"] = json.RawMessage(`3`)
			item := schemaReceiptProfile(p)
			item["avatar"] = json.RawMessage(schemaReceiptBlob3)
			view := schemaReceiptRoomView(p, 8)
			body := schemaReceiptRaw(map[string]any{"result": map[string]any{"ok": true, "value": map[string]any{"view": view, "item": item, "sessionId": edit.Expected.Item.SessionID}}})
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(body)))}, nil
		})
		if out, err := c.DshbotSaveRoomEdit(context.Background(), edit); err != nil || !out.Attempted {
			t.Fatal(out, err)
		}
		if calls != 2 {
			t.Fatal(calls)
		}
	})
}
