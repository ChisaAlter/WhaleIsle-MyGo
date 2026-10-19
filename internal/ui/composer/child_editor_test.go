package composer

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/ui/i18n"
)

func TestChildEditorDedicatedSend(t *testing.T) {
	d := NewDraftState("/model")
	var sent string
	disabled := false
	tt := ui.NewTester(func(c *ui.Context) {
		ChildEditor(c, ChildEditorModel{Draft: d, Disabled: disabled, OnSend: func(s string) { sent = s }})
	}, 600, 300)
	if err := tt.Click("Composer draft"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEnter)
	if sent != "/model" || d.ModelCommand.Open {
		t.Fatalf("borrowed root command: %q %+v", sent, d.ModelCommand)
	}
	sent = ""
	disabled = true
	tt.Frame()
	_ = tt.Click(i18n.T(i18n.KeyComposerSend))
	if sent != "" {
		t.Fatal("disabled child sent")
	}
	if tt.HasText("模型与思考强度 / Model and reasoning effort") {
		t.Fatal("parent controls rendered")
	}
}
func TestChildEditorMultilineUndo(t *testing.T) {
	d := NewDraftState("line")
	tt := ui.NewTester(func(c *ui.Context) { ChildEditor(c, ChildEditorModel{Draft: d}) }, 600, 300)
	_ = tt.Click("Composer draft")
	d.Client.SetSelectionUTF16(4, 4)
	tt.Key(ui.Shift, ui.KeyEnter)
	if d.Text != "line\n" {
		t.Fatalf("not multiline %q", d.Text)
	}
	if !d.Client.Undo() || d.Text != "line" {
		t.Fatal("native undo lost")
	}
}
