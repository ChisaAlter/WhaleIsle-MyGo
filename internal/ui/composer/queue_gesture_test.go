package composer

import (
	"testing"
	"whaleisle/internal/store/session"
)

func TestEmptyComposerQueueGestureBoundaries(t *testing.T) {
	d := &DraftState{Text: " \n\t"}
	if !d.CanSteerPendingQueue("A", true) {
		t.Fatal("explicit empty accelerated gesture rejected")
	}
	if d.CanSteerPendingQueue("A", false) || d.CanSteerPendingQueue("", true) {
		t.Fatal("plain Enter or home admitted")
	}
	d.Text = "message"
	if d.CanSteerPendingQueue("A", true) {
		t.Fatal("message turned into queue action")
	}
	d.Text = ""
	d.Attachments = []session.ComposerAttachment{{Name: "pending"}}
	if d.CanSteerPendingQueue("A", true) {
		t.Fatal("attachment draft turned into queue action")
	}
	d.Attachments = nil
	d.EditSeq = 1
	if d.CanSteerPendingQueue("A", true) {
		t.Fatal("edit turned into queue action")
	}
	d.EditSeq = 0
	d.Preparing = 1
	if d.CanSteerPendingQueue("A", true) {
		t.Fatal("uploading draft admitted")
	}
	d.Preparing = 0
	d.Sending = true
	if d.CanSteerPendingQueue("A", true) {
		t.Fatal("submitting draft admitted")
	}
	d.Sending = false
	d.ModelCommand = ModelCommandState{Open: true, SessionID: "A"}
	if d.CanSteerPendingQueue("A", true) {
		t.Fatal("command popup stolen")
	}
}
