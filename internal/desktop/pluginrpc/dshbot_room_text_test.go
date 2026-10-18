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

type roomTextTransport func(*http.Request) (*http.Response, error)

func (f roomTextTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

const roomTextFixtureJSON = `{"id":"room","kind":"room","sessionId":"canonical-room","name":"Fixture group","title":"","description":"preserve full description","workspaceId":"workspace-1","sectionId":"section-1","pinned":false,"hidden":false,"pinOrder":0,"readInitialized":true,"lastSeenSeq":4,"createdAt":1,"updatedAt":2,"objective":"","objectiveMaxRounds":16,"memoryReview":true,"notify":true,"allowedSenderIds":[],"memberBotIds":["a","b"],"maxRounds":3,"maxSpeaks":10,"avatar":{"kind":"blob","shape":"circle","color":"blue","crop":"","dataUrl":""},"model":{"provider":"","model":"","reasoningEffort":""},"capabilities":{"tools":{"mode":"all","names":[],"future":{"guard":true}},"skills":{"mode":"all","names":[]},"mcp":{"mode":"all","names":[]},"extra":{"nested":[1,2]}},"holds":{},"inbox":[],"futureTop":{"retain":{"bytes":"value"}}}`

func roomTextFixtureStatus() DshbotRoomEditStatus {
	var p map[string]json.RawMessage
	json.Unmarshal([]byte(roomTextFixtureJSON), &p)
	var row DshbotRoomItem
	json.Unmarshal([]byte(roomTextFixtureJSON), &row)
	return DshbotRoomEditStatus{Profile: p, Item: row, Sections: []string{"section-1"}, Catalog: DshbotRoomCatalog{Revision: 7, Items: []DshbotRoomItem{row, {DshbotCatalogItem: DshbotCatalogItem{ID: "a", Kind: "bot", SessionID: "sa"}}, {DshbotCatalogItem: DshbotCatalogItem{ID: "b", Kind: "bot", SessionID: "sb"}}}}}
}

func TestRoomTextNormalizeBoundariesAndPreservation(t *testing.T) {
	v, e := ParseDshbotRoomText(DshbotRoomTextValues{Name: "\ufeff  New group \u00a0", Description: " \nFirst line\nSecond line \ufeff"})
	if e != nil || v.Name != "New group" || v.Description != "First line\nSecond line" {
		t.Fatal(v, e)
	}
	for _, v := range []DshbotRoomTextValues{{Name: ""}, {Name: strings.Repeat("😀", 61)}, {Name: "ok", Description: strings.Repeat("😀", 6001)}, {Name: string([]byte{0xff})}} {
		if _, e := ParseDshbotRoomText(v); e == nil {
			t.Fatal("invalid text accepted")
		}
	}
	if _, e := ParseDshbotRoomText(DshbotRoomTextValues{Name: strings.Repeat("😀", 60), Description: strings.Repeat("x", 12000)}); e != nil {
		t.Fatal(e)
	}
	s := roomTextFixtureStatus()
	cmd := DshbotRoomEditCommand{Expected: s, Values: DshbotRoomEditValues{MemberBotIDs: s.Item.MemberBotIDs, MaxRounds: s.Item.MaxRounds, MaxSpeaks: s.Item.MaxSpeaks, Text: &DshbotRoomTextValues{Name: "New group", Description: ""}}}
	if e := ValidateDshbotRoomEdit(cmd); e != nil {
		t.Fatal(e)
	}
	copy := CloneDshbotRoomEditCommand(cmd)
	copy.Values.Text.Name = "changed clone"
	if cmd.Values.Text.Name != "New group" {
		t.Fatal("aliased text review")
	}
	p := DshbotRoomEditProfile(cmd)
	for k, before := range s.Profile {
		if k != "name" && k != "description" && !sameCapabilityJSON(before, p[k]) {
			t.Fatal("unrelated data changed", k)
		}
	}
	if string(p["title"]) != `""` || string(p["description"]) != `""` {
		t.Fatal("invented title or failed clear")
	}
	old := cmd
	old.Values.Text = nil
	if string(DshbotRoomEditProfile(old)["name"]) != string(s.Profile["name"]) {
		t.Fatal("legacy nil text behavior changed")
	}
	bad := CloneDshbotRoomEditCommand(cmd)
	bad.Values.Text.Name = " unreviewed "
	if ValidateDshbotRoomEdit(bad) == nil {
		t.Fatal("untrimmed review accepted")
	}
	bad = CloneDshbotRoomEditCommand(cmd)
	bad.Expected.Profile["title"] = json.RawMessage(`"room title"`)
	bad.Expected.Item.Title = "room title"
	if ValidateDshbotRoomEdit(bad) == nil {
		t.Fatal("fabricated room title allowed")
	}
}
func TestRoomTextExactCASReceiptAndWarnings(t *testing.T) {
	s := roomTextFixtureStatus()
	cmd := DshbotRoomEditCommand{Expected: s, Values: DshbotRoomEditValues{MemberBotIDs: s.Item.MemberBotIDs, MaxRounds: s.Item.MaxRounds, MaxSpeaks: s.Item.MaxSpeaks, Text: &DshbotRoomTextValues{Name: "New group", Description: "Full multiline\ndescription without truncation"}}}
	calls := 0
	mode := "ok"
	client := &Client{BaseURL: "http://127.0.0.1", HTTPClient: &http.Client{Transport: roomTextTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		var req struct {
			RPCID, Method string
			Payload       map[string]json.RawMessage
		}
		json.NewDecoder(r.Body).Decode(&req)
		want := DshbotRoomEditProfile(cmd)
		want["revision"] = json.RawMessage("7")
		w, _ := json.Marshal(want)
		got, _ := json.Marshal(req.Payload)
		if req.Method != "bot/update" || !sameCapabilityJSON(w, got) {
			t.Fatal("wrong exact update", req.Method, string(got))
		}
		if mode == "lost" {
			return nil, fmt.Errorf("response lost")
		}
		p := DshbotRoomEditProfile(cmd)
		p["updatedAt"] = json.RawMessage("8")
		item := CloneDshbotRoomEdit(cmd.Expected).Profile
		for k, v := range p {
			item[k] = v
		}
		item["avatar"] = json.RawMessage(`{"kind":"blob","shape":"circle","color":"blue"}`)
		if mode == "wrong-item" {
			item["description"] = json.RawMessage(`"wrong"`)
		}
		if mode == "changed-other" {
			p["futureTop"] = json.RawMessage(`{}`)
		}
		items := []any{map[string]string{"id": "a", "kind": "bot", "sessionId": "sa"}, map[string]string{"id": "b", "kind": "bot", "sessionId": "sb"}, p}
		view := map[string]any{"ns": "dsh-bot", "revision": 8, "value": map[string]any{"items": items, "sections": []any{map[string]string{"id": "section-1"}}}}
		value := map[string]any{"view": view, "item": item, "sessionId": "canonical-room", "warnings": []string{"Bot profile was saved, but Session title rename failed: complete detail"}}
		b, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": req.RPCID, "result": map[string]any{"ok": true, "value": value}})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(b)))}, nil
	})}}
	for _, v := range []string{"ok", "wrong-item", "changed-other", "lost"} {
		mode = v
		before := calls
		out, e := client.DshbotSaveRoomEdit(context.Background(), cmd)
		if calls != before+1 || !out.Attempted {
			t.Fatal("retry or no exact attempt")
		}
		if v == "ok" {
			if e != nil || out.Status.Item.Name != "New group" || out.Status.Item.Description != cmd.Values.Text.Description {
				t.Fatal(out, e)
			}
		} else if e == nil {
			t.Fatal("bad receipt accepted", v)
		}
		if v != "lost" && len(out.Warnings) != 1 {
			t.Fatal("rename warning lost", v)
		}
	}
	fresh := CloneDshbotRoomEdit(s)
	fresh.Catalog.Revision++
	fresh.Profile = DshbotRoomEditProfile(cmd)
	raw, _ := json.Marshal(fresh.Profile)
	json.Unmarshal(raw, &fresh.Item)
	if e := DshbotRoomEditApplied(cmd, fresh); e != nil {
		t.Fatal(e)
	}
	fresh.Profile["description"] = json.RawMessage(`"different"`)
	if DshbotRoomEditApplied(cmd, fresh) == nil {
		t.Fatal("text mismatch adopted")
	}
}
