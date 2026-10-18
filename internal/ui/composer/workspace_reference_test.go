package composer

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/filelink"
)

func TestWorkspaceReferenceInsertionUsesCaretAndUndo(t *testing.T) {
	d := NewDraftState("😀 before after")
	d.Attachments = []session.ComposerAttachment{{Path: "keep.png"}}
	at := ui.UTF16Len("😀 before")
	d.Client.SetSelectionUTF16(at, at)
	ref := filelink.Reference{BindingID: "host/session", SessionID: "session", WorkspaceRoot: "/remote", Path: "my file.go"}
	if err := InsertWorkspaceReference(d, ref, "host/session", "session", "/remote"); err != nil {
		t.Fatal(err)
	}
	if d.Text != "😀 before [my file.go](my%20file.go) after" || len(d.Attachments) != 1 || !d.EditFocus {
		t.Fatalf("draft = %#v", d)
	}
	if !d.Client.Undo() || d.Text != "😀 before after" {
		t.Fatal("reference insertion not undoable")
	}
}
func TestWorkspaceReferenceRejectsStaleSendingAndIME(t *testing.T) {
	ref := filelink.Reference{BindingID: "host/session", SessionID: "session", WorkspaceRoot: "/remote", Path: "a.go"}
	d := NewDraftState("keep")
	if err := InsertWorkspaceReference(d, ref, "other-host/session", "session", "/remote"); err == nil || d.Text != "keep" {
		t.Fatal("stale Host accepted")
	}
	d.Sending = true
	if err := InsertWorkspaceReference(d, ref, "host/session", "session", "/remote"); err == nil || d.Text != "keep" {
		t.Fatal("sending draft changed")
	}
	d.Sending = false
	d.Client.SetMarkedText(nil, "中", ui.TextInputRange{Start: 1, End: 1})
	before := d.Text
	if err := InsertWorkspaceReference(d, ref, "host/session", "session", "/remote"); err == nil || d.Text != before {
		t.Fatal("IME composition changed")
	}
}
