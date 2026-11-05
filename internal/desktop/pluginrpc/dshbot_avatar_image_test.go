package pluginrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"io"
	"net/http"
	"strings"
	"testing"
	"whaleisle/internal/desktop/avatarimage"
)

type imageTransport func(*http.Request) (*http.Response, error)

func (f imageTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	response, err := f(r)
	return fixtureResponseIdentity(r, response, err)
}
func imageProfileFixture() DshbotCapabilityStatus {
	var p map[string]json.RawMessage
	json.Unmarshal([]byte(`{"id":"b","kind":"bot","sessionId":"s","name":"Fixture","title":"title","sectionId":"","description":"Keep description","avatar":{"kind":"blob","shape":"circle","color":"blue","dataUrl":"","crop":""},"workspaceId":"/fixture","model":{"provider":"","model":"","reasoningEffort":""},"pinned":false,"hidden":false,"pinOrder":0,"allowedSenderIds":[],"memberBotIds":[],"objective":"Existing objective","objectiveMaxRounds":16,"memoryReview":false,"notify":false,"capabilities":{"tools":{"mode":"all","names":[]},"skills":{"mode":"selected","names":[]},"mcp":{"mode":"selected","names":[]}},"maxRounds":3,"maxSpeaks":10,"inbox":[],"readInitialized":true,"lastSeenSeq":0,"createdAt":1,"updatedAt":2,"futureField":{"keep":true}}`), &p)
	var policy DshbotCapabilityPolicies
	json.Unmarshal(p["capabilities"], &policy)
	raw, _ := json.Marshal(p)
	var bot DshbotCatalogItem
	json.Unmarshal(raw, &bot)
	return DshbotCapabilityStatus{Bot: bot, Revision: 8, Profile: p, Policies: policy, Catalog: DshbotCapabilityCatalog{Active: true, SkillsAvailable: true, Tools: []DshbotCapabilityEntry{{Name: "read"}}, Skills: []DshbotCapabilityEntry{}, MCP: []DshbotCapabilityEntry{}}}
}
func TestBotAvatarImageFullProfileCASReceipts(t *testing.T) {
	var b bytes.Buffer
	png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 32, 20)))
	source, _ := avatarimage.Decode(b.Bytes(), "synthetic.png", "png")
	url, _ := source.Bake(avatarimage.DefaultCrop("circle"))
	value := DshbotAvatarValues{Kind: "image", DataURL: url, Crop: "circle"}
	for _, mode := range []string{"success", "transport", "wrong-crop", "unrelated", "unknown-key", "session"} {
		t.Run(mode, func(t *testing.T) {
			expected := imageProfileFixture()
			calls := 0
			client := Client{BaseURL: "http://127.0.0.1", HTTPClient: &http.Client{Transport: imageTransport(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.URL.Path != "/dshbot/bot/update" {
					t.Fatal("unexpected call")
				}
				var wire clientRequest
				json.NewDecoder(req.Body).Decode(&wire)
				data, _ := json.Marshal(wire.Payload)
				var profile map[string]json.RawMessage
				json.Unmarshal(data, &profile)
				if string(profile["revision"]) != "8" || len(profile) != len(expected.Profile)+1 || bytes.Contains(data, []byte("synthetic.png")) {
					t.Fatal("CAS or source metadata")
				}
				for key, raw := range expected.Profile {
					if key != "avatar" && !sameCapabilityJSON(raw, profile[key]) {
						t.Fatal("unrelated", key)
					}
				}
				got, err := DshbotAvatarValue(DshbotCapabilityStatus{Profile: profile})
				if err != nil || got != value {
					t.Fatal("image mismatch")
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
				// Real normalized item has three fields, schema-backed view five.
				av := map[string]any{"kind": "image", "dataUrl": url, "crop": "circle", "shape": "", "color": ""}
				profile["avatar"], _ = json.Marshal(av)
				if mode == "wrong-crop" {
					av["crop"] = "square"
					profile["avatar"], _ = json.Marshal(av)
				}
				if mode == "unknown-key" {
					av["extra"] = true
					profile["avatar"], _ = json.Marshal(av)
				}
				if mode == "unrelated" {
					profile["notify"] = json.RawMessage(`true`)
				}
				sid := "s"
				if mode == "session" {
					sid = "other"
				}
				response, _ := json.Marshal(map[string]any{"view": map[string]any{"ns": "dsh-bot", "revision": 9, "value": map[string]any{"items": []any{profile}}}, "item": item, "sessionId": sid, "warnings": []string{"presentation warning"}})
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"result":{"ok":true,"value":` + string(response) + `}}`))}, nil
			})}}
			out, err := client.DshbotSaveAvatar(context.Background(), DshbotAvatarCommand{Expected: expected, Values: value})
			if (err == nil) != (mode == "success") || !out.Attempted || calls != 1 {
				t.Fatal(mode, err, calls)
			}
			if mode == "success" && len(out.Warnings) != 1 {
				t.Fatal("warning lost")
			}
		})
	}
	expected := imageProfileFixture()
	old := value
	expected.Profile["avatar"], _ = json.Marshal(old)
	next := old
	next.Crop = "square"
	if err := ValidateDshbotAvatarCommand(DshbotAvatarCommand{Expected: expected, Values: next}); err != nil {
		t.Fatal("same original dataURL crop", err)
	}
	bad := value
	bad.DataURL = "data:image/png;base64,AA=="
	if ValidateDshbotAvatarCommand(DshbotAvatarCommand{Expected: imageProfileFixture(), Values: bad}) == nil {
		t.Fatal("unbaked image")
	}
	calls := 0
	c := Client{BaseURL: "http://127.0.0.1", HTTPClient: &http.Client{Transport: imageTransport(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("must not send") })}}
	raw, _ := json.Marshal(value)
	out, err := c.saveDshbotProfileFields(context.Background(), imageProfileFixture(), map[string]json.RawMessage{"avatar": raw, "notify": json.RawMessage(`true`)})
	if err == nil || out.Attempted || calls != 0 {
		t.Fatal("mixed edit")
	}
}
