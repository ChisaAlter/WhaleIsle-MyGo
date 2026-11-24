package settings

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/pet"
	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"
)

func fullRailModel() Model {
	return Model{
		DesktopActive: true, Desktop: &DesktopAccess{},
		Appearance: &AppearanceAccess{}, Interface: &InterfaceAccess{}, Conversation: &ConversationAccess{},
		Models: &ModelsAccess{}, AgentPresets: &AgentPresetsAccess{}, Skills: &SkillsAccess{}, MCP: &MCPAccess{}, Plugins: &PluginsAccess{},
		Features: &FeaturesAccess{}, Pet: new(pet.Companion), About: &AboutAccess{},
		OnOpenMarket: func() {}, OnOpenRemote: func() {}, OnOpenUsage: func() {}, OnSelectVoice: func() {}, OnSelectDeepSeekAPIs: func() {},
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
		{"功能", []SectionID{SectionFeatures, SectionVoice, SectionDeepSeekAPIs, SectionPet, SectionRemote}},
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

// TestSettingsRailCellsShareAUniformGrid pins the rail's coss layout contract:
// every nav cell sits on the same column with the same box, stacked in group
// order above the options pane, which starts right of the rail.
func TestSettingsRailCellsShareAUniformGrid(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(previous) })
	m := fullRailModel()
	m.ActiveSection, m.SectionContent = SectionGeneral, func(c *ui.Context) { ui.Box(c).Label("origin").Size(8, 8) }
	tt := ui.NewTester(func(c *ui.Context) { ModalView(c, m) }, 1440, 864)
	var prev ui.Rect
	for i, label := range []string{"用量与费用", "通用设置", "外观", "对话", "模型"} {
		got, found := tt.Find(label)
		if !found || got.W <= 0 || got.H <= 0 {
			t.Fatalf("%s actual=%+v found=%v", label, got, found)
		}
		if i > 0 {
			if got.X != prev.X || got.W != prev.W || got.H != prev.H {
				t.Fatalf("%s off the rail grid: %+v vs %+v", label, got, prev)
			}
			if got.Y <= prev.Y {
				t.Fatalf("%s not stacked below %v: %+v", label, prev, got)
			}
		}
		prev = got
	}
	origin, found := tt.Find("origin")
	if !found || origin.X <= prev.X+prev.W {
		t.Fatalf("options pane is not right of the rail: %+v vs %+v", origin, prev)
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

// TestSettingsPanelPaintsThePopoverSurface pins the modal's surface contract:
// the panel reads coss's raised popover token in both schemes, floating over
// the black/32 dimming backdrop.
func TestSettingsPanelPaintsThePopoverSurface(t *testing.T) {
	bg := ui.RGB(8, 96, 8)
	var popover ui.Color
	m := fullRailModel()
	m.ActiveSection, m.SectionContent = SectionGeneral, func(c *ui.Context) {}
	tt := ui.NewTester(func(c *ui.Context) {
		popover = tk(c).Popover
		ui.Column(c).Fill().Background(bg)
		ModalView(c, m)
	}, 1440, 864)
	settle := func() {
		t0 := time.Now()
		tt.SetTime(t0)
		tt.Frame()
		tt.SetTime(t0.Add(time.Second))
		tt.Frame()
	}
	near := func(px, want uint8) bool { d := int(px) - int(want); return d > -4 && d < 4 }
	settle()
	panel := ui.Rect{}
	for _, r := range tt.FindAll(i18n.T(i18n.KeySettingsTitle)) {
		if r.W*r.H > panel.W*panel.H {
			panel = r
		}
	}
	if panel.W == 0 {
		t.Fatalf("settings panel not found: %v", tt.Texts())
	}
	for _, dark := range []bool{false, true} {
		tt.SetDark(dark)
		settle()
		px := tt.Image().RGBAAt(int(panel.X+panel.W)-24, int(panel.Y+panel.H/2))
		if !near(px.R, popover.R) || !near(px.G, popover.G) || !near(px.B, popover.B) {
			t.Fatalf("dark%v panel paints %v, want popover %v", dark, px, popover)
		}
		mask := tt.Image().RGBAAt(4, 4)
		if !near(mask.G, uint8(float32(bg.G)*0.68+0.5)) || mask.G == popover.G {
			t.Fatalf("dark%v backdrop is %v, want black/32 over %v", dark, mask, bg)
		}
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
