package main

import (
	"os"
	"testing"

	"whaleisle/internal/config"
)

func TestApplyRemoteConfigPatch(t *testing.T) {
	c := config.Defaults()
	if err := applyRemoteConfigPatch(&c, map[string]any{"remoteMode": "lan", "remotePort": float64(3499), "remoteEnabled": true, "remoteLanTls": false, "remoteBindAddress": "127.0.0.1", "theme": "deepseek.dark"}); err != nil {
		t.Fatal(err)
	}
	if c.RemoteMode != "lan" || c.RemotePort != 3499 || !c.RemoteEnabled || c.RemoteBindAddress != "127.0.0.1" || c.Theme != "deepseek.dark" {
		t.Fatalf("patch not applied: %+v", c)
	}
	for _, patch := range []map[string]any{
		{"remoteToken": "secret"},
		{"remoteRelayToken": "secret"},
		{"credentials": map[string]any{}},
		{"remotePort": "3499"},
		{"remoteEnabled": "yes"},
		{"theme": 42},
	} {
		if err := applyRemoteConfigPatch(&c, patch); err == nil {
			t.Fatalf("patch %v accepted", patch)
		}
	}
}

func TestRedactedRemoteConfigHasNoSecrets(t *testing.T) {
	c := config.Defaults()
	c.RemoteEnabled = true
	c.RemoteMode = "lan"
	c.RemotePort = 3499
	out := redactedRemoteConfig(c)
	if out["remoteMode"] != "lan" || out["remotePort"] != 3499 || out["remoteEnabled"] != true {
		t.Fatalf("redacted config missing fields: %v", out)
	}
	for key := range out {
		if key == "remoteToken" || key == "remoteRelayToken" || key == "credentials" {
			t.Fatalf("secret field %q leaked", key)
		}
	}
}

func TestSavePhoneConfigPatchPersistsRedacted(t *testing.T) {
	t.Setenv("WHALEISLE_USER_DATA", t.TempDir())
	store, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s := &mainState{cfgStore: store}
	saved, err := s.savePhoneConfigPatch(map[string]any{"remoteMode": "lan", "remotePort": float64(3499)})
	if err != nil {
		t.Fatal(err)
	}
	if saved["remoteMode"] != "lan" || saved["remotePort"] != 3499 {
		t.Fatalf("save result = %v", saved)
	}
	raw, err := os.ReadFile(store.Path())
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{"remoteToken", "remoteRelayToken"} {
		if contains(string(raw), needle) {
			t.Fatalf("config.json persisted %q", needle)
		}
	}
	if !contains(string(raw), `"remoteMode": "lan"`) {
		t.Fatalf("patch not persisted: %s", raw)
	}
}

func contains(haystack, needle string) bool {
	return len(needle) == 0 || (len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0)
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
