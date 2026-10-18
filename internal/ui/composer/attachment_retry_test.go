package composer

import (
	"errors"
	"testing"
	"whaleisle/internal/store/session"
)

func TestAttachmentPreparationRetainsIndividualFailureAndRetries(t *testing.T) {
	d := NewDraftState("keep text")
	d.Attachments = []session.ComposerAttachment{{Path: "a.txt"}, {Path: "b.txt"}}
	a, ok := d.BeginAttachmentPreparation(0, false)
	if !ok || d.Preparing != 1 {
		t.Fatal("upload did not begin")
	}
	d.CompleteAttachmentPreparation(0, a, a, errors.New("offline"))
	if d.Attachments[0].PrepareError != "offline" || d.Attachments[1].PrepareError != "" {
		t.Fatal("failure not file-scoped")
	}
	if _, ok = d.BeginAttachmentPreparation(1, true); ok {
		t.Fatal("retried nonfailed file")
	}
	a, ok = d.BeginAttachmentPreparation(0, true)
	if !ok {
		t.Fatal("retry refused")
	}
	a.SessionID = "session-a"
	d.CompleteAttachmentPreparation(0, d.Attachments[0], a, nil)
	if d.Preparing != 0 || d.Attachments[0].PrepareError != "" || d.Attachments[0].SessionID != "session-a" || d.Text != "keep text" {
		t.Fatalf("bad retry result: %+v", d)
	}
}
func TestAttachmentRetryCannotReplaceRemovedOrBusyDraft(t *testing.T) {
	d := NewDraftState("")
	d.Attachments = []session.ComposerAttachment{{Path: "a", PrepareError: "failed"}}
	d.Sending = true
	if _, ok := d.BeginAttachmentPreparation(0, true); ok {
		t.Fatal("retry while sending")
	}
	d.Sending = false
	a, _ := d.BeginAttachmentPreparation(0, true)
	if _, ok := d.BeginAttachmentPreparation(0, true); ok {
		t.Fatal("duplicate retry")
	}
	d.Attachments = nil
	d.CompleteAttachmentPreparation(0, a, a, nil)
	if d.Preparing != 0 || len(d.Attachments) != 0 {
		t.Fatal("removed attachment restored")
	}
}
