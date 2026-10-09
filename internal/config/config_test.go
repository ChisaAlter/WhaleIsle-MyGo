package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestMissingFileGivesDefaults(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if got, want := s.Get(), Defaults(); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestDefaultsMatchReferenceDefaults(t *testing.T) {
	d := Defaults()
	if d.Host != "127.0.0.1" || d.Port != 3080 || d.Theme != "deepseek" || d.Locale != "zh" {
		t.Fatalf("core defaults drifted: %+v", d)
	}
	if d.RemotePort != 6767 || d.RemoteMode != "relay" || d.RemoteBindAddress != "127.0.0.1" {
		t.Fatalf("remote defaults drifted: %+v", d)
	}
	if d.RemoteRelayUseTls || d.RemoteRelayUrl != "" || d.RemoteRelayEndpoint != "" || d.RemoteAppBaseUrl != "" {
		t.Fatalf("relay defaults drifted: %+v", d)
	}
	if !d.CloseToTray || d.OpenAtLogin || d.OpenDevTools {
		t.Fatalf("window defaults drifted: %+v", d)
	}
	if !d.HarnessAutoRestart || d.HarnessRestartMaxAttempts != 3 || d.HarnessRestartBaseDelayMs != 1000 {
		t.Fatalf("harness defaults drifted: %+v", d)
	}
	if !d.QuitAfterStart || !d.AutoStartDesktop || !d.AskOnUpdate || d.DownloadRoute != "" {
		t.Fatalf("launcher defaults drifted: %+v", d)
	}
	if d.DshbotEnabled || !d.WhaleAssistantEnabled || !d.RemoteWorkspaceEnabled {
		t.Fatalf("plugin toggles drifted: %+v", d)
	}
}

// TestAbsentKeysKeepDefaults checks that keys missing from an existing file
// keep their default value while explicit values (including false and zero)
// are honored as written.
func TestAbsentKeysKeepDefaults(t *testing.T) {
	dir := t.TempDir()
	initial := `{"port": 4000, "harnessAutoRestart": false, "whaleAssistantEnabled": false}`
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	c := s.Get()
	if !c.QuitAfterStart || !c.AutoStartDesktop || !c.AskOnUpdate || !c.RemoteWorkspaceEnabled {
		t.Fatalf("absent boolean keys lost their true defaults: %+v", c)
	}
	if c.HarnessRestartMaxAttempts != 3 || c.HarnessRestartBaseDelayMs != 1000 {
		t.Fatalf("absent harness keys lost defaults: %+v", c)
	}
	if c.RemotePort != 6767 || c.RemoteMode != "relay" {
		t.Fatalf("absent remote keys lost defaults: %+v", c)
	}
	if c.HarnessAutoRestart || c.WhaleAssistantEnabled {
		t.Fatalf("explicit false not honored: %+v", c)
	}
	if c.Port != 4000 {
		t.Fatalf("explicit port not honored: %+v", c)
	}
}

func TestUpdateKeepsUnknownKeys(t *testing.T) {
	dir := t.TempDir()
	initial := `{"port": 4000, "pet": {"enabled": true, "xRatio": 0.9}, "live2dPet": {"enabled": true, "growth": {"points": 7}}, "pluginRecovery": {"skipUserPlugins": true, "reason": "manual"}}`
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if c := s.Get(); c.Port != 4000 || !c.SkipUserPlugins {
		t.Fatalf("decoded %+v", c)
	}
	if err := s.Update(func(c *Config) {
		c.Workspace = "/src/app"
		c.Theme = "midnight"
		c.RemotePort = 7000
		c.DisabledPlugins = []string{" dsh-im ", "", "dshbot", "dsh-im"}
	}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if raw["workspace"] != "/src/app" || raw["port"] != float64(4000) {
		t.Fatalf("known keys not written: %v", raw)
	}
	if raw["theme"] != "midnight" || raw["remotePort"] != float64(7000) {
		t.Fatalf("new fields not written: %v", raw)
	}
	// pet / live2dPet / pluginRecovery ride along byte-for-byte.
	if pet, ok := raw["pet"].(map[string]any); !ok || pet["xRatio"] != 0.9 {
		t.Fatalf("pet key lost or rewritten: %v", raw["pet"])
	}
	if l2d, ok := raw["live2dPet"].(map[string]any); !ok || l2d["growth"].(map[string]any)["points"] != float64(7) {
		t.Fatalf("live2dPet key lost or rewritten: %v", raw["live2dPet"])
	}
	rec, ok := raw["pluginRecovery"].(map[string]any)
	if !ok || rec["skipUserPlugins"] != true || rec["reason"] != "manual" {
		t.Fatalf("pluginRecovery lost or rewritten: %v", raw["pluginRecovery"])
	}
	plugins, ok := raw["disabledPlugins"].([]any)
	if !ok || len(plugins) != 2 || plugins[0] != "dsh-im" || plugins[1] != "dshbot" {
		t.Fatalf("disabledPlugins not trimmed/deduped: %v", raw["disabledPlugins"])
	}
	// Round-trip: a second Open+Update must still keep the blobs.
	s2, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s2.Update(func(c *Config) { c.Port = 4001 }); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatal(err)
	}
	raw = map[string]any{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["pet"]; !ok {
		t.Fatal("pet lost on second save")
	}
	if rec, ok := raw["pluginRecovery"].(map[string]any); !ok || rec["reason"] != "manual" {
		t.Fatal("pluginRecovery lost on second save")
	}
	if _, ok := raw["components"]; ok {
		t.Fatal("components must not be invented when absent")
	}
}

func TestComponentsCarriedThrough(t *testing.T) {
	dir := t.TempDir()
	initial := `{"components": {"terminal": {"enabled": true}}}`
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if string(s.Get().Components) != `{"terminal": {"enabled": true}}` {
		t.Fatalf("components not decoded: %s", s.Get().Components)
	}
	if err := s.Update(func(c *Config) { c.Port = 4000 }); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	// MarshalIndent re-indents embedded raw JSON, so compare values, not bytes.
	comp, ok := raw["components"].(map[string]any)
	if !ok {
		t.Fatalf("components lost on save: %v", raw["components"])
	}
	term, ok := comp["terminal"].(map[string]any)
	if !ok || term["enabled"] != true {
		t.Fatalf("components rewritten on save: %v", raw["components"])
	}
}

func TestNormalizeClamps(t *testing.T) {
	d := Defaults()
	c := normalize(Config{
		Port:                      70000,
		Theme:                     "bad theme!",
		Locale:                    "fr",
		RemotePort:                80,
		RemoteMode:                "ssh",
		RemoteBindAddress:         "999.1.1.1",
		RemoteRelayUrl:            "https://relay.example.com/base",
		RemoteAppBaseUrl:          "http://192.168.1.5:3389/dshd",
		HarnessRestartMaxAttempts: 99,
		HarnessRestartBaseDelayMs: 10,
		DownloadRoute:             "gitee",
		DisabledPlugins:           []string{"  ", "x", "x"},
		SidebarWidth:              10,
		TerminalHeight:            5,
	})
	if c.Port != d.Port || c.Theme != d.Theme || c.Locale != d.Locale {
		t.Fatalf("core clamp failed: %+v", c)
	}
	if c.RemotePort != d.RemotePort || c.RemoteMode != "relay" || c.RemoteBindAddress != d.RemoteBindAddress {
		t.Fatalf("remote clamp failed: %+v", c)
	}
	if c.RemoteRelayUrl != "relay.example.com:443" || c.RemoteRelayEndpoint != "relay.example.com:443" || c.RemoteRelayUseTls {
		t.Fatalf("relay resolution failed: %+v", c)
	}
	if c.RemoteAppBaseUrl != "" {
		t.Fatalf("private-LAN app base URL must collapse to empty: %q", c.RemoteAppBaseUrl)
	}
	if c.HarnessRestartMaxAttempts != 3 || c.HarnessRestartBaseDelayMs != 1000 {
		t.Fatalf("harness clamp failed: %+v", c)
	}
	if c.DownloadRoute != "cnb" {
		t.Fatalf("gitee alias not mapped: %q", c.DownloadRoute)
	}
	if len(c.DisabledPlugins) != 1 || c.DisabledPlugins[0] != "x" {
		t.Fatalf("disabledPlugins clamp failed: %v", c.DisabledPlugins)
	}
	if c.SidebarWidth != d.SidebarWidth || c.TerminalHeight != d.TerminalHeight {
		t.Fatalf("layout clamp failed: %+v", c)
	}
}

func TestNormalizeKeepsValidValues(t *testing.T) {
	c := normalize(Config{
		Port:                      5000,
		Theme:                     "midnight.blue",
		Locale:                    "en",
		RemotePort:                7700,
		RemoteMode:                "lan",
		RemoteBindAddress:         "0.0.0.0",
		RemoteRelayUrl:            "relay.example.com",
		RemoteAppBaseUrl:          "https://app.example.com/dshd/",
		HarnessRestartMaxAttempts: 5,
		HarnessRestartBaseDelayMs: 2000,
		DownloadRoute:             "github",
	})
	if c.Port != 5000 || c.Theme != "midnight.blue" || c.Locale != "en" {
		t.Fatalf("core values rewritten: %+v", c)
	}
	if c.RemotePort != 7700 || c.RemoteMode != "lan" || c.RemoteBindAddress != "0.0.0.0" {
		t.Fatalf("remote values rewritten: %+v", c)
	}
	if c.RemoteRelayEndpoint != "relay.example.com:443" || c.RemoteRelayUrl != "relay.example.com:443" {
		t.Fatalf("relay endpoint not normalized: %+v", c)
	}
	if c.RemoteAppBaseUrl != "https://app.example.com/dshd" {
		t.Fatalf("app base URL not normalized: %q", c.RemoteAppBaseUrl)
	}
	if c.HarnessRestartMaxAttempts != 5 || c.HarnessRestartBaseDelayMs != 2000 || c.DownloadRoute != "github" {
		t.Fatalf("launcher/harness values rewritten: %+v", c)
	}
}

func TestUpdateClampsOnWrite(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Update(func(c *Config) {
		c.Port = -1
		c.RemotePort = 100
		c.DownloadRoute = "gitee"
	}); err != nil {
		t.Fatal(err)
	}
	c := s.Get()
	if c.Port != Defaults().Port || c.RemotePort != Defaults().RemotePort || c.DownloadRoute != "cnb" {
		t.Fatalf("clamp not applied on update: %+v", c)
	}
}
