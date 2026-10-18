package composer

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/store/session"
)

func TestMessageEditFixture(t *testing.T) {
	d := NewDraftState("saved draft")
	d.Attachments = []session.ComposerAttachment{{Path: "saved.png"}}
	saved := string(d.SaveDraft())
	if !d.BeginEdit("s", 2, "old") || d.BeginEdit("s", 2, "again") {
		t.Fatal("edit admission")
	}
	if d.Text != "old" || len(d.Attachments) != 0 || string(d.SaveDraft()) != saved {
		t.Fatal("stashed draft not preserved")
	}
	d.Text = "/new"
	d.Error = "retry"
	if handleSlashSend(Model{Editing: true}, d.Text) {
		t.Fatal("edit claimed command")
	}
	d.Client.SetMarkedText(nil, "中", ui.TextInputRange{})
	m := Model{Editing: true, OnCancelEdit: d.EndEdit}
	if draftInputEvent(d.Client, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEscape}) || d.EditSeq == 0 {
		t.Fatal("IME escape canceled edit")
	}
	d.Client.SetSelectionUTF16(0, 0)
	if !draftInputEvent(d.Client, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEscape}) {
		t.Fatal("escape ignored")
	}
	if d.Text != "saved draft" || len(d.Attachments) != 1 || d.EditSeq != 0 || d.Error != "" {
		t.Fatal("cancel did not restore")
	}
	d.BeginEdit("s", 2, "old")
	d.Sending = true
	d.EndEdit()
	if d.EditSeq == 0 {
		t.Fatal("pending request canceled")
	}
	d.Sending = false
	d.EndEdit()
	if string(d.SaveDraft()) != saved {
		t.Fatal("success lost draft")
	}
}
