package main

import (
	"bytes"
	"math"
	"os"
	"reflect"
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

func TestSavePhoneConfigPatchRejectsInvalidWithoutWriting(t *testing.T) {
	for name, patch := range map[string]map[string]any{
		"wrong theme type":      {"theme": 42},
		"invalid theme id":      {"theme": "../dark"},
		"invalid locale":        {"locale": "de"},
		"invalid bind":          {"remoteBindAddress": "not-an-ip", "theme": "deepseek.dark"},
		"invalid public page":   {"remoteAppBaseUrl": "not-a-url", "theme": "deepseek.dark"},
		"private public page":   {"remoteAppBaseUrl": "http://[fd00::1]/pair", "theme": "deepseek.dark"},
		"public page query":     {"remoteAppBaseUrl": "https://example.com/pair?token=x", "theme": "deepseek.dark"},
		"fractional port":       {"remotePort": 3499.5},
		"port below range":      {"remotePort": 80},
		"port above range":      {"remotePort": 65536},
		"port NaN":              {"remotePort": math.NaN()},
		"port infinity":         {"remotePort": math.Inf(1)},
		"invalid mode":          {"remoteMode": "ssh"},
		"mixed invalid patch":   {"theme": "deepseek.dark", "remoteMode": "lan", "locale": 42},
		"protected mixed patch": {"theme": "deepseek.dark", "remoteToken": "not-allowed"},
	} {
		t.Run(name, func(t *testing.T) {
			store, err := config.Open(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			// Establish a real baseline, including values an invalid mixed patch
			// would otherwise partially overwrite.
			if err := store.Update(func(c *config.Config) { c.Theme = "deepseek"; c.Locale = "zh"; c.RemoteMode = "relay" }); err != nil {
				t.Fatal(err)
			}
			before := store.Get()
			raw, err := os.ReadFile(store.Path())
			if err != nil {
				t.Fatal(err)
			}
			s := &mainState{cfgStore: store}
			for attempt := 0; attempt < 12; attempt++ {
				saved, err := s.savePhoneConfigPatch(patch)
				if err == nil || saved != nil {
					t.Fatalf("invalid patch reported success: saved=%v, error=%v", saved, err)
				}
				if after := store.Get(); !reflect.DeepEqual(after, before) {
					t.Fatalf("invalid patch changed in-memory config: before=%+v after=%+v", before, after)
				}
				afterRaw, err := os.ReadFile(store.Path())
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(raw, afterRaw) {
					t.Fatal("invalid patch wrote the configuration file")
				}
			}
			if _, err := s.savePhoneConfigPatch(map[string]any{"theme": "deepseek.dark", "locale": "en"}); err != nil {
				t.Fatalf("rejection invalidated subsequent valid save: %v", err)
			}
		})
	}
}

func TestApplyRemoteConfigPatchFailureIsAtomic(t *testing.T) {
	for attempt := 0; attempt < 24; attempt++ {
		c := config.Defaults()
		before := c
		if err := applyRemoteConfigPatch(&c, map[string]any{"theme": "deepseek.dark", "remoteMode": "lan", "locale": 42}); err == nil {
			t.Fatal("invalid mixed patch was accepted")
		}
		if !reflect.DeepEqual(c, before) {
			t.Fatal("invalid mixed patch partially applied")
		}
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
