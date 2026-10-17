package composer

import (
	"context"
	"errors"
	"testing"
	"whaleisle/internal/store/session"
)

func TestAttachmentQueueRemovalAndLateCompletion(t *testing.T) {
	d := NewDraftState("keep")
	d.Attachments = []session.ComposerAttachment{{Path: "same"}, {Path: "second"}}
	a, _ := d.BeginAttachmentPreparation(0, false)
	b, _ := d.BeginAttachmentPreparation(1, false)
	ctx, cancel := context.WithCancel(context.Background())
	d.SetAttachmentCancel(a, cancel)
	if !d.RemoveAttachment(0) || ctx.Err() == nil || d.Preparing != 1 {
		t.Fatal("remove did not cancel only its upload")
	}
	d.Attachments = append(d.Attachments, session.ComposerAttachment{Path: "same"})
	d.CompleteAttachmentPreparation(0, a, a, nil)
	if d.Preparing != 1 || d.Attachments[1].SessionID != "" {
		t.Fatal("late result touched new upload")
	}
	b.SessionID = "saved"
	d.CompleteAttachmentPreparation(1, b, b, nil)
	if d.Preparing != 0 || d.Attachments[0].SessionID != "saved" {
		t.Fatal("shifted upload lost")
	}
}
func TestAttachmentQueueFailedSendRequiresRetry(t *testing.T) {
	d := NewDraftState("keep")
	d.Attachments = []session.ComposerAttachment{{Path: "a"}}
	a, _ := d.BeginAttachmentPreparation(0, false)
	d.CompleteAttachmentPreparation(0, a, a, errors.New("offline"))
	if _, ok := d.BeginDetachedSend(); ok {
		t.Fatal("failed attachment admitted")
	}
	a, ok := d.BeginAttachmentPreparation(0, true)
	if !ok || a.PrepareError != "" {
		t.Fatal("explicit retry not cleared")
	}
}
