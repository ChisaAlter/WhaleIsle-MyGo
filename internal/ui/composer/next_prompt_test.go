package composer

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

func nextPromptRegressionModel() Model {
	d := NewDraftState("")
	return Model{Draft: &d.Text, State: d, TextInput: d.Client, Snapshot: session.ViewState{ClientReady: true, HistoryReady: true, SelectedID: "a", Items: []session.Item{{SessionID: "a"}}}, NextPrompt: session.NextPromptSuggestion{SessionID: "a", Seq: 3, Text: "继续检查"}}
}

func TestNextPromptEligibilityAndAcceptanceGates(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*Model)
	}{
		{"not-open", func(m *Model) { m.Snapshot.HistoryReady = false }},
		{"opening", func(m *Model) { m.Snapshot.HistoryLoading = true }},
		{"open-error", func(m *Model) { m.Snapshot.HistoryError = "lost feed" }},
		{"managed", func(m *Model) { m.Snapshot.Items[0].Presentation = &session.SessionPresentation{Composer: "managed"} }},
		{"permission-popup", func(m *Model) { m.State.PermissionCommand = ModelCommandState{Open: true, SessionID: "a"} }},
		{"model-popup", func(m *Model) { m.State.ModelCommand = ModelCommandState{Open: true, SessionID: "a"} }},
		{"running", func(m *Model) { m.Snapshot.Items[0].Running = true }},
		{"subagent", func(m *Model) { m.Snapshot.Items[0].Origin = "subagent" }},
		{"draft-attachment", func(m *Model) { m.State.Attachments = []session.ComposerAttachment{{Path: "kept.txt"}} }},
		{"inert", func(m *Model) { m.Inert = true }},
		{"pending-interaction", func(m *Model) { m.StopBlocked = true }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := nextPromptRegressionModel()
			calls := 0
			m.OnAcceptNextPrompt = func() string { calls++; return m.NextPrompt.Text }
			tc.change(&m)
			if acceptNextPrompt(m) || calls != 0 || m.State.Text != "" {
				t.Fatal("ineligible suggestion consumed its owner callback or changed the draft")
			}
			var painted bool
			ui.NewTester(func(c *ui.Context) { painted = paintNextPrompt(c, m, c.Theme().TextMuted) }, 600, 300)
			if painted || calls != 0 {
				t.Fatal("ineligible ghost suggestion remained actionable")
			}
		})
	}
}

func TestNextPromptTabInsertsWithoutSendingOrInterruptingIME(t *testing.T) {
	m := nextPromptRegressionModel()
	accepted, submitted := 0, 0
	m.OnAcceptNextPrompt = func() string { accepted++; return m.NextPrompt.Text }
	m.OnSubmit = func() { submitted++ }
	if !draftInputEvent(m.TextInput, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyTab}) || m.State.Text != "继续检查" || accepted != 1 || submitted != 0 {
		t.Fatal("plain Tab did not insert exactly the offered text without a prompt")
	}
	selection := m.TextInput.Selection().Range
	if selection.Start != ui.UTF16Len(m.State.Text) || selection.End != selection.Start {
		t.Fatalf("suggestion caret was not restored at the UTF-16 end: %+v", selection)
	}
	m = nextPromptRegressionModel()
	m.OnAcceptNextPrompt = func() string { accepted++; return m.NextPrompt.Text }
	m.TextInput.SetMarkedText(nil, "拼", ui.TextInputRange{Start: 1, End: 1})
	before := m.State.Text
	if acceptNextPrompt(m) || m.State.Text != before || accepted != 1 {
		t.Fatal("suggestion interrupted marked Chinese text")
	}
}
