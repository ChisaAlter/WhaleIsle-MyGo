package settings

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/config"
	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestSettingsSearchRevealsAndHighlightsCollapsedControls(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(previous) })

	prefs := theme.NewEditor(theme.Defaults(), func(theme.Preferences) error { return nil })
	workspace := `C:\workspace`
	port := "6767"
	autoRestart := true
	attempts := "3"
	delay := "1000"
	policy := config.RestartPreferences{AutoRestart: true, MaxAttempts: 3, BaseDelayMs: 1000}

	m := fullRailModel()
	m.Appearance = &AppearanceAccess{
		Host:        bodyFixtureHost(),
		Preferences: prefs,
	}
	m.Desktop = &DesktopAccess{
		Host:          bodyFixtureHost(),
		Preferences:   prefs,
		Fields:        DesktopFields{Workspace: &workspace, Port: &port, HarnessAutoRestart: &autoRestart, HarnessRestartMaxAttempts: &attempts, HarnessRestartBaseDelayMs: &delay},
		RestartPolicy: &policy,
		OnSave:        func() {},
	}
	m.About = &AboutAccess{
		AppName:    "Whale Isle",
		Version:    "0.3.13",
		Identifier: "ai.deepseek.whaleisle",
		UserData:   `C:\Users\test\AppData\Roaming\WhaleIsle`,
	}

	var highlightedKey, highlightedTitle string
	tt := ui.NewTester(func(c *ui.Context) {
		ModalView(c, m)
		jump := settingsSearchJumpState(c)
		highlightedKey = jump.HighlightedKey
		highlightedTitle = jump.HighlightedTitle
	}, 1440, 920)

	// 1. Search for "输入框字体" (nested inside Appearance -> 字体 -> 高级 disclosure)
	if err := tt.Click("搜索设置"); err != nil {
		t.Fatal(err)
	}
	tt.Type("输入框字体")
	if !tt.HasText("输入框字体") {
		t.Fatalf("search results missing 输入框字体: %v", tt.Texts())
	}
	if err := tt.Click("输入框字体"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText("填写本机已安装的字体名。留空则跟随界面字体。") {
		t.Fatalf("selecting 输入框字体 did not auto-expand advanced typography disclosure: %v", tt.Texts())
	}
	if highlightedKey != "font-composer" && highlightedTitle != "输入框字体" {
		t.Fatalf("expected font-composer to be visually highlighted, got key=%q title=%q", highlightedKey, highlightedTitle)
	}

	// 2. Search for "代码字体" (nested inside Appearance -> general-font-settings disclosure)
	if err := tt.Click("搜索设置"); err != nil {
		t.Fatal(err)
	}
	tt.Type("代码字体")
	if !tt.HasText("代码字体") {
		t.Fatalf("search results missing 代码字体: %v", tt.Texts())
	}
	if err := tt.Click("代码字体"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText("文本字体") || !tt.HasText("代码字体") {
		t.Fatalf("selecting 代码字体 did not auto-expand general-font-settings disclosure: %v", tt.Texts())
	}
	if highlightedKey != "general-family-code" && highlightedTitle != "代码字体" {
		t.Fatalf("expected general-family-code to be visually highlighted, got key=%q title=%q", highlightedKey, highlightedTitle)
	}

	// 3. Search for "内核崩溃自动重启" (nested inside About -> 应用数据与诊断 -> 本地运行与诊断 disclosures)
	if err := tt.Click("搜索设置"); err != nil {
		t.Fatal(err)
	}
	tt.Type("内核崩溃自动重启")
	if !tt.HasText("内核崩溃自动重启") {
		t.Fatalf("search results missing 内核崩溃自动重启: %v", tt.Texts())
	}
	if err := tt.Click("内核崩溃自动重启"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText("允许崩溃后自动重启") || !tt.HasText("最多重试次数（1–10）") {
		t.Fatalf("selecting 内核崩溃自动重启 did not auto-expand nested diagnostics disclosures: %v", tt.Texts())
	}
	if highlightedKey != "harness-restart-policy" && highlightedTitle != "内核崩溃自动重启" {
		t.Fatalf("expected harness-restart-policy to be visually highlighted, got key=%q title=%q", highlightedKey, highlightedTitle)
	}
}

func TestSettingsActiveFilterQueryAutoExpandsDisclosuresAndHighlightsRows(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(previous) })

	prefs := theme.NewEditor(theme.Defaults(), func(theme.Preferences) error { return nil })
	filter := "Composer font"
	m := Model{
		Store:            store.New(),
		AppearanceActive: true,
		Appearance: &AppearanceAccess{
			Host:        bodyFixtureHost(),
			Preferences: prefs,
		},
		FilterPtr: &filter,
		OnClose:   func() {},
	}

	var highlightedKey, highlightedTitle string
	tt := ui.NewTester(func(c *ui.Context) {
		ModalView(c, m)
		jump := settingsSearchJumpState(c)
		highlightedKey = jump.HighlightedKey
		highlightedTitle = jump.HighlightedTitle
	}, 1200, 900)

	if !tt.HasText("Composer font") || !tt.HasText("Leave empty to follow the interface font.") {
		t.Fatalf("active FilterPtr query did not auto-expand typography-advanced disclosure: %v", tt.Texts())
	}
	if highlightedKey != "font-composer" && highlightedTitle != "Composer font" {
		t.Fatalf("expected font-composer to be highlighted while FilterPtr matches, got key=%q title=%q", highlightedKey, highlightedTitle)
	}

	filter = "Text font"
	tt.Frame()
	if !tt.HasText("Text font") || !tt.HasText("Code font") {
		t.Fatalf("active FilterPtr query did not auto-expand general-font-settings disclosure: %v", tt.Texts())
	}
	if highlightedKey != "general-family-text" && highlightedTitle != "Text font" {
		t.Fatalf("expected general-family-text to be highlighted while FilterPtr matches, got key=%q title=%q", highlightedKey, highlightedTitle)
	}
}

func TestComposerBeamAndTypingFxMatchVendorLocales(t *testing.T) {
	previous := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(previous) })

	i18n.SetLocale(i18n.LocaleZH)
	if got := effectChoiceOptionLabel(false, "mode", "legacy"); got != "Legacy" {
		t.Fatalf("beam mode legacy zh = %q, want Legacy", got)
	}
	if got := effectChoiceOptionLabel(false, "mode", "aurora"); got != "Aurora" {
		t.Fatalf("beam mode aurora zh = %q, want Aurora", got)
	}
	if got := effectChoiceOptionLabel(false, "palette.id", "aurora"); got != "极光" {
		t.Fatalf("beam palette aurora zh = %q, want 极光", got)
	}
	if got := effectChoiceOptionLabel(false, "palette.id", "sunset"); got != "日落" {
		t.Fatalf("beam palette sunset zh = %q, want 日落", got)
	}
	if got := effectChoiceOptionLabel(true, "palette.id", "sunset"); got != "落日" {
		t.Fatalf("typing palette sunset zh = %q, want 落日", got)
	}
	if got := effectChoiceOptionLabel(false, "direction", "pingpong"); got != "往返" {
		t.Fatalf("beam direction pingpong zh = %q, want 往返", got)
	}
	if got := effectChoiceOptionLabel(false, "easing", "smoothstep"); got != "平滑曲线" {
		t.Fatalf("beam easing smoothstep zh = %q, want 平滑曲线", got)
	}
	if got := effectChoiceOptionLabel(true, "preset", "drop"); got != "落锤" {
		t.Fatalf("typing preset drop zh = %q, want 落锤", got)
	}
	if got := effectChoiceOptionLabel(true, "caretMode", "system"); got != "跟随系统" {
		t.Fatalf("typing caretMode system zh = %q, want 跟随系统", got)
	}

	i18n.SetLocale(i18n.LocaleEN)
	if got := effectChoiceOptionLabel(false, "direction", "pingpong"); got != "Ping-pong" {
		t.Fatalf("beam direction pingpong en = %q, want Ping-pong", got)
	}
	if got := effectChoiceOptionLabel(false, "easing", "ease-in-out"); got != "Ease in/out" {
		t.Fatalf("beam easing ease-in-out en = %q, want Ease in/out", got)
	}
	if got := effectChoiceOptionLabel(false, "easing", "smoothstep"); got != "Smooth curve" {
		t.Fatalf("beam easing smoothstep en = %q, want Smooth curve", got)
	}
	if got := effectChoiceOptionLabel(true, "caretMode", "system"); got != "System caret" {
		t.Fatalf("typing caretMode system en = %q, want System caret", got)
	}
	if got := appearancePresetChoices(false)[0][1]; got != "Follow theme" {
		t.Fatalf("appearance preset default en = %q, want Follow theme", got)
	}
}
