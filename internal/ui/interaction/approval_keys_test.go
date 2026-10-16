package interaction

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/store/interaction"
)

func TestApprovalKeyboard(t *testing.T) {
	p := interaction.Pending{Event: interaction.EventApproval, EventID: "approval-A", AgentID: "session-A"}
	live := p
	allowed, denied := 0, 0
	allow := func(id string) {
		if id != p.EventID {
			t.Fatal("wrong approval")
		}
		allowed++
	}
	deny := func(id string) {
		if id != p.EventID {
			t.Fatal("wrong approval")
		}
		denied++
	}
	snapshot := func() interaction.ViewState { return interaction.ViewState{Pending: []interaction.Pending{live}} }
	var keys approvalKeys
	press := func(key ui.Key, mods ui.Modifiers, repeat, button bool) bool {
		return keys.handle(ui.InputEvent{Kind: ui.InputKeyDown, Key: key, Mods: mods, Repeat: repeat}, p, button, snapshot, allow, deny)
	}
	if !press(ui.KeyEnter, 0, false, false) || !press(ui.KeyEscape, 0, false, false) || allowed != 1 || denied != 1 {
		t.Fatal("focused approval keys missing")
	}
	if press(ui.KeyEnter, 0, false, true) || !press(ui.KeyEscape, 0, false, true) || allowed != 1 || denied != 2 {
		t.Fatal("button Enter was hijacked or Escape missing")
	}
	for _, guard := range []string{"repeat", "modified", "sending", "error", "foreign-agent", "foreign-event", "questions", "continued", "composition", "composition-ended"} {
		live = p
		keys = approvalKeys{}
		mods, repeat := ui.Modifiers(0), false
		switch guard {
		case "repeat":
			repeat = true
		case "modified":
			mods = ui.Shift
		case "sending":
			live.Sending = true
		case "error":
			live.Error = "retry required"
		case "foreign-agent":
			live.AgentID = "session-B"
		case "foreign-event":
			live.EventID = "approval-B"
		case "questions":
			live.Event = interaction.EventUserQuestions
		case "continued":
			live.Continued = true
		case "composition":
			keys.composing = true
		case "composition-ended":
			keys.compositionEnded = true
		}
		press(ui.KeyEnter, mods, repeat, false)
		press(ui.KeyEscape, mods, repeat, false)
		if allowed != 1 || denied != 2 {
			t.Fatalf("%s bypassed approval guard", guard)
		}
	}
	live = p
	keys = approvalKeys{}
	text := "draft"
	modal := false
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Children(func() {
			ui.Box(c.Key(p.EventID)).Height(40).Width(300).Focusable().Label("Approval body").HandleInput(func(ev ui.InputEvent) bool { return keys.handle(ev, p, false, snapshot, allow, deny) })
			ui.TextInput(c, &text).Label("Editor")
		})
		if modal {
			ui.Modal(c, &modal, func() { ui.TextInput(c, &text).Label("Modal editor").AutoFocus() })
		}
	}, 500, 300)
	if err := tt.Click("Approval body"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEnter)
	tt.Key(0, ui.KeyEscape)
	if allowed != 2 || denied != 3 {
		t.Fatal("focused native handler not reached")
	}
	if err := tt.Click("Editor"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEnter)
	tt.Key(0, ui.KeyEscape)
	if allowed != 2 || denied != 3 {
		t.Fatal("editor keys reached approval")
	}
	modal = true
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Key(0, ui.KeyEscape)
	if allowed != 2 || denied != 3 {
		t.Fatal("modal keys reached approval")
	}
}
