package composer

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

// Removing target/expiry checks or routing Escape before local editing/IME
// handling must fail this fixture. It exercises the shipped native input path.
func TestStopSequence(t *testing.T) {
	now := time.Unix(100, 0)
	target := stopTarget{sessionID: "A", turn: 7, generation: 2, host: 1}
	var sequence stopSequence
	if sequence.press(target, now, stopSequenceInterval) || !sequence.press(target, now.Add(500*time.Millisecond), stopSequenceInterval) {
		t.Fatal("two independent presses within inclusive 500ms window must stop")
	}
	if sequence.press(target, now.Add(501*time.Millisecond), stopSequenceInterval) {
		t.Fatal("accepted pair must reset")
	}
	sequence.reset()
	sequence.press(target, now, stopSequenceInterval)
	if sequence.press(target, now.Add(501*time.Millisecond), stopSequenceInterval) {
		t.Fatal("expired pair stopped")
	}
	for _, changed := range []stopTarget{{sessionID: "B", turn: 7, generation: 2, host: 1}, {sessionID: "A", turn: 8, generation: 2, host: 1}, {sessionID: "A", turn: 7, generation: 3, host: 1}, {sessionID: "A", turn: 7, generation: 2, host: 2}} {
		sequence.reset()
		sequence.press(target, now, stopSequenceInterval)
		if sequence.press(changed, now.Add(time.Millisecond), stopSequenceInterval) {
			t.Fatal("changed cancellation target paired")
		}
	}
	sequence.reset()
	sequence.press(target, now, 800*time.Millisecond)
	if !sequence.press(target, now.Add(700*time.Millisecond), 800*time.Millisecond) {
		t.Fatal("configured interval was replaced by default")
	}
	d := NewDraftState("draft")
	stopped, edits := 0, 0
	m := Model{Draft: &d.Text, State: d, TextInput: d.Client, UseTextInputClient: true, Snapshot: session.ViewState{ClientReady: true, StopSequenceMS: 500, SelectedID: "A", StopTurnSeq: 7, StopGeneration: 2, Items: []session.Item{{SessionID: "A", Running: true}}}, OnStop: func() { stopped++ }, OnCancelEdit: func() { edits++ }}
	escape := func(mods ui.Modifiers, repeat bool) bool {
		return draftInputEvent(d.Client, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEscape, Mods: mods, Repeat: repeat})
	}
	if !escape(0, false) || stopped != 0 || !escape(0, false) || stopped != 1 {
		t.Fatal("native double Escape did not stop once")
	}
	for _, blocked := range []string{"edit", "model", "suggest", "ime", "pending", "idle", "modified", "repeat", "other-key"} {
		d.Client.stop.reset()
		escape(0, false)
		switch blocked {
		case "edit":
			m.Editing = true
		case "model":
			d.ModelCommand = ModelCommandState{Open: true, SessionID: "A"}
		case "suggest":
			m.stopSuggestionsOpen = true
		case "ime":
			d.Client.marked = &ui.TextInputRange{}
		case "pending":
			m.StopBlocked = true
		case "idle":
			m.Snapshot.Items[0].Running = false
		}
		switch blocked {
		case "modified":
			escape(ui.Shift, false)
		case "repeat":
			escape(0, true)
		case "other-key":
			draftInputEvent(d.Client, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyLeft})
		default:
			escape(0, false)
		}
		if stopped != 1 {
			t.Fatalf("%s incorrectly stopped", blocked)
		}
		m.Editing = false
		d.ModelCommand.Open = false
		m.stopSuggestionsOpen = false
		d.Client.marked = nil
		m.StopBlocked = false
		m.Snapshot.Items[0].Running = true
		escape(0, false)
		if stopped != 1 {
			t.Fatalf("%s retained earlier press", blocked)
		}
	}
	if edits != 1 {
		t.Fatal("Escape no longer cancels edit")
	}
	d.Client.stop.reset()
	other := "other"
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Children(func() { ui.TextInput(c, &other).Label("Other input"); View(c, m) })
	}, 900, 600)
	if err := tt.Click("Composer draft"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEscape)
	if err := tt.Click("Other input"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEscape)
	tt.Key(0, ui.KeyEscape)
	if stopped != 1 {
		t.Fatal("foreign focus stopped response")
	}
	if err := tt.Click("Composer draft"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEscape)
	if stopped != 1 {
		t.Fatal("focus loss retained first press")
	}
	tt.Key(0, ui.KeyEscape)
	if stopped != 2 || d.Text != "draft" {
		t.Fatal("focused input stop changed draft or failed")
	}
}

func TestStopSequenceRegions(t *testing.T) {
	d := NewDraftState("draft")
	stopped, approvals := 0, 0
	m := Model{Snapshot: session.ViewState{ClientReady: true, StopSequenceMS: 500, SelectedID: "A", StopTurnSeq: 1, Items: []session.Item{{SessionID: "A", Running: true}}}, OnStop: func() { stopped++ }}
	editor := "text"
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Children(func() {
			ui.Button(c, "Outside region")
			region := ui.Column(c)
			BindStopRegion(region, d.Client, m, true)
			region.Children(func() {
				ui.Button(c, "Transcript action")
				ui.TextInput(c, &editor).Label("Transcript editor")
				ui.Box(c).Size(120, 30).Focusable().Label("Approval owner").HandleInput(func(ev ui.InputEvent) bool {
					if ev.Kind == ui.InputKeyDown && ev.Key == ui.KeyEscape {
						approvals++
						return true
					}
					return false
				})
				nested := ui.Column(c)
				BindStopRegion(nested, d.Client, m, false)
				nested.Children(func() { ui.Button(c, "Composer control") })
			})
		})
	}, 900, 600)
	click := func(label string) {
		t.Helper()
		if err := tt.Click(label); err != nil {
			t.Fatal(err)
		}
	}
	pair := func() { tt.Key(0, ui.KeyEscape); tt.Key(0, ui.KeyEscape) }
	click("Transcript action")
	pair()
	if stopped != 1 {
		t.Fatal("chat region double Esc unavailable")
	}
	click("Outside region")
	pair()
	if stopped != 1 {
		t.Fatal("outside control stopped response")
	}
	click("Approval owner")
	pair()
	if stopped != 1 || approvals != 2 {
		t.Fatal("region stole approval Escape")
	}
	click("Transcript editor")
	tt.Compose("候选", 2)
	pair()
	if stopped != 1 {
		t.Fatal("region stole IME Escape")
	}
	click("Transcript action")
	tt.Key(0, ui.KeyEscape)
	click("Composer control")
	tt.Key(0, ui.KeyEscape)
	if stopped != 1 {
		t.Fatal("different regions paired")
	}
	tt.Key(0, ui.KeyEscape)
	if stopped != 2 {
		t.Fatal("composer control region unavailable")
	}
	click("Transcript action")
	tt.Key(0, ui.KeyEscape)
	tt.Key(ui.Shift, ui.KeyEscape)
	tt.Key(0, ui.KeyEscape)
	if stopped != 2 {
		t.Fatal("modified Escape preserved sequence")
	}
	tt.Key(0, ui.KeyEscape)
	if stopped != 3 {
		t.Fatal("fresh chat sequence failed")
	}
}
