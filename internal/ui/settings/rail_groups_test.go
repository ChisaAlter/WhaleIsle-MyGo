package settings

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/pet"
	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func fullRailModel() Model {
	return Model{
		DesktopActive: true, Desktop: &DesktopAccess{},
		Appearance: &AppearanceAccess{}, Interface: &InterfaceAccess{}, Conversation: &ConversationAccess{},
		Models: &ModelsAccess{}, AgentPresets: &AgentPresetsAccess{}, Skills: &SkillsAccess{}, MCP: &MCPAccess{}, Plugins: &PluginsAccess{},
		Features: &FeaturesAccess{}, Pet: new(pet.Companion), About: &AboutAccess{},
		OnOpenMarket: func() {}, OnOpenRemote: func() {}, OnOpenUsage: func() {},
		OnClose: func() {},
	}
}

// The installed 0.3.13 rail is grouped, in this order, under these headers.
func TestSettingsRailIsGroupedLikeTheInstalledBuild(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(previous) })
	want := []struct {
		label string
		ids   []SectionID
	}{
		{"账户", []SectionID{SectionUsageStats}},
		{"通用", []SectionID{SectionGeneral, SectionAppearance, SectionInterface, SectionConversation}},
		{"智能体", []SectionID{SectionModels, SectionAgentPresets, SectionSkills, SectionMCP, SectionPlugins, SectionMarket}},
		{"功能", []SectionID{SectionFeatures, SectionPet, SectionRemote}},
		{"其他", []SectionID{SectionAbout}},
	}
	groups := navigationGroups(fullRailModel(), store.ViewState{})
	if len(groups) != len(want) {
		t.Fatalf("groups = %d, want %d: %+v", len(groups), len(want), groups)
	}
	for i, group := range groups {
		if group.label != want[i].label || len(group.entries) != len(want[i].ids) {
			t.Fatalf("group %d = %q/%d entries, want %q/%d", i, group.label, len(group.entries), want[i].label, len(want[i].ids))
		}
		for j, entry := range group.entries {
			if entry.id != want[i].ids[j] {
				t.Fatalf("group %q entry %d = %s, want %s", group.label, j, entry.id, want[i].ids[j])
			}
		}
	}
	labels := map[SectionID]string{SectionRemote: "手机远程", SectionUsageStats: "用量与费用", SectionConversation: "对话", SectionFeatures: "内置功能"}
	for id, label := range labels {
		if got := sectionLabel(id); got != label {
			t.Errorf("label %s = %q, want %q", id, got, label)
		}
	}
}

func TestSettingsRailGeometryMatchesTheInstalledBuild(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(previous) })
	m := fullRailModel()
	m.ActiveSection, m.SectionContent = SectionGeneral, func(c *ui.Context) { ui.Box(c).Label("origin").Size(8, 8) }
	tt := ui.NewTester(func(c *ui.Context) { ModalView(c, m) }, 1440, 864)
	for _, expected := range []struct {
		label               string
		x, y, width, height float32
	}{
		// panel (320,32): search 164x32 at +12,+64; first group label 20 DIP at +114;
		// cells 36 DIP high, 2 DIP apart; groups 12 DIP apart.
		{"用量与费用", 332, 168, 164, 36},
		{"通用设置", 332, 238, 164, 36},
		{"外观", 332, 276, 164, 36},
		{"对话", 332, 352, 164, 36},
		{"模型", 332, 422, 164, 36},
	} {
		got, found := tt.Find(expected.label)
		if !found || got.X != expected.x || got.Y != expected.y || got.W != expected.width || got.H != expected.height {
			t.Fatalf("%s actual=%+v found=%v want=(%g,%g,%g,%g)", expected.label, got, found, expected.x, expected.y, expected.width, expected.height)
		}
	}
}

func TestSettingsSearchListsRowsAndTheEmptyState(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(previous) })
	m := fullRailModel()
	m.SectionContent, m.ActiveSection = func(c *ui.Context) { ui.Text(c, "page body") }, SectionGeneral
	tt := ui.NewTester(func(c *ui.Context) { ModalView(c, m) }, 1440, 864)
	if !tt.HasText("page body") {
		t.Fatalf("page not painted before searching: %v", tt.Texts())
	}
	if err := tt.Click("搜索设置"); err != nil {
		t.Fatal(err)
	}
	tt.Type("字号")
	if tt.HasText("page body") {
		t.Fatal("results did not replace the page")
	}
	for _, want := range []string{"字号大小", "代码字号", "外观 · 16px", "外观 · 仅影响会话内容的字号", "外观 · 13px"} {
		if !tt.HasText(want) {
			t.Fatalf("search for 字号 lacks %q: %v", want, tt.Texts())
		}
	}
	// The rail keeps every group while searching, and the clear button empties the box.
	if !tt.HasText("智能体") || !tt.HasText("手机远程") {
		t.Fatalf("rail was filtered: %v", tt.Texts())
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("zzzz")
	if !tt.HasText("没有找到「zzzz」相关的设置") {
		t.Fatalf("empty state missing: %v", tt.Texts())
	}
	if err := tt.Click(i18n.T(i18n.KeySettingsNavSearchClear)); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("page body") {
		t.Fatalf("clearing the search did not bring the page back: %v", tt.Texts())
	}
}

func TestSettingsSearchResultOpensItsSection(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(previous) })
	opened := 0
	m := fullRailModel()
	m.Appearance = &AppearanceAccess{}
	m.OnSelectAppearance = func() { opened++ }
	tt := ui.NewTester(func(c *ui.Context) { ModalView(c, m) }, 1440, 864)
	if err := tt.Click("搜索设置"); err != nil {
		t.Fatal(err)
	}
	tt.Type("代码字号")
	if err := tt.Click("代码字号"); err != nil {
		t.Fatal(err)
	}
	if opened != 1 {
		t.Fatalf("result opened the appearance page %d times, want 1", opened)
	}
}

func TestSettingsPanelBaseIsTheRaisedLayer(t *testing.T) {
	light := theme.Resolve(theme.IDDeepseekLight).Active(false)
	dark := theme.Resolve(theme.IDDeepseekDark).Active(true)
	if got := settingsPanelBase(light, false); got != ui.RGB(255, 255, 255) {
		t.Fatalf("light panel base = %v, want white", got)
	}
	if got := settingsPanelBase(dark, true); got != theme.Color("#232324") {
		t.Fatalf("dark panel base = %v, want rgb(35, 35, 36)", got)
	}
}

func TestFeaturesPageHasTheThreeInstalledRows(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(previous) })
	assistant, remote := false, false
	var set []string
	access := &FeaturesAccess{
		AssistantEnabled: &assistant, OnSetAssistant: func(v bool) { set = append(set, "assistant") },
		RemoteWorkspaceEnabled: &remote, OnSetRemoteWorkspace: func(v bool) { set = append(set, "remote") },
		Project: &ProjectPreferenceAccess{OnSet: func(v bool) { set = append(set, "project") }},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(24).Children(func() { featuresPane(c, Model{Features: access}, c.Theme().Text, c.Theme().TextMuted) })
	}, 700, 700)
	last := -1.0
	for _, title := range []string{"鲸鱼娘助理", "Project", "远程工作区（SSH）"} {
		rect, ok := tt.Find(title)
		if !ok || float64(rect.Y) <= last {
			t.Fatalf("%s missing or out of order: %+v ok=%v texts=%v", title, rect, ok, tt.Texts())
		}
		last = float64(rect.Y)
	}
	hints := 0
	for _, text := range tt.Texts() {
		if text == "切换后会自动重启 Harness。" {
			hints++
		}
	}
	if hints != 2 {
		t.Fatalf("restart hint appears %d times, want 2 (Project, SSH): %v", hints, tt.Texts())
	}
	// Tab reaches the three installed switches in page order; Space flips each one
	// and reports it through its own setter.
	// (the Project help button sits between the first two switches)
	for range 4 {
		tt.Key(0, ui.KeyTab)
		tt.Key(0, ui.KeySpace)
	}
	if strings.Join(set, ",") != "assistant,project,remote" {
		t.Fatalf("setters = %v, want assistant,project,remote", set)
	}
}
