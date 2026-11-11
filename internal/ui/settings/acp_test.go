package settings

import (
	"encoding/json"
	"github.com/egoist/mygo/ui"
	"os"
	"testing"
	"whaleisle/internal/desktop/acp"
	"whaleisle/internal/ui/i18n"
)

func TestACPSettingsDraftAndModelPreferences(t *testing.T) {
	prev := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(prev)
	var agent acp.Agent
	_ = json.Unmarshal([]byte(`{"id":"local","name":"Local coding agent","command":"node","args":["agent.mjs"],"env":{"LOCAL_ENDPOINT":"http://127.0.0.1"},"registrySource":"local","permissionPolicy":"ask","enabled":true,"binaryFound":true,"configuredModels":[],"allModels":[{"id":"fast","name":"Fast"}],"favoriteModels":[],"hiddenModels":[],"futureField":{"keep":true}}`), &agent)
	a := &ACPAccess{Catalog: acp.Catalog{Agents: []acp.Agent{agent}}, Selected: "local", Available: true, Loaded: true}
	calls := 0
	a.OnSelect = func(id string) { a.Selected = id }
	a.OnAction = func(method string, p map[string]any) {
		calls++
		if method != "agent/save" && method != "agent/test" {
			t.Fatal(method)
		}
		raw, _ := json.Marshal(p["agent"])
		var saved map[string]any
		_ = json.Unmarshal(raw, &saved)
		if saved["futureField"] == nil {
			t.Fatal("dropped unknown fields")
		}
		if calls == 1 && saved["name"] != "Draft name" {
			t.Fatalf("Host refresh erased native draft: %v", saved["name"])
		}
		if method == "agent/test" && (p["id"] != "local" || saved["name"] != "Probe draft") {
			t.Fatal("ACP test ignored unsaved config", p)
		}
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Scroll(c).Fill().Padding(20).Children(func() { acpSettings(c, a, c.Theme().Text, c.Theme().TextMuted) })
	}, 880, 1400)
	if err := tt.Click("ACP Display name"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Ctrl, ui.KeyA)
	tt.Type("Draft name")
	a.Catalog.Agents[0].Name = "Host refreshed name"
	tt.Frame()
	if err := tt.Click("Save agent"); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatal(calls)
	}
	if err := tt.Click("Discard draft"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Model configuration"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Favorite fast"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Hidden fast"); err != nil {
		t.Fatal(err)
	}
	if calls != 3 {
		t.Fatal(calls)
	}
	if dir := os.Getenv("SETTINGS_GALLERY_DIR"); dir != "" {
		writeGalleryPNG(t, dir, "acp-settings-light.png", tt)
		tt.SetSize(430, 1200)
		tt.Frame()
		writeGalleryPNG(t, dir, "acp-settings-narrow.png", tt)
	}
	if err := tt.Click("ACP Display name"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Ctrl, ui.KeyA)
	tt.Type("Probe draft")
	if err := tt.Click("Test ACP"); err != nil {
		t.Fatal(err)
	}
	if calls != 4 {
		t.Fatal(calls)
	}
}

func TestACPLoginLifecycleControls(t *testing.T) {
	prev := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(prev)
	agent := acp.Agent{ID: "local", Enabled: true, Account: &acp.Account{Authenticated: true, AvailableMethods: []acp.Method{{ID: "key", Label: "API key", Fields: []acp.Field{{Key: "API_KEY", Label: "Key", Secret: true}}}}}, AuthFlow: &acp.Flow{FlowID: "flow", Phase: "waiting", Interaction: &acp.Interaction{ID: "browser", Type: "browser", RequiresConsent: true, AcceptsCallback: true, URL: "https://example.invalid/sign-in"}}}
	a := &ACPAccess{}
	methods := []string{}
	call := func(method string, p map[string]any) { methods = append(methods, method) }
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Scroll(c).Fill().Padding(20).Children(func() { acpLogin(c, a, agent, call, c.Theme().Text, c.Theme().TextMuted) })
	}, 600, 1000)
	for _, label := range []string{"Save credentials only · API key", "Allow browser sign-in", "Cancel sign-in", "Reauthorize", "Confirm reauthorization", "Sign out", "Also remove saved API keys", "Confirm sign-out"} {
		if err := tt.Click(label); err != nil {
			t.Fatal(label, err)
		}
	}
	want := []string{"agent/login", "agent/login/respond", "agent/login/cancel", "agent/login/reauth", "agent/logout"}
	if len(methods) != len(want) {
		t.Fatal(methods)
	}
	for i, v := range want {
		if methods[i] != v {
			t.Fatal(methods)
		}
	}
	if dir := os.Getenv("SETTINGS_GALLERY_DIR"); dir != "" {
		tt.SetDark(true)
		tt.Frame()
		writeGalleryPNG(t, dir, "acp-login-dark.png", tt)
	}
}

func TestACPCustomAgentKeepsChineseNameAndRejectsExistingID(t *testing.T) {
	prev := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(prev)
	a := &ACPAccess{Catalog: acp.Catalog{Agents: []acp.Agent{{ID: "existing"}}}}
	calls := 0
	call := func(method string, p map[string]any) {
		calls++
		raw, _ := json.Marshal(p["agent"])
		var saved map[string]any
		_ = json.Unmarshal(raw, &saved)
		if method != "agent/save" || saved["id"] != "my-agent" || saved["name"] != "我的编程助手" || saved["enabled"] != false {
			t.Fatal(method, saved)
		}
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Scroll(c).Fill().Padding(20).Children(func() {
			acpAgentEditor(c, a, acp.Agent{RegistrySource: "local", PermissionPolicy: "ask"}, call, c.Theme().Text, c.Theme().TextMuted)
		})
	}, 680, 1100)
	if err := tt.Click("ACP Display name"); err != nil {
		t.Fatal(err)
	}
	tt.Type("我的编程助手")
	if err := tt.Click("ACP Agent ID (lowercase letters, numbers, - or _)"); err != nil {
		t.Fatal(err)
	}
	tt.Type("existing")
	if err := tt.Click("Save agent"); err != nil {
		t.Fatal(err)
	}
	if calls != 0 || !tt.HasText("This agent ID already exists") {
		t.Fatal("custom creation overwrote existing config")
	}
	if err := tt.Click("ACP Agent ID (lowercase letters, numbers, - or _)"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Ctrl, ui.KeyA)
	tt.Type("my-agent")
	if err := tt.Click("Save agent"); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatal(calls)
	}
}
