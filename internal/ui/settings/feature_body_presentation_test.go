package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/egoist/mygo/ui"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"whaleisle/internal/accountdetails"
	"whaleisle/internal/desktop/mcpconfig"
	"whaleisle/internal/desktop/providerprofiles"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/store/session"
	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"
)

type featureTransport func(*http.Request) (*http.Response, error)

func (f featureTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func featureModelStore(t *testing.T) *store.Store {
	t.Helper()
	schema := json.RawMessage(`{"uid":4,"refs":{"1":{"type":"string"},"2":{"type":"object","dict":{"id":1}},"3":{"type":"array","inner":2},"4":{"type":"object","dict":{"models":3}}}}`)
	s := store.New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://127.0.0.1:1"}, HTTP: &http.Client{Transport: featureTransport(func(r *http.Request) (*http.Response, error) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		recorder := httptest.NewRecorder()
		value := map[string]any{"writable": true, "namespaces": []map[string]any{{"ns": "general", "revision": 1, "schema": json.RawMessage(`{"uid":1,"refs":{"1":{"type":"object","dict":{}}}}`), "value": map[string]any{}}, {"ns": "llm-deepseek", "revision": 7, "schema": schema, "value": map[string]any{"models": []map[string]any{{"id": "fixture-model"}}}}}}
		_ = json.NewEncoder(recorder).Encode(map[string]any{"type": "server-response", "rpcId": req["rpcId"], "result": map[string]any{"ok": true, "value": value}})
		return recorder.Result(), nil
	})}}, nil)
	s.RefreshDescribe(context.Background())
	deadline := time.Now().Add(time.Second)
	for s.Snapshot().Loading && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if s.Snapshot().Err != "" || len(s.Snapshot().Namespaces) != 2 {
		t.Fatalf("fixture describe %+v", s.Snapshot())
	}
	s.Select("general")
	return s
}
func TestFeatureModelsKeepStructuredEditorsAndBoundDrafts(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	s := featureModelStore(t)
	binding := s.BindProviderEditor("llm-deepseek", "deepseek", nil)
	tt := ui.NewTester(func(c *ui.Context) {
		modelNamespaceBody(c, Model{Store: s, Models: &ModelsAccess{configurationBinding: binding}}, "llm-deepseek")
	}, 780, 1200)
	if !tt.HasText("Models") || !tt.HasText("Model ID") || !tt.HasText("Model options") || !strings.Contains(binding.Snapshot().Drafts["models"], "fixture-model") {
		t.Fatal(tt.Texts())
	}
	if err := tt.Click("Add model"); err != nil {
		t.Fatal(err)
	}
	if dir := os.Getenv("SECONDARY_SNAPSHOT_DIR"); dir != "" {
		for _, width := range []int{360, 780} {
			view := ui.NewTester(func(c *ui.Context) {
				ui.Scroll(c).Fill().Padding(20).Children(func() { modelNamespaceBody(c, Model{Store: s, Models: &ModelsAccess{}}, "llm-deepseek") })
			}, width, 760)
			f, err := os.Create(filepath.Join(dir, fmt.Sprintf("settings-models-structured-%d.png", width)))
			if err != nil {
				t.Fatal(err)
			}
			if err = png.Encode(f, view.Image()); err != nil {
				t.Fatal(err)
			}
			if err = f.Close(); err != nil {
				t.Fatal(err)
			}
		}
	}
	snap := binding.Snapshot()
	if !strings.Contains(snap.Drafts["models"], `"id":""`) || s.Snapshot().SelectedNS != "general" {
		t.Fatalf("typed editor lost binding %+v", snap)
	}
}
func TestFeatureCompactSkillControlsAndGroups(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	inspect, deleted, enabled := 0, 0, false
	a := &SkillsAccess{Skills: []SkillRow{{Name: "research", Description: "Read and summarize source material", Directory: "/fixture/never-exposed-as-a-standing-path", Source: "user-dsh", Writable: true, Groups: []string{"Research"}}}, OnInspect: func(string) { inspect++ }, OnDelete: func(string) { deleted++ }, OnSetModelInvocable: func(_ string, value bool) { enabled = value }}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Gap(12).Children(func() { skillsPane(c, Model{Skills: a}, c.Theme().Text, c.Theme().TextMuted) })
	}, 780, 900)
	if tt.HasText("/fixture/never-exposed-as-a-standing-path") {
		t.Fatal("path leaked into standing row")
	}
	if err := tt.Click("Model invocation · research"); err != nil {
		t.Fatal(err)
	}
	if !enabled || inspect != 0 {
		t.Fatal("invocation control triggered row inspection")
	}
	if err := tt.Click("Delete skill · research"); err != nil {
		t.Fatal(err)
	}
	if deleted != 1 || inspect != 0 {
		t.Fatal("delete control triggered row inspection")
	}
	if err := tt.Click("Skill group · Research"); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.Find("Inspect skill · research"); ok {
		t.Fatal("collapsed group still shows skill row")
	}
}
func TestFeatureMCPCompactControlsAndAuthorizationGate(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	edited, deleted, tested, toggled := 0, 0, 0, false
	a := &MCPAccess{HasListRPC: true, Servers: []MCPServerRow{{ID: "fixture", ServerName: "Fixture server", Writable: true, Enabled: true, Status: "connected"}}, OnEdit: func(string) { edited++ }, OnDelete: func(string) { deleted++ }, OnTest: func(string) { tested++ }, OnSetEnabled: func(_ string, v bool) { toggled = !v }, OnAuthorize: func(string) { t.Fatal("authorization was not required") }}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Gap(12).Children(func() { mcpPane(c, Model{MCP: a}, c.Theme().Text, c.Theme().TextMuted) })
	}, 780, 900)
	if _, found := tt.Find("Sign in"); found {
		t.Fatal("sign-in shown for connected stdio server")
	}
	for _, label := range []string{"Edit MCP server · fixture", "Delete MCP server · fixture", "Enable MCP server · fixture", i18n.T(i18n.KeySettingsMCPTest)} {
		if err := tt.Click(label); err != nil {
			t.Fatal(err)
		}
	}
	if edited != 1 || deleted != 1 || tested != 1 || !toggled {
		t.Fatalf("compact controls lost callbacks %d %d %d %v", edited, deleted, tested, toggled)
	}
	errText := "HTTP 401 unauthorized"
	a.Catalog = &MCPCatalogAccess{Snapshot: mcpconfig.Snapshot{Rows: []api.McpServerSnapshotServersItem{{Id: "fixture", Spec: json.RawMessage(`{"transport":"streamable-http"}`), Connection: &api.McpServerSnapshotServersItemConnection{LastError: &errText}}}}}
	if !mcpNeedsSignIn(a, a.Servers[0]) {
		t.Fatal("HTTP authentication error did not expose sign-in")
	}
}

// These captures use inert callbacks equivalent to the production row actions.
func TestFeatureBodySoftwareSnapshots(t *testing.T) {
	dir := os.Getenv("SECONDARY_SNAPSHOT_DIR")
	if dir == "" {
		t.Skip("set SECONDARY_SNAPSHOT_DIR")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	noop := func() {}
	rowNoop := func(string) {}
	for _, state := range []string{"data", "empty", "loading", "error", "no-permission"} {
		name, contact := "Offline fixture account", "fixture@example.invalid"
		account := &AccountAccess{ClientReady: true, Status: "credential-stored", OnRefresh: noop, OnSignOut: noop, OnContact: noop, OnOpenUsage: func(string) {}, OnOpenTopUp: func(string) {}, UsageURL: "https://usage.fixture.invalid", TopUpURL: "https://topup.fixture.invalid", Details: accountdetails.State{Profile: accountdetails.ProfileState{Status: accountdetails.Ready, Value: accountdetails.Profile{Name: &name, Contact: &contact}}, Balance: accountdetails.BalanceState{Status: accountdetails.Ready, Wallets: []accountdetails.Wallet{{Currency: "USD", Balance: "12.50"}}}}}
		models := &ModelsAccess{ProviderProfiles: &ProviderProfilesAccess{Snapshot: providerprofiles.Snapshot{Ready: true, Rows: []providerprofiles.Row{{ID: "fixture", DisplayName: "Fixture provider", Provider: "custom", Namespace: "llm-pi-ai", Removable: true}}}, OnDelete: rowNoop}, OnRefresh: noop}
		mcp := &MCPAccess{HasListRPC: true, Servers: []MCPServerRow{{ID: "fixture", ServerName: strings.Repeat("Long MCP server ", 3), Origin: "managed", Enabled: true, Writable: true, Status: "connected · 8 tools"}}, OnRefresh: noop, OnEdit: rowNoop, OnDelete: rowNoop, OnTest: rowNoop, OnSetEnabled: func(string, bool) {}, Form: &MCPFormAccess{OnOpen: noop}}
		query, source, create, hub := "", "", "", ""
		skills := &SkillsAccess{Skills: []SkillRow{{Name: "research", Source: "user-dsh", Directory: "/fixture/skills/research", Description: strings.Repeat("Long skill description to verify compact native rows. ", 4), Groups: []string{"Research"}, Writable: true, ModelInvocable: true}, {Name: "review", Source: "bundled", Description: "Independent bundled skill remains read-only", Groups: []string{"Research"}}}, Query: &query, SourceFilter: &source, CreateName: &create, HubQuery: &hub, OnInspect: rowNoop, OnDelete: rowNoop, OnOpenDirectory: rowNoop, OnSetModelInvocable: func(string, bool) {}, OnSetGroupInvocable: func(string, bool) {}, OnRefresh: noop, OnCreate: rowNoop, OnSearchHub: rowNoop}
		presets := session.AgentPresetManagerState{Loaded: true, Writable: true, DefaultID: "standard", Rows: []session.ManagedAgentPreset{{ID: "standard", BuiltIn: true, Description: "Everyday tasks with tools and planning"}, {ID: "ptc", BuiltIn: true, Description: "Programmatic tool calling"}, {ID: "custom", Name: "Fixture custom", Description: "Long custom preset metadata remains contained"}}}
		switch state {
		case "empty":
			account.Status = "signed-out"
			models.ProviderProfiles.Snapshot.Rows = nil
			mcp.Servers = nil
			skills.Skills = nil
			presets.Rows = nil
		case "loading":
			account.Loading = true
			account.Details.Balance.Status = accountdetails.Loading
			models.ProviderProfiles.Snapshot.Busy = true
			mcp.Servers = nil
			mcp.Busy = true
			skills.Busy = true
			presets.Loaded = false
			presets.Busy = true
		case "error":
			account.Err = "Fixture account error"
			models.Err = "Fixture models error"
			mcp.Err = "Fixture MCP error"
			skills.Err = "Fixture skills error"
			presets.Err = "Fixture preset error"
		case "no-permission":
			account.ClientReady = false
			models.ProviderProfiles.Snapshot.Rows[0].Removable = false
			mcp.Servers[0].Writable = false
			skills.Skills[0].Writable = false
			presets.Writable = false
		}
		selected := ""
		pages := map[string]func(*ui.Context){"account": func(c *ui.Context) { accountPane(c, Model{Account: account}, c.Theme().Text, c.Theme().TextMuted) }, "models": func(c *ui.Context) { modelsPane(c, Model{Models: models}, c.Theme().Text, c.Theme().TextMuted) }, "mcp": func(c *ui.Context) { mcpPane(c, Model{MCP: mcp}, c.Theme().Text, c.Theme().TextMuted) }, "skills": func(c *ui.Context) { skillsPane(c, Model{Skills: skills}, c.Theme().Text, c.Theme().TextMuted) }, "agent-presets": func(c *ui.Context) {
			agentPresetRosterPane(c, &AgentPresetsAccess{Selected: &selected, Reader: &AgentPresetReaderAccess{OnSelect: func(uint64, string) {}}}, presets, c.Theme().Text, c.Theme().TextMuted)
		}}
		for page, paint := range pages {
			for _, width := range []int{360, 780} {
				tt := ui.NewTester(func(c *ui.Context) {
					ui.Scroll(c).Fill().Padding(20).Children(func() { ui.Column(c).FillWidth().MinWidth(0).Gap(16).Children(func() { paint(c) }) })
				}, width, 760)
				f, err := os.Create(filepath.Join(dir, fmt.Sprintf("settings-%s-%s-%d.png", page, state, width)))
				if err != nil {
					t.Fatal(err)
				}
				if err = png.Encode(f, tt.Image()); err != nil {
					t.Fatal(err)
				}
				if err = f.Close(); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}
