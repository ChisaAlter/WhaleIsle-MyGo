package composer

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// TestSendButtonFollowsDraftText pins the empty-draft disabled state and the
// transition to an enabled send after text exists.
func TestSendButtonFollowsDraftText(t *testing.T) {
	d := NewDraftState("")
	sent := 0
	m := Model{
		Draft: &d.Text, State: d, TextInput: d.Client, UseTextInputClient: true,
		Snapshot: session.ViewState{SelectedID: "A"},
		OnSubmit: func() { sent++ },
	}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 900, 620)
	tt.Frame()
	_ = tt.Click(i18n.T(i18n.KeyComposerSend))
	tt.Frame()
	if sent != 0 {
		t.Fatalf("empty draft sent %d messages", sent)
	}
	d.Text = "hello"
	tt.Frame()
	if err := tt.Click(i18n.T(i18n.KeyComposerSend)); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if sent != 1 {
		t.Fatalf("typed draft sent %d messages, want 1", sent)
	}
}
