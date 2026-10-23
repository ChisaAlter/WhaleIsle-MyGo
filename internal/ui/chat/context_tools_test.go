package chat

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestToolContextCountsNamesAndExitPresence(t *testing.T) {
	old := i18n.Locale()
	defer i18n.SetLocale(old)
	for _, tc := range []struct {
		locale, title, summary, added, removed string
	}{
		{i18n.LocaleEN, "Tools updated", "2 added, 1 removed", "Added: search, read_file", "Removed: old_search"},
		{i18n.LocaleZH, "工具已更新", "新增 2 个，移除 1 个", "新增：search, read_file", "移除：old_search"},
	} {
		t.Run(tc.locale, func(t *testing.T) {
			i18n.SetLocale(tc.locale)
			msg := contextFixture("tools", "tool-registry", `[{"type":"tool-addition","toolName":"search"},{"type":"tool-addition","toolName":"read_file"},{"type":"tool-removal","toolName":"old_search"}]`)
			for _, reduced := range []bool{false, true} {
				tt := ui.NewTester(func(c *ui.Context) {
					ui.Column(c).FillWidth().Padding(12).Children(func() {
						ContextInjectionView(c, msg, "s", theme.Resolve(theme.IDDeepseekLight).Light)
					})
				}, 420, 400)
				tt.SetPreferences(ui.Preferences{ReduceMotion: reduced})
				at := time.Unix(100, 0)
				tt.SetTime(at)
				name := tc.title + " " + tc.summary
				row, ok := tt.Find(name)
				if !ok || row.H != 24 || row.X != 12 || row.W != 396 {
					t.Fatalf("disclosure geometry = %+v found=%v", row, ok)
				}
				if tt.HasText("tool-registry") || tt.HasText(tc.added) {
					t.Fatalf("collapsed tool notice exposed metadata/body: %q", tt.Texts())
				}
				if err := tt.Click(name); err != nil {
					t.Fatal(err)
				}
				tt.SetTime(at.Add(100 * time.Millisecond))
				if !tt.HasText(tc.added) || !tt.HasText(tc.removed) {
					t.Fatalf("expanded name lists absent: %q", tt.Texts())
				}
				if err := tt.Click(name); err != nil {
					t.Fatal(err)
				}
				if tt.HasText(tc.added) || tt.HasText(tc.removed) {
					t.Fatal("retained exit body remains interactive/findable")
				}
				tt.SetTime(at.Add(301 * time.Millisecond))
				if tt.HasText(tc.added) || tt.HasText(tc.removed) {
					t.Fatal("exit body remained mounted after 200ms")
				}
			}
		})
	}
}

func TestToolContextSingletonAndMixedContent(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	for _, tc := range []struct{ kind, label string }{{"tool-addition", "Tool added: last_tool"}, {"tool-removal", "Tool removed: last_tool"}} {
		msg := contextFixture("single", "tool-registry", `[{"type":"`+tc.kind+`","toolName":"last_tool"}]`)
		tt := ui.NewTester(func(c *ui.Context) { ContextInjectionView(c, msg, "s", theme.Resolve(theme.IDDeepseekLight).Light) }, 360, 160)
		if !tt.HasText(tc.label) || tt.HasText("Context injection") || tt.HasText("tool-registry") {
			t.Fatalf("singleton notice = %q", tt.Texts())
		}
		if err := tt.Click(tc.label); err != nil {
			t.Fatal(err)
		}
		if tt.HasText("Added:") || tt.HasText("Removed:") || tt.HasText("Copy context") {
			t.Fatal("single tool notice acquired an expanded body")
		}
	}
	mixed := contextFixture("mixed", "tool-registry", `[{"type":"text","text":"mixed content"},{"type":"tool-addition","toolName":"search"}]`)
	tt := ui.NewTester(func(c *ui.Context) { ContextInjectionView(c, mixed, "s", theme.Resolve(theme.IDDeepseekLight).Light) }, 480, 400)
	if !tt.HasText("Context injection") || tt.HasText("Tools updated") || tt.HasText("snapshot") {
		t.Fatalf("mixed content was treated as an all-tool notice: %q", tt.Texts())
	}
	if err := tt.Click("Context injection tool-registry"); err != nil || !strings.Contains(strings.Join(tt.Texts(), "\n"), "mixed content") || !tt.HasText("Unknown content block") {
		t.Fatalf("mixed opaque detail missing: %v", err)
	}
}
