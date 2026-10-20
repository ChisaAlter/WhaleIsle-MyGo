package settings

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestAgentPresetSettingsParity(t *testing.T) {
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	rows := session.AgentPresetManagerState{Loaded: true, Writable: true, DefaultID: "standard", Rows: []session.ManagedAgentPreset{
		{ID: "standard", BuiltIn: true},
		{ID: "mine", Name: "My mode", Description: "Own description"},
		{ID: "broken", Hidden: true, Broken: "missing declaration"},
		{ID: "hidden", Hidden: true},
	}}
	selected, reading := "", ""
	a := AgentPresetsAccess{Selected: &selected, Reader: &AgentPresetReaderAccess{State: session.AgentPresetReaderState{Ticket: 3}, OnSelect: func(ticket uint64, id string) {
		if ticket != 3 {
			t.Fatal(ticket)
		}
		reading = id
	}}}
	tt := ui.NewTester(func(c *ui.Context) { agentPresetRosterPane(c, &a, rows, ui.RGB(255, 255, 255), ui.RGB(150, 150, 150)) }, 1200, 1800)
	text := strings.Join(tt.Texts(), "\n")
	for _, want := range []string{"内置", "自定义", "Own description", "missing declaration"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing roster detail %q in %s", want, text)
		}
	}
	if strings.Contains(text, "hidden") {
		t.Fatal("healthy hidden preset leaked")
	}
	if err := tt.Click("查看配置 · broken"); err != nil {
		t.Fatal(err)
	}
	if reading != "broken" || selected != "" {
		t.Fatalf("independent reader: reading=%q selected=%q", reading, selected)
	}
	if err := tt.Click("选择默认 · My mode (mine)"); err != nil {
		t.Fatal(err)
	}
	if selected != "My mode (mine)" {
		t.Fatal("default row was not staged")
	}
	if err := tt.Click("模式说明 · standard"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(tt.Texts(), "\n"), "工作方式") {
		t.Fatal("mode explanation did not open")
	}
	tt.Key(0, ui.KeyTab)
	tt.Key(0, ui.KeyEnter)

	if !strings.Contains(strings.Join(tt.Texts(), "\n"), "修复一个问题") {
		t.Fatal("mode examples did not open")
	}
	if err := tt.Click("关闭帮助 · standard"); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(tt.Texts(), "\n"), "修复一个问题") {
		t.Fatal("help did not close")
	}
	if selected != "My mode (mine)" || reading != "broken" {
		t.Fatal("help changed selection")
	}
	for _, locale := range []string{i18n.LocaleZH, i18n.LocaleEN} {
		i18n.SetLocale(locale)
		for _, id := range []string{"standard", "ptc", "minimal", "cordis"} {
			guide, ok := agentPresetGuide(session.ManagedAgentPreset{ID: id, BuiltIn: true})
			if !ok || guide.Explanation == "" || guide.Usage == "" {
				t.Fatalf("missing guide %s %v", id, locale)
			}
			if _, ok := agentPresetGuide(session.ManagedAgentPreset{ID: id, Name: "Custom override"}); ok {
				t.Fatal("custom preset inherited built-in claims")
			}
		}
	}
	if _, ok := agentPresetGuide(session.ManagedAgentPreset{ID: "unknown", BuiltIn: true}); ok {
		t.Fatal("unknown guide")
	}
	large := strings.Repeat("界", session.AgentPresetContentLimit*2/3+3) + "\nEND\n"
	var restored strings.Builder
	pages := 0
	for start := 0; start < len(large); {
		page, next := agentPresetDocumentPage(large, start)
		if len(page) > session.AgentPresetContentLimit || !utf8.ValidString(page) || next <= start {
			t.Fatal("invalid document page")
		}
		restored.WriteString(page)
		start = next
		pages++
	}
	if restored.String() != large || pages < 3 {
		t.Fatal("pages lost part of the declaration")
	}
}
