package pluginmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	manager "whaleisle/internal/pluginmanager"
	"whaleisle/internal/ui/i18n"
)

type pageFixture struct{}

type enabledOfficialFixture struct{ pageFixture }

func (f enabledOfficialFixture) Call(ctx context.Context, method string, args, out any) error {
	if method != "pluginManager/listBundles" {
		return f.pageFixture.Call(ctx, method, args, out)
	}
	raw, _ := json.Marshal([]manager.Bundle{
		{Name: "@deepseek-ai/dsh-experimental-tool-worktree", Official: true, Optional: true, Enabled: true, Availability: "installation", Meta: &manager.Meta{Title: manager.Text(`"Worktree tool"`)}},
		{Name: "@deepseek-ai/dsh-experimental-agent-team-profile", Official: true, Optional: true, Enabled: true, Availability: "installation", Meta: &manager.Meta{Title: manager.Text(`"Agent team"`)}},
	})
	return json.Unmarshal(raw, out)
}

func TestMoreRemainsReachableWhenAllOfficialPackagesAreEnabled(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	c := manager.New()
	c.Attach(enabledOfficialFixture{}, nil)
	if err := c.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	tt := ui.NewTester(func(ctx *ui.Context) { ui.Column(ctx).Fill().Children(func() { View(ctx, Model{Controller: c}) }) }, 520, 760)
	if err := tt.Click("More"); err != nil {
		t.Fatal(err)
	}
	team, ok1 := tt.Find("Agent team")
	worktree, ok2 := tt.Find("Worktree tool")
	if !ok1 || !ok2 || team.Y >= worktree.Y {
		t.Fatalf("More lost enabled packages or original official order: team=%v worktree=%v", team, worktree)
	}
	savePageSnapshot(t, tt, "plugins-more-all-enabled.png")
}

func (pageFixture) Call(_ context.Context, method string, _, out any) error {
	var value any
	switch method {
	case "pluginInventory/list":
		value = map[string]any{"managementAvailable": true, "entries": []any{}}
	case "pluginManager/listBundles":
		value = []manager.Bundle{
			{Name: "fixture-plugin", Version: "1.0.0", Availability: "profile", Installed: true, Removable: true, Enabled: true, Description: strings.Repeat("A readable plugin description. ", 8), Meta: &manager.Meta{Title: manager.Text(`"Fixture plugin"`)}, Rows: []manager.BundleRow{{RowID: "patch-id", ModuleName: "fixture-module", EntryID: "loader-fixture", Meta: &manager.Meta{Title: manager.Text(`"Fixture component"`)}}}},
			{Name: "@deepseek-ai/dsh-experimental-agent-team-profile", Official: true, Optional: true, Availability: "installation", Meta: &manager.Meta{Title: manager.Text(`"Agent team"`)}},
			{Name: "@deepseek-ai/dsh-experimental-tool-worktree", Official: true, Optional: true, Availability: "installation", Meta: &manager.Meta{Title: manager.Text(`"Worktree tool"`)}},
		}
	case "pluginManager/listPlugins":
		value = []manager.Plugin{{EntryID: "loader-fixture", Enabled: true, Phase: "active"}}
	case "pluginMarket/listIndexes":
		value = []manager.Index{{URL: "https://fixture.invalid/index.json", Title: manager.Text(`"Fixture market"`)}}
	case "pluginMarket/search":
		entry := manager.MarketEntry{Name: "fixture-discover", Title: manager.Text(`"Discover fixture"`), Description: manager.Text(`"A source-backed isolated entry with exact install versions."`), Author: "Fixture author", DefaultVersion: "2.0.0"}
		entry.Installation.Status = "not-installed"
		version := manager.MarketVersion{Version: "2.0.0", Compatibility: manager.Compatibility{Compatible: true}}
		version.Install.Spec = "fixture-discover@2.0.0"
		entry.Versions = []manager.MarketVersion{version}
		value = manager.SearchResult{URL: "https://fixture.invalid/index.json", Title: manager.Text(`"Fixture market"`), FetchedAt: 10, Entries: []manager.MarketEntry{entry}, Total: 1}
	case "pluginManager/registries":
		value = manager.RegistryInfo{}
	default:
		return fmt.Errorf("isolated page fixture refuses %s", method)
	}
	if out == nil {
		return nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}
func pageModel(t *testing.T) Model {
	t.Helper()
	c := manager.New()
	c.Attach(pageFixture{}, nil)
	if err := c.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	return Model{Controller: c, Contributions: []Contribution{
		{Kind: "item", Key: "agent-loop", Title: "Agent loop", Summary: "Configure parallel tool calls", Order: 30, Render: func(c *ui.Context, _ manager.Subject) { ui.Text(c, "Agent loop configuration fixture") }},
		{Kind: "bundle", Key: "fixture-plugin", Render: func(c *ui.Context, _ manager.Subject) { ui.Text(c, "Bundle configuration fixture") }},
		{Kind: "row", Key: "fixture-plugin#patch-id", Render: func(c *ui.Context, _ manager.Subject) { ui.Text(c, "Row configuration fixture") }},
	}}
}

// testPage is the page frame's reading of the fixture controller for a test
// that renders one of its parts directly.
func testPage(m Model, s manager.State) *page {
	return &page{m: m, s: s, a: m.Controller.Actions(s.Owner)}
}

func TestInstallTerminalShowsRunMetadataInNarrowDialog(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	m := pageModel(t)
	s := m.Controller.Snapshot().Value
	exit := 1
	s.Install = manager.InstallState{Open: true, Phase: "failed", Spec: "fixture-plugin", DetailsOpen: true, Runs: []manager.InstallRun{{JobID: "pnpm-1", Command: "pnpm add fixture-plugin", CWD: "fixture-profile", Output: "Installing fixture\nFixture failure", Chunks: []manager.InstallOutput{{Stream: "stdout", Text: "Installing fixture"}, {Stream: "stderr", Text: "Fixture failure"}}, Settled: true, ExitCode: &exit}}}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { installDialog(c, testPage(m, s)) })
	}, 520, 760)
	for _, label := range []string{"pnpm add fixture-plugin", "fixture-profile", "Installing fixture\nFixture failure", "Exit: 1"} {
		if !tt.HasText(label) {
			t.Fatalf("terminal omitted %q", label)
		}
	}
	command, ok := tt.Find("pnpm add fixture-plugin")
	if !ok || command.W <= 0 || command.X < 16 || command.X+command.W > 504 {
		t.Fatalf("terminal escaped narrow dialog: %+v", command)
	}
	savePageSnapshot(t, tt, "plugins-install-terminal-narrow.png")
}

func TestManagerNativeNavigationConfigurationsAndNarrowLayout(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	m := pageModel(t)
	tt := ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { View(c, m) }) }, 1200, 900)
	card, ok := tt.Find("Fixture plugin")
	if !ok || card.W <= 0 || card.X < 24 || card.X+card.W > 1176 {
		t.Fatalf("installed package outside native page: %+v", card)
	}
	savePageSnapshot(t, tt, "plugins-installed-wide.png")
	itemTitle, _ := tt.Find("Agent loop")
	itemSummary, _ := tt.Find("Configure parallel tool calls")
	if itemTitle.X != itemSummary.X {
		t.Fatalf("item heading detached from its summary: %v %v", itemTitle, itemSummary)
	}
	if err := tt.Click("Agent loop"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Agent loop configuration fixture") {
		t.Fatal("official item did not render its configuration slot")
	}
	if err := tt.Click("Back to plugins"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Fixture plugin"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Bundle configuration fixture") || !tt.HasText("fixture-module") {
		t.Fatal("package details lost config or declared rows")
	}
	saveSurfaceSnapshot(t, tt, "surface-plugins.png")
	savePageSnapshot(t, tt, "plugins-package-detail.png")
	if err := tt.Click("Fixture component"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Row configuration fixture") || m.Controller.Snapshot().Value.View.RowID != "patch-id" {
		t.Fatal("row configuration did not use package#patch row id")
	}
	if err := tt.Click("Back to plugins"); err != nil {
		t.Fatal(err)
	}
	if m.Controller.Snapshot().Value.View.Kind != "package" {
		t.Fatal("row breadcrumb did not return to its package")
	}
	if err := tt.Click("Back to plugins"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("More"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Worktree tool") {
		t.Fatal("more list omitted held-back official package")
	}
	savePageSnapshot(t, tt, "plugins-more.png")
	if err := tt.Click("Back to plugins"); err != nil {
		t.Fatal(err)
	}
	tt.SetSize(520, 760)
	card, ok = tt.Find("Fixture plugin")
	if !ok || card.W <= 0 || card.X < 24 || card.X+card.W > 496 {
		t.Fatalf("narrow package escaped page: %+v", card)
	}
	savePageSnapshot(t, tt, "plugins-installed-narrow.png")
}

func TestManagerDiscoverDetailsAndSourcesStaySeparate(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	m := pageModel(t)
	owner := m.Controller.Snapshot().Value.Owner
	if err := m.Controller.LoadIndexes(context.Background(), owner); err != nil {
		t.Fatal(err)
	}
	m.Controller.Actions(owner).SelectTab("discover")
	tt := ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { View(c, m) }) }, 1200, 900)
	if !tt.HasText("Fixture market") || !tt.HasText("Discover fixture") {
		t.Fatal("generic Discover did not use its Host listing")
	}
	savePageSnapshot(t, tt, "plugins-discover-wide.png")
	if err := tt.Click("Discover fixture"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("fixture-discover@2.0.0") || !tt.HasText("Install this version") {
		t.Fatal("entry detail lost source-provided exact target")
	}
	savePageSnapshot(t, tt, "plugins-discover-detail.png")
	if err := tt.Click("Back to Discover"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Switch plugin market, currently Fixture market"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Manage index sources…"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("https://fixture.invalid/index.json") || !tt.HasText("Add source") {
		t.Fatal("configured source editor unavailable")
	}
	tt.SetSize(520, 760)
	savePageSnapshot(t, tt, "plugins-discover-sources-narrow.png")
}

func TestManagerDisconnectedPageNeverShowsCuratedCatalog(t *testing.T) {
	m := Model{Controller: manager.New()}
	tt := ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { View(c, m) }) }, 800, 600)
	if !tt.HasText(copyText("运行时未连接", "Runtime disconnected")) ||
		!tt.HasText(copyText("连接后即可管理此 profile 的插件。", "Connect to manage this profile's plugins.")) {
		t.Fatal("disconnected manager state was hidden")
	}
	if tt.HasText("Catalog source:") {
		t.Fatal("manager fell back to desktop marketplace")
	}
	savePageSnapshot(t, tt, "plugins-disconnected.png")
}

func savePageSnapshot(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	dir := surfaceGalleryDir(os.Getenv("SECONDARY_SNAPSHOT_DIR"))
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, tt.Image()); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

// surfaceGalleryDir routes a surface snapshot to the evidence gallery when
// SURFACES_GALLERY_DIR is set, so the same test writes documentation images
// without a second harness.
func surfaceGalleryDir(fallback string) string {
	if dir := os.Getenv("SURFACES_GALLERY_DIR"); dir != "" {
		return dir
	}
	return fallback
}

// saveSurfaceSnapshot writes the rendered surface for the evidence gallery when
// SURFACES_GALLERY_DIR is set.
func saveSurfaceSnapshot(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	dir := os.Getenv("SURFACES_GALLERY_DIR")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, tt.Image()); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestPluginPageDiscoverTabIsReachable(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	m := pageModel(t)
	owner := m.Controller.Snapshot().Value.Owner
	// The tab row mounts only once the market service answered — the native
	// stand-in for the original's plugins.tab slot contribution.
	if err := m.Controller.LoadIndexes(context.Background(), owner); err != nil {
		t.Fatal(err)
	}
	tt := ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { View(c, m) }) }, 1200, 900)
	if !tt.HasText("Discover") {
		t.Fatalf("the discover tab is unreachable: %v", tt.Texts())
	}
	if !tt.HasText("Installed") {
		t.Fatalf("the installed tab went missing: %v", tt.Texts())
	}
	// The visible control reaches the existing Host catalog controller.
	if err := tt.Click("Discover"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if m.Controller.Snapshot().Value.Tab != "discover" {
		t.Fatal("Discover click did not select the native discovery view")
	}
}

func TestAddMenuOffersTheOfficialMarketplace(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	m := pageModel(t)
	opened := []string{}
	m.OnOpenURL = func(url string) { opened = append(opened, url) }
	tt := ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { View(c, m) }) }, 1200, 900)
	if err := tt.Click("Choose add method"); err != nil {
		t.Fatalf("add menu trigger missing: %v", err)
	}
	if !tt.HasText("Plugin marketplace") {
		t.Fatalf("marketplace entry missing: %v", tt.Texts())
	}
	if !tt.HasText("Browse plugins in the official dsh marketplace (awesome-dsh-plugin.com)") {
		t.Fatalf("marketplace description missing: %v", tt.Texts())
	}
	if !tt.HasText("Install existing plugin") {
		t.Fatal("install entry disappeared")
	}
	before := m.Controller.Snapshot().Value
	if err := tt.Click("Plugin marketplace"); err != nil {
		t.Fatal(err)
	}
	if len(opened) != 1 || opened[0] != "https://awesome-dsh-plugin.com" {
		t.Fatalf("marketplace opened %v", opened)
	}
	after := m.Controller.Snapshot().Value
	if after.Tab != before.Tab || after.View.Kind != before.View.Kind {
		t.Fatalf("marketplace entry changed the page: %+v -> %+v", before.View, after.View)
	}
}

func TestAddMenuMarketplaceIsDisabledWithoutABrowser(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	m := pageModel(t)
	tt := ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { View(c, m) }) }, 1200, 900)
	if err := tt.Click("Choose add method"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Plugin marketplace") {
		t.Fatalf("disabled entry should still be listed: %v", tt.Texts())
	}
	if err := tt.Click("Plugin marketplace"); err != nil {
		t.Fatal(err)
	}
	// Nothing to call and nothing to change.
	if snap := m.Controller.Snapshot().Value; snap.View.Kind != "" && snap.View.Kind != "list" {
		t.Fatalf("disabled marketplace entry navigated: %+v", snap.View)
	}
}

type onDemandSubagentFixture struct {
	pageFixture
	installed []string
}

func (f *onDemandSubagentFixture) Call(ctx context.Context, method string, args, out any) error {
	switch method {
	case "pluginManager/listBundles":
		raw, _ := json.Marshal([]manager.Bundle{
			{Name: "@deepseek-ai/dsh-base", Version: "0.2.2-rc.5", Availability: "installation", Enabled: true},
			{
				Name: "@deepseek-ai/dsh-subagent-codex", Official: true, Availability: "missing",
				Meta: &manager.Meta{
					Title:       manager.Text(`{"en":"Codex subagent","zh":"Codex 子智能体"}`),
					Description: manager.Text(`{"en":"Delegate one task to Codex using its native authentication and settings.","zh":"使用 Codex 原生认证和设置委派单个任务。"}`),
				},
				Error: &manager.ManagementError{
					Code:       "operation-error",
					Diagnostic: `Official bundle @deepseek-ai/dsh-subagent-codex: expected one workspace package in C:\runtime\vendor\deepseek-harness, found 0; restore the checkout and run pnpm install`,
				},
			},
			{
				Name: "@deepseek-ai/dsh-subagent-claude-code", Official: true, Availability: "missing",
				InstallTarget: &manager.InstallTarget{Spec: "@deepseek-ai/dsh-subagent-claude-code@0.2.2-rc.5", Version: "0.2.2-rc.5"},
				Meta: &manager.Meta{
					Title:       manager.Text(`{"en":"Claude Code subagent","zh":"Claude Code 子智能体"}`),
					Description: manager.Text(`{"en":"Delegate one task to Claude Code using its native authentication and settings.","zh":"使用 Claude Code 原生认证和设置委派单个任务。"}`),
				},
			},
		})
		return json.Unmarshal(raw, out)
	case "pluginManager/installBundle":
		raw, _ := json.Marshal(args)
		var req struct {
			Spec string `json:"spec"`
		}
		_ = json.Unmarshal(raw, &req)
		f.installed = append(f.installed, req.Spec)
		resp, _ := json.Marshal(manager.Change{Application: "applied", Changed: true, Target: req.Spec})
		return json.Unmarshal(resp, out)
	default:
		return f.pageFixture.Call(ctx, method, args, out)
	}
}

func TestOnDemandOfficialSubagentsShowNotInstalledWithoutProblemAndRemainToggleable(t *testing.T) {
	for _, tc := range []struct {
		name           string
		locale         string
		moreBtn        string
		wantTitle      string
		wantBadge      string
		wantSource     string
		forbidProblem  string
		forbidCombined string
	}{
		{
			name:           "zh",
			locale:         i18n.LocaleZH,
			moreBtn:        "更多",
			wantTitle:      "Codex 子智能体",
			wantBadge:      "未安装",
			wantSource:     "官方 · 按需安装",
			forbidProblem:  "异常",
			forbidCombined: "未安装 / 包不可用",
		},
		{
			name:           "en",
			locale:         i18n.LocaleEN,
			moreBtn:        "More",
			wantTitle:      "Codex subagent",
			wantBadge:      "Not installed",
			wantSource:     "Official · On-demand",
			forbidProblem:  "Problem",
			forbidCombined: "Not installed / missing files",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			i18n.SetLocale(tc.locale)
			t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
			f := &onDemandSubagentFixture{}
			c := manager.New()
			c.Attach(f, nil)
			t.Cleanup(c.Detach)
			if err := c.Load(context.Background()); err != nil {
				t.Fatal(err)
			}
			tt := ui.NewTester(func(ctx *ui.Context) { ui.Column(ctx).Fill().Children(func() { View(ctx, Model{Controller: c}) }) }, 860, 760)
			if err := tt.Click(tc.moreBtn); err != nil {
				t.Fatalf("open more: %v", err)
			}
			if !tt.HasText(tc.wantTitle) {
				t.Fatalf("missing subagent row %q in %v", tc.wantTitle, tt.Texts())
			}
			if !tt.HasText(tc.wantBadge) {
				t.Fatalf("missing badge %q in %v", tc.wantBadge, tt.Texts())
			}
			if !tt.HasText(tc.wantSource) {
				t.Fatalf("missing source label %q in %v", tc.wantSource, tt.Texts())
			}
			if tt.HasText(tc.forbidProblem) {
				t.Fatalf("unexpected problem badge %q rendered for healthy on-demand subagent: %v", tc.forbidProblem, tt.Texts())
			}
			if tt.HasText(tc.forbidCombined) {
				t.Fatalf("unexpected combined unavailable badge %q rendered: %v", tc.forbidCombined, tt.Texts())
			}
			// Verify the switch is unlocked and clicking it opens the on-demand install flow.
			switchLabel := "Enable @deepseek-ai/dsh-subagent-codex"
			if tc.locale == i18n.LocaleZH {
				switchLabel = "启用 @deepseek-ai/dsh-subagent-codex"
			}
			if err := tt.Click(switchLabel); err != nil {
				t.Fatalf("click subagent switch %q: %v", switchLabel, err)
			}
			if !c.Snapshot().Value.Install.Open || c.Snapshot().Value.Install.Spec != "@deepseek-ai/dsh-subagent-codex@0.2.2-rc.5" {
				t.Fatalf("switch did not open on-demand catalog install: %+v", c.Snapshot().Value.Install)
			}
		})
	}
}

func TestInstalledBundleWithMissingFilesStillShowsUnavailableAndProblemBadges(t *testing.T) {
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	c := manager.New()
	c.Attach(testCallFunc(func(ctx context.Context, method string, args, out any) error {
		if method == "pluginManager/listBundles" {
			raw, _ := json.Marshal([]manager.Bundle{
				{
					Name: "broken-local-plugin", Availability: "missing", Installed: true, Removable: true,
					Meta:  &manager.Meta{Title: manager.Text(`"损坏的插件"`)},
					Error: &manager.ManagementError{Code: "not-bundle"},
				},
			})
			return json.Unmarshal(raw, out)
		}
		return pageFixture{}.Call(ctx, method, args, out)
	}), nil)
	t.Cleanup(c.Detach)
	if err := c.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	tt := ui.NewTester(func(ctx *ui.Context) { ui.Column(ctx).Fill().Children(func() { View(ctx, Model{Controller: c}) }) }, 860, 760)
	if !tt.HasText("异常") || !tt.HasText("包不可用") {
		t.Fatalf("installed bundle with missing files must show 异常 and 包不可用: %v", tt.Texts())
	}
}

type testCallFunc func(context.Context, string, any, any) error

func (f testCallFunc) Call(ctx context.Context, method string, args, out any) error {
	return f(ctx, method, args, out)
}
