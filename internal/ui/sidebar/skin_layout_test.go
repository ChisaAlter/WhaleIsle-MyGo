package sidebar

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestSkinSidebarConditionalAssistantAndEmptyArchive(t *testing.T) {
	i18n.SetLocale("en")
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	for _, collapsed := range []bool{false, true} {
		for _, enabled := range []bool{false, true} {
			m := Model{Collapsed: collapsed, OnCreate: func() {}, OnToggle: func() {}, Snapshot: session.ViewState{WorkspaceReady: true}}
			if enabled {
				m.OnOpenAssistant = func() {}
			}
			tt := ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { View(c, m) }) }, 280, 700)
			_, present := tt.Find(i18n.T(i18n.KeySidebarAssistant))
			if present != enabled {
				t.Fatalf("collapsed%v enabled%v row%v", collapsed, enabled, present)
			}
			if !collapsed {
				name := i18n.T(i18n.KeySidebarArchivedSection)
				if _, ok := tt.Find(name); !ok {
					t.Fatalf("empty default archive disclosure missing texts%q", tt.Texts())
				}
				if err := tt.Click(name); err != nil {
					t.Fatal(err)
				}
				if !tt.HasText("No archived sessions") {
					t.Fatal("empty archive expansion has no explanation")
				}
			}
		}
	}
}
func TestSkinSidebarMetricsKeepDefaultPreferenceScale(t *testing.T) {
	old := theme.CurrentPreferences()
	t.Cleanup(func() { theme.SetPreferences(old) })
	p := theme.Defaults()
	p.Interface.Size = 20
	theme.SetPreferences(p)
	if l := theme.ResolvedSkinLayout(theme.IDSystem, false); l.SidebarFont != 17.5 {
		t.Fatalf("default font user scale lost%+v", l)
	}
	p.Light = "codex"
	theme.SetPreferences(p)
	if l := theme.ResolvedSkinLayout(theme.IDSystem, false); l.SidebarFont != 13 || l.NewHeight != 32 || l.BrandHeight != 44 {
		t.Fatalf("skin absolute source values%+v", l)
	}
}
