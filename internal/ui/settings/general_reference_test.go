package settings

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/shortcuts"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// TestGeneralReferenceModalSnapshot renders the production settings modal at
// the reference viewport. With SECONDARY_SNAPSHOT_DIR set (the existing body
// snapshot hook) it writes a PNG for the visual comparison; without it the test
// still proves the modal builds at 1440x920.
func TestGeneralReferenceModalSnapshot(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(old) })

	h := bodyFixtureHost()
	e := shortcuts.NewEditor(nil, nil, func(map[string]string) error { return nil })
	locale := "zh"
	closeToTray := true
	prefs := theme.NewEditor(theme.Defaults(), func(theme.Preferences) error { return nil })
	m := Model{
		DesktopActive: true,
		Desktop: &DesktopAccess{
			Host:        h,
			Shortcuts:   e,
			Preferences: prefs,
			Fields:      DesktopFields{Locale: &locale, CloseToTray: &closeToTray},
			OnSave:      func() {},
		},
		OnClose: func() {},
	}
	tt := ui.NewTester(func(c *ui.Context) { ModalView(c, m) }, 1440, 920)
	// The shared dialog surface fades in; let the tester advance its clock so
	// the snapshot shows the settled surface, not the first animated frame.
	for range 12 {
		tt.Frame()
		time.Sleep(16 * time.Millisecond)
	}
	if !tt.HasText("通用设置") || !tt.HasText("编辑快捷键") {
		t.Fatalf("settings modal did not render its General page: %v", tt.Texts())
	}
	saveBodySnapshot(t, tt, "settings-general-reference-1440.png")
}

// TestGeneralPreferencesUseReferenceCopy pins the General page's row copy to
// the installed build's locale dictionaries (ui-permission-presets,
// ui-chat, ui-shortcuts). The page used to invent its own longer labels and
// descriptions, which also made every row taller than the reference.
func TestGeneralPreferencesUseReferenceCopy(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(old) })

	h := bodyFixtureHost()
	e := shortcuts.NewEditor(nil, nil, func(map[string]string) error { return nil })
	locale := "zh"
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(24).Children(func() {
			desktopPane(c, Model{Desktop: &DesktopAccess{Host: h, Shortcuts: e, Fields: DesktopFields{Locale: &locale}}}, c.Theme().Text, c.Theme().TextMuted)
		})
	}, 900, 1600)

	for _, want := range []string{
		"权限",
		"选择新会话的默认权限模式",
		"语言",
		"工作步骤展示",
		"选择希望看到多少工具调用细节",
		"工作步骤收起时机",
		"选择何时自动收起工作步骤",
		"网页链接默认打开方式",
		"对话中网页链接的打开位置",
		"快捷键",
		"查看和编辑当前可用的快捷键和输入操作",
		"编辑快捷键",
	} {
		if !tt.HasText(want) {
			t.Fatalf("General page is missing reference copy %q\n%v", want, tt.Texts())
		}
	}

	// The old invented labels must not survive. HasText is a substring match, so
	// compare the exact label nodes where one label contains another.
	stale := []string{"新任务默认权限", "只影响以后创建的任务", "界面语言", "选择界面使用的语言", "工作详情", "键盘快捷键"}
	for _, text := range stale {
		if tt.HasText(text) {
			t.Fatalf("General page still shows the drifted copy %q", text)
		}
	}
	for _, text := range tt.Texts() {
		if text == "链接打开方式" {
			t.Fatal("General page still labels the link row without 网页…默认打开方式")
		}
	}
}

// TestGeneralShortcutRowOpensTheEditor keeps the reference interaction: the
// compact Setting-Cell's button opens the shortcut editor in a dialog, and the
// dialog closes again from its own Close button.
func TestGeneralShortcutRowOpensTheEditor(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(old) })

	h := bodyFixtureHost()
	e := shortcuts.NewEditor(nil, nil, func(map[string]string) error { return nil })
	locale := "zh"
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(24).Children(func() {
			desktopPane(c, Model{Desktop: &DesktopAccess{Host: h, Shortcuts: e, Fields: DesktopFields{Locale: &locale}}}, c.Theme().Text, c.Theme().TextMuted)
		})
	}, 900, 1600)

	if err := tt.Click("edit-shortcuts"); err != nil {
		t.Fatalf("shortcut row has no reachable editor button: %v", err)
	}
	if !tt.HasText("恢复全部默认") {
		t.Fatalf("shortcut reference did not open: %v", tt.Texts())
	}
	if err := tt.Click("关闭快捷键"); err != nil {
		t.Fatalf("shortcut dialog has no close action: %v", err)
	}
	if tt.HasText("恢复全部默认") {
		t.Fatal("shortcut dialog stayed open after its close action")
	}
}
