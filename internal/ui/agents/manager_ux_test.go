package agents

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/errcopy"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestPresetManagerLocalizedAndSanitizesBrokenError(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })

	rawBroken := "dial tcp 127.0.0.1:9: connectex: connection refused"
	state := session.AgentPresetManagerState{
		Loaded:    true,
		Writable:  true,
		DefaultID: "default",
		Rows: []session.ManagedAgentPreset{
			{
				ID:          "default",
				Name:        "Default Agent",
				Description: "Primary preset",
			},
			{
				ID:     "broken-one",
				Name:   "Broken Agent",
				Broken: rawBroken,
			},
		},
	}

	tt := ui.NewTester(func(c *ui.Context) {
		presetManagerStateView(c, state, func() {}, func(string) {}, c.Theme().Text, c.Theme().TextMuted)
	}, 700, 500)

	for _, want := range []string{
		"Agent preset manager",
		"Refresh presets",
		"Configuration unavailable: " + errcopy.SentenceText(rawBroken),
	} {
		if !tt.HasText(want) {
			t.Fatalf("missing localized EN copy %q; texts=%q", want, tt.Texts())
		}
	}
	for _, banned := range []string{"Agent 预设管理", "刷新预设", "配置不可用：", rawBroken} {
		if tt.HasText(banned) {
			t.Fatalf("unexpected raw or unlocalized text %q; texts=%q", banned, tt.Texts())
		}
	}
}

func TestAgentsPanelSelectedRowRendersInDarkMode(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })

	m := Model{
		ThemeID:    "deepseek",
		SystemDark: true,
		Snapshot: session.ViewState{
			SelectedID: "child-1",
			Items: []session.Item{
				{SessionID: "root-1", Title: "Root Task"},
				{SessionID: "child-1", ParentID: "root-1", Title: "Worker Alpha", Running: true},
			},
		},
	}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 420, 500)
	tt.SetDark(true)
	tt.Frame()
	if !tt.HasText("Worker Alpha") {
		t.Fatalf("missing selected child row in dark mode; texts=%q", tt.Texts())
	}
}
