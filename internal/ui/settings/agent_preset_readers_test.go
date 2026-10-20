package settings

import (
	"fmt"
	"maps"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestPresetGuideRetainsEachPageScrollAndReturnsToOpeningAction(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	original := presetGuidesEN["standard"]
	guide := presetModeGuide{Intro: "Fixture guide"}
	for i := 0; i < 35; i++ {
		guide.Explanation += fmt.Sprintf("Explanation paragraph %02d\n\n", i)
		guide.Usage += fmt.Sprintf("Usage paragraph %02d\n\n", i)
	}
	presetGuidesEN["standard"] = guide
	t.Cleanup(func() { presetGuidesEN["standard"] = original })
	visible := true
	tt := ui.NewTester(func(c *ui.Context) {
		if visible {
			agentPresetGuidePane(c, session.ManagedAgentPreset{ID: "standard", BuiltIn: true}, c.Theme().Text, c.Theme().Text)
		}
	}, 800, 520)
	click := func(label string) {
		t.Helper()
		if err := tt.Click(label); err != nil {
			t.Fatal(err)
		}
	}
	positions := func(prefix string) map[string]ui.Rect {
		t.Helper()
		out := map[string]ui.Rect{}
		for i := 0; i < 35; i++ {
			label := fmt.Sprintf("%s paragraph %02d", prefix, i)
			if r, ok := tt.Find(label); ok && r.H > 0 && r.W > 0 {
				out[label] = r
			}
		}
		if len(out) == 0 {
			t.Fatal("no visible guide paragraphs", prefix)
		}
		return out
	}
	click(i18n.T(KeyPass4ModeExplanation) + " · standard")
	if !tt.Focused("Mode details") || tt.Focused(i18n.T(KeyPass4ModeExplanation) + " · standard") {
		t.Fatal("opening guide did not focus its selected tab")
	}
	initialExplanation := positions("Explanation")
	tt.Move(400, 350)
	tt.Scroll(400, 350, 0, 190)
	scrolledExplanation := positions("Explanation")
	if maps.Equal(initialExplanation, scrolledExplanation) {
		t.Fatal("explanation did not scroll")
	}
	tt.Key(0, ui.KeyTab)
	tt.Key(0, ui.KeyEnter)
	initialUsage := positions("Usage")
	tt.Scroll(400, 350, 0, 95)
	scrolledUsage := positions("Usage")
	if maps.Equal(initialUsage, scrolledUsage) {
		t.Fatal("usage did not scroll")
	}
	tt.Key(ui.Shift, ui.KeyTab)
	tt.Key(0, ui.KeyEnter)
	if !maps.Equal(positions("Explanation"), scrolledExplanation) {
		t.Fatal("returning to explanation lost its own scroll")
	}
	tt.Key(0, ui.KeyTab)
	tt.Key(0, ui.KeyEnter)
	if !maps.Equal(positions("Usage"), scrolledUsage) {
		t.Fatal("returning to usage lost its own scroll")
	}
	tt.Key(0, ui.KeyEscape)
	if tt.HasText("Fixture guide") || !tt.Focused(i18n.T(KeyPass4ModeExplanation) + " · standard") {
		t.Fatal("Escape did not close only the guide and restore its opening action")
	}
	click("How to use · standard")
	if !maps.Equal(positions("Usage"), initialUsage) {
		t.Fatal("new guide opening retained the previous reader scroll")
	}
	click("Close help · standard")
	if !tt.Focused("How to use · standard") {
		t.Fatal("close button returned to the wrong action")
	}
	click(i18n.T(KeyPass4ModeExplanation) + " · standard")
	visible = false
	tt.Frame()
	visible = true
	tt.Frame()
	if tt.HasText("Fixture guide") {
		t.Fatal("leaving Settings reopened the old reader")
	}
}

func TestPresetConfigurationSingleClickModalAndReturnFocus(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	state := session.AgentPresetReaderState{Ticket: 3}
	focus := presetReaderFocus{}
	reads, retries, closed := 0, 0, 0
	rows := session.AgentPresetManagerState{Loaded: true, Rows: []session.ManagedAgentPreset{{ID: "broken", Broken: "fixture declaration error"}}}
	tt := ui.NewTester(func(c *ui.Context) {
		a := &AgentPresetReaderAccess{State: state, focus: &focus, OnSelect: func(ticket uint64, id string) {
			if ticket != 3 || id != "broken" {
				t.Fatal("wrong captured view action", ticket, id)
			}
			reads++
			state.SelectedID = id
			state.Open = true
			state.Busy = true
		}, OnRead: func(ticket uint64) {
			if ticket != state.Ticket {
				t.Fatal("wrong retry owner")
			}
			retries++
			state.Busy = true
			state.Error = ""
		}, OnClose: func(ticket uint64) {
			if ticket != state.Ticket {
				t.Fatal("wrong close owner")
			}
			closed++
			state.Open = false
		}, OnCopy: func(uint64) { t.Fatal("render or open copied the declaration") }}
		ui.Column(c).Fill().Children(func() {
			if agentPresetViewAction(c, a, "broken", state.Busy) {
				a.OnSelect(a.State.Ticket, "broken")
			}
			agentPresetReaderPane(c, a, rows, c.Theme().Text, c.Theme().Text)
		})
	}, 800, 600)
	if reads != 0 {
		t.Fatal("render read a declaration")
	}
	if err := tt.Click("View configuration · broken"); err != nil {
		t.Fatal(err)
	}
	if reads != 1 || !tt.HasText("Loading declaration") {
		t.Fatal("single view action did not open the loading modal")
	}
	tt.Key(0, ui.KeyTab)
	tt.Key(0, ui.KeyTab)
	if reads != 1 || !state.Open || tt.Focused("View configuration · broken") {
		t.Fatal("modal let keyboard focus or a second admission escape to the card")
	}
	state.Busy = false
	state.Error = "fixture refusal"
	tt.Frame()
	if !tt.HasText("fixture refusal") {
		t.Fatal("read failure vanished")
	}
	if err := tt.Click("Retry"); err != nil {
		t.Fatal(err)
	}
	if retries != 1 || reads != 1 {
		t.Fatal("retry changed the selected declaration")
	}
	state.Busy = false
	state.Error = ""
	state.Loaded = true
	state.Content = "- id: broken\n  prompt: do not execute\n"
	state.TotalBytes = len(state.Content)
	tt.Frame()
	const sourceLabel = "Preset configuration source"
	if err := tt.Click(sourceLabel); err != nil {
		t.Fatal("read-only source control not visible:", err)
	}
	// TextArea's buffer is not in Tester.Texts/HasText. Exercise its real
	// selection/copy path, then prove that input cannot mutate the buffer.
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Key(ui.Cmd, ui.KeyC)
	if got := tt.Clipboard(); got != state.Content {
		t.Fatalf("read-only source lost bytes: %q", got)
	}
	if _, active := tt.TextCaret(); active {
		t.Fatal("read-only source enabled IME editing")
	}
	tt.Type("SHOULD NOT REPLACE SOURCE")
	tt.Key(0, ui.KeyBackspace)
	tt.Command("paste")
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Key(ui.Cmd, ui.KeyC)
	if got := tt.Clipboard(); got != state.Content {
		t.Fatalf("read-only editor accepted an edit: %q", got)
	}
	tt.Key(0, ui.KeyEscape)
	if state.Open || closed != 1 || !tt.Focused("View configuration · broken") {
		t.Fatal("modal close failed to retire and return to its original card")
	}
	if _, present := tt.Find(sourceLabel); present || reads != 1 {
		t.Fatal("closing reader reopened content or repeated the read")
	}
}
