package main

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"whaleisle/internal/desktop/notifications"
	"whaleisle/internal/desktop/wallpapers"
)

func TestSystemToastPayloadStrictSchema(t *testing.T) {
	encode := func(raw string) string { return base64.RawURLEncoding.EncodeToString([]byte(raw)) }
	valid := systemToastPayload{Version: 1, AppID: notifications.AppUserModelID, Arguments: "mygo-notification:chat%3A1&action=open", UserInput: map[string]string{"reply": "yes"}}
	raw, _ := json.Marshal(valid)
	p, err := decodeSystemToastPayload(encode(string(raw)))
	if err != nil || p.UserInput["reply"] != "yes" {
		t.Fatal(p, err)
	}
	for _, bad := range []string{`{}`, `{"version":1,"appID":"other","arguments":"mygo-notification:x"}`, `{"version":1,"appID":"` + notifications.AppUserModelID + `","arguments":"https://example.com"}`, string(raw) + `{}`, strings.TrimSuffix(string(raw), "}") + `,"unknown":true}`} {
		if _, err := decodeSystemToastPayload(encode(bad)); err == nil {
			t.Errorf("accepted %s", bad)
		}
	}
	if _, err := decodeSystemToastPayload(strings.Repeat("a", 70000)); err == nil {
		t.Fatal("oversized payload accepted")
	}
}

func TestDesktopGlassPolicy(t *testing.T) {
	// Exact system API support is validated in the isolated native fixture.
	if got := desktopGlassMaterial("wallpaper.png", wallpapers.Defaults(), true, false); got != "" {
		t.Fatalf("wallpaper activated desktop glass: %q", got)
	}
	if got := desktopGlassMaterial("", wallpapers.Defaults(), false, false); got != "" {
		t.Fatalf("light activated desktop glass: %q", got)
	}
	if got := desktopGlassMaterial("", wallpapers.Defaults(), true, true); got != "" {
		t.Fatalf("high contrast activated desktop glass: %q", got)
	}
}
