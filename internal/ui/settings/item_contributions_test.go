package settings

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginui"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestSettingsItemWinnerReplacesActualInterfaceControlAndSearch(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(old) })
	replaced, after := true, false
	h := bodyFixtureHost()
	writes := 0
	h.OnChange = func(string, string, string) { writes++ }
	contributions := pluginui.PublicContributions{
		KeepNative: func(slot, _, id string) bool {
			return slot != "settings.interface.item" || id != "titlebar-git" || !replaced
		},
		Entries: func(slot, _ string) []pluginui.PublicSlotEntry {
			if slot != "settings.interface.item" {
				return nil
			}
			git := pluginui.PublicSlotEntry{ID: "titlebar-git", Native: true}
			terminal := pluginui.PublicSlotEntry{ID: "terminal-toggle", Native: true}
			surfaces := pluginui.PublicSlotEntry{ID: "surfaces-toggle", Native: true}
			if !replaced {
				return []pluginui.PublicSlotEntry{git, terminal, surfaces}
			}
			external := pluginui.PublicSlotEntry{ID: "titlebar-git", Handle: "replacement", Label: "Replacement Git", Nodes: []pluginui.NativeNode{{Text: "Replacement Git"}}}
			if after {
				return []pluginui.PublicSlotEntry{terminal, external, surfaces}
			}
			return []pluginui.PublicSlotEntry{external, terminal, surfaces}
		},
		RenderEntry: func(c *ui.Context, _, _, handle string) bool {
			ui.Text(c, "Replacement Git")
			return handle == "replacement"
		},
	}
	m := Model{Interface: &InterfaceAccess{Host: h}, PublicContributions: contributions}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			done := beginSettingsItemContributions(c, contributions, SectionInterface)
			defer done()
			interfacePane(c, m, c.Theme().Text, c.Theme().TextMuted)
		})
	}, 780, 1400)
	for _, position := range []bool{false, true} {
		after = position
		tt.Frame()
		if tt.HasText("Titlebar Git actions") {
			t.Fatal("shadowed native Git control remained")
		}
		count := 0
		for _, text := range tt.Texts() {
			if text == "Replacement Git" {
				count++
			}
		}
		if count != 1 {
			t.Fatalf("replacement rendered %d times", count)
		}
		r, _ := tt.Find("Replacement Git")
		terminal, _ := tt.Find("Terminal drawer toggle")
		if (r.Y > terminal.Y) != position {
			t.Fatalf("replacement ignored original order: external=%+v terminal=%+v", r, terminal)
		}
		for _, entry := range searchEntries(m) {
			if entry.title == "Titlebar Git actions" {
				t.Fatal("shadowed native search entry remained")
			}
		}
		matches := settingsSearchMatches(m, map[SectionID]bool{SectionInterface: true}, "Replacement Git")
		if len(matches) != 1 {
			t.Fatalf("replacement search entries: %+v", matches)
		}
	}
	if writes != 0 {
		t.Fatal("rendering a replacement wrote native settings")
	}
	replaced = false
	tt.Frame()
	if !tt.HasText("Titlebar Git actions") || tt.HasText("Replacement Git") {
		t.Fatal("withdraw did not restore only the native recipe")
	}
	tt.Key(0, ui.KeyTab)
	tt.Key(0, ui.KeySpace)
	if writes != 1 {
		t.Fatal("restored native control did not regain its callback")
	}
}

func TestSettingsItemWinnerRemovesCompositeFontAndEffectPayloads(t *testing.T) {
	old, locale := theme.CurrentPreferences(), i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { theme.SetPreferences(old); i18n.SetLocale(locale) })
	replaced := false
	contributions := pluginui.PublicContributions{
		KeepNative: func(slot, _, id string) bool {
			return !replaced || slot != "settings.appearance.item" || id != "font-size" && id != "composer-typing-fx"
		},
		Entries: func(slot, _ string) []pluginui.PublicSlotEntry {
			if slot != "settings.appearance.item" || !replaced {
				return nil
			}
			return []pluginui.PublicSlotEntry{{ID: "font-size", Handle: "font"}, {ID: "composer-typing-fx", Handle: "typing"}, {ID: "composer-beam", Native: true}}
		},
		RenderEntry: func(c *ui.Context, _, _, handle string) bool { ui.Text(c, "External "+handle); return true },
	}
	a := &AppearanceAccess{Host: bodyFixtureHost(), Preferences: theme.NewEditor(theme.Defaults(), func(theme.Preferences) error { return nil })}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			done := beginSettingsItemContributions(c, contributions, SectionAppearance)
			defer done()
			appearancePane(c, Model{Appearance: a, PublicContributions: contributions}, c.Theme().Text, c.Theme().TextMuted)
		})
	}, 900, 3000)
	if err := tt.Click("Typing effect settings"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Save and apply") {
		t.Fatal("fixture did not open the actual typing editor")
	}
	replaced = true
	tt.Frame()
	if tt.HasText("Typing effects") || tt.HasText("Save and apply") || tt.HasText("Only affects conversation content") {
		t.Fatal("shadowed composite retained a native row/editor")
	}
	for _, label := range []string{"External font", "External typing", "Thinking glow when sending"} {
		if !tt.HasText(label) {
			t.Fatal("replacement removed an unrelated recipe", label)
		}
	}
	replaced = false
	tt.Frame()
	if !tt.HasText("Typing effects") || !tt.HasText("Only affects conversation content") || tt.HasText("Save and apply") {
		t.Fatal("withdraw did not remount clean native composite payloads")
	}
}
