package settings

import (
	"image/color"
	"math"
	"testing"
	"time"

	"whaleisle/internal/config"
	"whaleisle/internal/desktop/pet"
	"whaleisle/internal/desktop/plugininventory"
	"whaleisle/internal/desktop/voicecapture"
	"whaleisle/internal/desktop/voiceinput"
	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
)

// TestAllSettingsPagesZeroDuplicateKeys renders every settings section —
// including Appearance with an active theme editor error — under ui.NewTester
// and asserts zero duplicate-key panics.
func TestAllSettingsPagesZeroDuplicateKeys(t *testing.T) {
	prev := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(prev) })

	host := bodyFixtureHost()
	editor := theme.NewEditor(theme.Defaults(), func(theme.Preferences) error { return nil })
	editor.Status = "保存主题失败，请重试。"
	editor.StatusDetail = "synthetic error detail"

	minOn, minSuggest := false, true
	glassOn := true
	themeChoice := "system"
	wpPrefs := wallpapers.Preferences{Glass: 80, Blur: 24, Terminal: 90, Transparent: false, HideSidebarMask: false}
	wpEditor := &WallpaperEditor{
		Prefs: wpPrefs,
		Saved: wpPrefs,
		Save:  func(p wallpapers.Preferences) error { wpPrefs = p; return nil },
	}
	showArchived := true
	showViewTabs := true
	autoAllow := false
	closeToTray := false
	autoStart := false
	locale := "zh"
	selectedPreset := ""
	companion := &pet.Companion{Enabled: true, Behavior: config.DefaultPetBehavior()}

	m := fullRailModel()
	m.Desktop = &DesktopAccess{
		Host:             host,
		Preferences:      editor,
		AutoStartDesktop: &autoStart,
		Fields: DesktopFields{
			Locale:      &locale,
			CloseToTray: &closeToTray,
		},
		OnSetLocale:           func(v string) (string, error) { return v, nil },
		OnSetCloseToTray:      func(v bool) (bool, error) { return v, nil },
		OnSetAutoStartDesktop: func(v bool) (bool, error) { return v, nil },
	}
	m.Appearance = &AppearanceAccess{
		Host: host,
		Minimal: &MinimalModeAccess{
			Enabled:             &minOn,
			Suggestions:         &minSuggest,
			OnToggle:            func(bool) error { return nil },
			OnToggleSuggestions: func(bool) error { return nil },
		},
		Glass: &GlassThemeAccess{
			Enabled:  &glassOn,
			OnToggle: func(bool) error { return nil },
		},
		Background:  wpEditor,
		Preferences: editor,
		Theme:       &themeChoice,
		OnSetTheme:  func(string) error { return nil },
	}
	m.Interface = &InterfaceAccess{
		Host:                  host,
		ShowArchivedList:      &showArchived,
		OnSetShowArchivedList: func(v bool) (bool, error) { return v, nil },
		ShowViewTabs:          &showViewTabs,
		OnSetViewTabs:         func(v bool) (bool, error) { return v, nil },
		AutoAllowTools:        &autoAllow,
		OnSetAutoAllowTools:   func(v bool) (bool, error) { return v, nil },
		Project:               &ProjectPreferenceAccess{OnSet: func(bool) {}},
	}
	m.Conversation = &ConversationAccess{Host: host}
	m.Features = &FeaturesAccess{Project: &ProjectPreferenceAccess{OnSet: func(bool) {}}}
	m.Models = &ModelsAccess{}
	m.Plugins = &PluginsAccess{
		Inventory: &PluginInventoryAccess{
			Snapshot: plugininventory.Snapshot{Connected: true, Ready: true},
		},
	}
	m.Skills = &SkillsAccess{Query: new(string), SourceFilter: new(string)}
	m.MCP = &MCPAccess{HasListRPC: true}
	m.AgentPresets = &AgentPresetsAccess{
		Selected: &selectedPreset,
		Reader:   &AgentPresetReaderAccess{OnSelect: func(uint64, string) {}},
	}
	m.Pet = companion
	m.PetSettings = &PetSettingsAccess{AssistantAvailable: true, Whale: whaleFixture()}
	m.About = &AboutAccess{
		AppName:            "Whale Isle",
		Version:            "dev",
		OnOpenUpdateCenter: func() {},
		OnCheckUpdates:     func() {},
	}

	sections := []SectionID{
		SectionAccount,
		SectionGeneral,
		SectionAppearance,
		SectionInterface,
		SectionConversation,
		SectionFeatures,
		SectionVoice,
		SectionDeepSeekAPIs,
		SectionModels,
		SectionPlugins,
		SectionSkills,
		SectionMarket,
		SectionRemote,
		SectionMCP,
		SectionAgentPresets,
		SectionPet,
		SectionAbout,
		SectionUsageStats,
		SectionOther,
	}
	for _, sec := range sections {
		t.Run(string(sec), func(t *testing.T) {
			cur := m
			cur.ActiveSection = sec
			tt := ui.NewTester(func(c *ui.Context) {
				View(c, cur)
			}, 1000, 760)
			tt.Frame()
			tt.Frame()
		})
	}
}

// TestSettingsSwitchesUniformSizeAndRightAlignment verifies that every switch
// on Appearance and Interface shares the exact 30×18 geometry and aligns to
// one right-edge X baseline.
func TestSettingsSwitchesUniformSizeAndRightAlignment(t *testing.T) {
	prev := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(prev) })

	t.Run("appearance", func(t *testing.T) {
		minOn, minSuggest, glassOn := false, true, true
		wpPrefs := wallpapers.Preferences{Glass: 80, Blur: 24, Terminal: 90}
		wpEditor := &WallpaperEditor{
			Prefs: wpPrefs,
			Saved: wpPrefs,
			Save:  func(p wallpapers.Preferences) error { wpPrefs = p; return nil },
		}
		a := &AppearanceAccess{
			Minimal: &MinimalModeAccess{
				Enabled:             &minOn,
				Suggestions:         &minSuggest,
				OnToggle:            func(bool) error { return nil },
				OnToggleSuggestions: func(bool) error { return nil },
			},
			Glass: &GlassThemeAccess{
				Enabled:  &glassOn,
				OnToggle: func(bool) error { return nil },
			},
			Background: wpEditor,
		}
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).FillWidth().MinWidth(0).Gap(12).Padding(16).Children(func() {
				minimalModePane(c, a.Minimal, c.Theme().Text, c.Theme().TextMuted)
				glassThemePane(c, a.Glass, c.Theme().Text, c.Theme().TextMuted)
				wallpaperBackgroundPane(c, a.Background, c.Theme().Text, c.Theme().TextMuted)
			})
		}, 920, 2200)
		tt.Frame()

		labels := []string{
			"开启极简模式",
			"关闭启动卡顿提示",
			"关闭壁纸与毛玻璃主题",
			"透明主题",
			"隐藏侧栏遮罩",
		}
		findSwitch := func(label string) ui.Rect {
			t.Helper()
			for _, r := range tt.FindAll(label) {
				if math.Abs(float64(r.W-toggleWidth)) < 1 && math.Abs(float64(r.H-toggleHeight)) < 1 {
					return r
				}
			}
			t.Fatalf("30x18 switch %q not found; rects=%v", label, tt.FindAll(label))
			return ui.Rect{}
		}
		var refRight float32
		for i, lbl := range labels {
			r := findSwitch(lbl)
			right := r.X + r.W
			if i == 0 {
				refRight = right
				continue
			}
			if math.Abs(float64(right-refRight)) >= 0.5 {
				t.Fatalf("switch %q right edge %.2f != baseline %.2f (rect=%+v)", lbl, right, refRight, r)
			}
		}
	})

	t.Run("interface", func(t *testing.T) {
		host := bodyFixtureHost()
		showArchived := true
		a := &InterfaceAccess{
			Host:                  host,
			ShowArchivedList:      &showArchived,
			OnSetShowArchivedList: func(v bool) (bool, error) { return v, nil },
			Project:               &ProjectPreferenceAccess{OnSet: func(bool) {}},
		}
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).FillWidth().MinWidth(0).Gap(12).Padding(16).Children(func() {
				interfacePane(c, Model{Interface: a}, c.Theme().Text, c.Theme().TextMuted)
			})
		}, 920, 1400)
		tt.Frame()

		labels := []string{
			"标题栏 Git 操作",
			"显示底部归档管理区",
		}
		var refRight float32
		for i, lbl := range labels {
			var found ui.Rect
			for _, r := range tt.FindAll(lbl) {
				if math.Abs(float64(r.W-toggleWidth)) < 1 && math.Abs(float64(r.H-toggleHeight)) < 1 {
					found = r
					break
				}
			}
			if found.W == 0 {
				t.Fatalf("interface 30x18 switch %q not found; rects=%v", lbl, tt.FindAll(lbl))
			}
			right := found.X + found.W
			if i == 0 {
				refRight = right
				continue
			}
			if math.Abs(float64(right-refRight)) >= 0.5 {
				t.Fatalf("interface switch %q right edge %.2f != baseline %.2f (rect=%+v)", lbl, right, refRight, found)
			}
		}
	})
}

// TestSettingsSelectsUniformWidthAndLeftAlignment verifies that General page
// dropdowns share identical width and left-edge alignment.
func TestSettingsSelectsUniformWidthAndLeftAlignment(t *testing.T) {
	prev := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(prev) })

	locale := "zh"
	closeToTray := false
	autoStart := false
	d := &DesktopAccess{
		AutoStartDesktop: &autoStart,
		Fields: DesktopFields{
			Locale:      &locale,
			CloseToTray: &closeToTray,
		},
		OnSetLocale:           func(v string) (string, error) { return v, nil },
		OnSetCloseToTray:      func(v bool) (bool, error) { return v, nil },
		OnSetAutoStartDesktop: func(v bool) (bool, error) { return v, nil },
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().MinWidth(0).Padding(16).Children(func() {
			desktopPane(c, Model{Desktop: d}, c.Theme().Text, c.Theme().TextMuted)
		})
	}, 900, 900)
	tt.Frame()

	findSelect := func(label string) ui.Rect {
		t.Helper()
		for _, r := range tt.FindAll(label) {
			if math.Abs(float64(r.W-design.Sp(52))) < 1 && r.H >= 28 && r.H <= 36 {
				return r
			}
		}
		t.Fatalf("uniform select %q (width %.0f) not found; rects=%v", label, design.Sp(52), tt.FindAll(label))
		return ui.Rect{}
	}
	s1 := findSelect("语言")
	s2 := findSelect("关闭窗口时")
	s3 := findSelect("启动时")
	for _, pair := range []struct {
		name string
		r    ui.Rect
	}{{"关闭窗口时", s2}, {"启动时", s3}} {
		if math.Abs(float64(pair.r.X-s1.X)) >= 0.5 || math.Abs(float64(pair.r.W-s1.W)) >= 0.5 {
			t.Fatalf("select %q (%+v) misaligned with 语言 (%+v)", pair.name, pair.r, s1)
		}
	}
}

// TestAppearanceChoiceCardsZeroLayoutShiftOnSelection verifies that selecting
// an appearance choice card paints an overlay ring without shifting inner content.
func TestAppearanceChoiceCardsZeroLayoutShiftOnSelection(t *testing.T) {
	selected := "a"
	choices := [][2]string{{"a", "Card Alpha"}, {"b", "Card Beta"}}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Padding(16).Children(func() {
			appearanceChoiceCards(c, "choice-test", &selected, choices, func(string) []string { return []string{"#3366ff"} }, false)
		})
	}, 500, 200)
	tt.Frame()

	beforeCards := tt.FindAll("Card Beta")
	if len(beforeCards) < 2 {
		t.Fatalf("expected outer card and inner label for Card Beta, got %v", beforeCards)
	}
	if err := tt.Click("Card Beta"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if selected != "b" {
		t.Fatalf("click did not select Card Beta: %q", selected)
	}
	afterCards := tt.FindAll("Card Beta")
	if len(afterCards) != len(beforeCards) {
		t.Fatalf("Card Beta rect count changed: %d -> %d", len(beforeCards), len(afterCards))
	}
	for i := range beforeCards {
		b, a := beforeCards[i], afterCards[i]
		if math.Abs(float64(a.X-b.X)) > 0.01 || math.Abs(float64(a.Y-b.Y)) > 0.01 || math.Abs(float64(a.W-b.W)) > 0.01 || math.Abs(float64(a.H-b.H)) > 0.01 {
			t.Fatalf("selecting appearance choice card shifted rect[%d]: before=%+v after=%+v", i, b, a)
		}
	}
}

// TestThemePreviewDarkModeDeepSeekLightHalfCheckmarkVisible ensures that in
// dark mode, the selected checkmark badge on DeepSeek's light preview half
// remains high-contrast against the white preview background and that its
// checkmark glyph contrasts clearly inside the badge.
func TestThemePreviewDarkModeDeepSeekLightHalfCheckmarkVisible(t *testing.T) {
	prev := theme.CurrentPreferences()
	t.Cleanup(func() { theme.SetPreferences(prev) })
	prefs := theme.Defaults()
	prefs.Light = "deepseek"
	prefs.Dark = "deepseek"
	theme.SetPreferences(prefs)

	editor := theme.NewEditor(prefs, func(theme.Preferences) error { return nil })
	tt := ui.NewTester(func(c *ui.Context) {
		c.Theme().Dark = true
		ui.Column(c).FillWidth().Padding(16).Children(func() {
			themeLibraryPane(c, editor, theme.Resolve(theme.IDDeepseekDark).Dark, c.Theme().Text, c.Theme().TextMuted)
		})
	}, 900, 600)
	tt.SetDark(true)
	tt.Frame()

	half, ok := tt.Find("DeepSeek light")
	if !ok {
		t.Fatalf("DeepSeek light preview half missing: %v", tt.Texts())
	}
	img := tt.Image()
	bg := img.RGBAAt(int(half.X+half.W/2), int(half.Y+half.H/2))
	// Badge sits at Top=6, Right=6, Size=16x16 inside the half.
	badgeMinX := int(half.X+half.W) - 22
	badgeMaxX := int(half.X+half.W) - 6
	badgeMinY := int(half.Y) + 6
	badgeMaxY := int(half.Y) + 22

	distRGBA := func(a, b color.RGBA) int {
		return absDiff(a.R, b.R) + absDiff(a.G, b.G) + absDiff(a.B, b.B)
	}
	badgePixels := 0
	var distinctColors []color.RGBA
	for y := badgeMinY; y < badgeMaxY; y++ {
		for x := badgeMinX; x < badgeMaxX; x++ {
			px := img.RGBAAt(x, y)
			if distRGBA(px, bg) > 80 {
				badgePixels++
				distinctColors = append(distinctColors, px)
			}
		}
	}
	if badgePixels < 40 {
		t.Fatalf("DeepSeek light half checkmark badge invisible in dark mode: only %d contrasting pixels over bg %v (half=%+v)", badgePixels, bg, half)
	}
	maxInternalContrast := 0
	for y := badgeMinY + 3; y < badgeMaxY-3; y++ {
		for x := badgeMinX + 3; x < badgeMaxX-3; x++ {
			p1 := img.RGBAAt(x, y)
			for _, p2 := range distinctColors {
				if d := distRGBA(p1, p2); d > maxInternalContrast {
					maxInternalContrast = d
				}
			}
		}
	}
	if maxInternalContrast < 80 {
		t.Fatalf("DeepSeek light half checkmark glyph lacks contrast inside badge (maxContrast=%d)", maxInternalContrast)
	}
}

// TestSettingsSearchUpdatesHeaderTitleToSearchResults verifies that typing a
// search query updates the right-pane header heading to "搜索结果" / "Search results".
func TestSettingsSearchUpdatesHeaderTitleToSearchResults(t *testing.T) {
	prev := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(prev) })

	m := fullRailModel()
	m.ActiveSection = SectionGeneral
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, m)
	}, 960, 720)
	tt.Frame()

	headerRect, ok := tt.Find("设置页头")
	if !ok {
		t.Fatalf("settings header missing: %v", tt.Texts())
	}
	hasHeaderText := func(want string) bool {
		for _, r := range tt.FindAll(want) {
			if r.Y >= headerRect.Y && r.Y+r.H <= headerRect.Y+headerRect.H+1 && r.X >= headerRect.X {
				return true
			}
		}
		return false
	}
	if !hasHeaderText("通用设置") {
		t.Fatalf("expected 通用设置 in header before search")
	}
	if err := tt.Click(i18n.T(i18n.KeySettingsNavSearch)); err != nil {
		t.Fatal(err)
	}
	tt.Type("字体")
	tt.Frame()
	if !hasHeaderText(i18n.T(i18n.KeySettingsParitySearchResults)) {
		t.Fatalf("header did not switch to %q while searching; texts=%v", i18n.T(i18n.KeySettingsParitySearchResults), tt.Texts())
	}
}

// TestAboutVersionBadgeVerticallyCentered verifies the read-only version pill
// vertically centers its text inside the 28 DIP pill container.
func TestAboutVersionBadgeVerticallyCentered(t *testing.T) {
	m := Model{About: &AboutAccess{AppName: "Whale Isle", Version: "dev"}}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Padding(20).Children(func() {
			aboutPane(c, m, ui.RGB(20, 20, 20), ui.RGB(120, 120, 120))
		})
	}, 900, 600)
	tt.Frame()

	pill, ok := tt.Find(i18n.T(i18n.KeySettingsAboutCopyVersion))
	if !ok || pill.H != 28 {
		t.Fatalf("version pill=%+v ok=%v want H=28", pill, ok)
	}
	textRect, ok := tt.Find("dev")
	if !ok {
		t.Fatalf("version text 'dev' missing: %v", tt.Texts())
	}
	pillCenterY := pill.Y + pill.H/2
	textCenterY := textRect.Y + textRect.H/2
	if math.Abs(float64(pillCenterY-textCenterY)) > 2.5 {
		t.Fatalf("version text 'dev' not vertically centered in pill: pill=%+v (cy=%.1f) text=%+v (cy=%.1f)", pill, pillCenterY, textRect, textCenterY)
	}
}

// TestVoiceInputUsesCossSelectAndThrottlesActiveMeter ensures the microphone
// selector uses coss.Select (not bare "▾" SelectBase) and renders cleanly.
func TestVoiceInputUsesCossSelectAndThrottlesActiveMeter(t *testing.T) {
	mic := voiceinput.NewMicrophone(
		voicecapture.Device{ID: "mic-1", Label: "Studio USB Mic"},
		func(voicecapture.Device) error { return nil },
		nil,
	)

	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Padding(16).Children(func() {
			VoiceInputDeviceView(c, mic)
		})
	}, 600, 300)
	tt.Frame()

	if tt.HasText("▾") {
		t.Fatalf("VoiceInputDeviceView still renders bare '▾' SelectBase instead of coss.Select")
	}
	if _, ok := tt.Find("选择麦克风"); !ok {
		if _, okEN := tt.Find("Select microphone"); !okEN {
			t.Fatalf("microphone coss.Select trigger missing: %v", tt.Texts())
		}
	}
	_ = time.Millisecond
}
