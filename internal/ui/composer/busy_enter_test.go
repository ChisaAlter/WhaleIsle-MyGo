package composer

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestBusyEnterNativeGestureIMEAndFocus(t *testing.T) {
	d := NewDraftState("draft 😀")
	var gestures []bool
	legacy := 0
	m := Model{Draft: &d.Text, State: d, TextInput: d.Client, UseTextInputClient: true, Snapshot: session.ViewState{SelectedID: "A"}, OnSubmitGesture: func(acc bool) { gestures = append(gestures, acc) }, OnSubmit: func() { legacy++ }}
	key := func(mods ui.Modifiers, repeat bool) bool {
		return draftInputEvent(d.Client, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEnter, Mods: mods, Repeat: repeat})
	}
	for _, mods := range []ui.Modifiers{0, ui.Ctrl, ui.Super} {
		if !key(mods, false) {
			t.Fatal("gesture not handled")
		}
	}
	if len(gestures) != 3 || gestures[0] || !gestures[1] || !gestures[2] || legacy != 0 {
		t.Fatalf("gesture forwarding: %v legacy=%d", gestures, legacy)
	}
	before := d.Text
	key(ui.Alt, false)
	key(ui.Ctrl|ui.Super, false)
	key(ui.Ctrl|ui.Shift, false)
	key(0, true)
	if len(gestures) != 3 || d.Text != before {
		t.Fatal("alternate/repeated chord submitted or edited")
	}
	key(ui.Shift, false)
	if d.Text != before+"\n" || len(gestures) != 3 {
		t.Fatal("Shift newline changed")
	}
	d.Client.SetMarkedText(nil, "候选", ui.TextInputRange{Start: 2, End: 2})
	before = d.Text
	if key(0, false) || len(gestures) != 3 || d.Text != before {
		t.Fatal("IME Enter was claimed")
	}
	d.Client.UnmarkText()
	d.Preparing = 1
	key(0, false)
	d.Preparing = 0
	d.Sending = true
	key(0, false)
	d.Sending = false
	if len(gestures) != 3 {
		t.Fatal("busy editor submitted")
	}
	other := "other"
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Children(func() {
			ui.TextInput(c, &other).Label("Other input")
			View(c, m)
		})
	}, 900, 600)
	if err := tt.Click("Other input"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEnter)
	if len(gestures) != 3 || !tt.Focused("Other input") {
		t.Fatal("composer stole foreign Enter/focus")
	}
	if err := tt.Click("Composer draft"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Ctrl, ui.KeyEnter)
	if len(gestures) != 4 || !gestures[3] {
		t.Fatalf("shipped native Ctrl+Enter: %v", gestures)
	}
	if err := tt.Click(i18n.T(i18n.KeyComposerSend)); err != nil {
		t.Fatal(err)
	}
	if len(gestures) != 5 || gestures[4] {
		t.Fatalf("primary button differs from plain Enter: %v", gestures)
	}
}
