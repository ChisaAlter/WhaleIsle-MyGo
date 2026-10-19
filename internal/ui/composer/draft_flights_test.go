package composer

import (
	"errors"
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/store/session"
)

func TestDetachedDraftsPreserveNewerInputAndFailure(t *testing.T) {
	d := NewDraftState("first")
	d.Attachments = []session.ComposerAttachment{{Path: "one.txt", Name: "one"}}
	first, ok := d.BeginDetachedSend()
	if !ok || d.Text != "" || len(d.Attachments) != 0 || d.Sending {
		t.Fatal("ordinary send did not detach")
	}
	d.Text = "second"
	second, ok := d.BeginDetachedSend()
	if !ok {
		t.Fatal("second admission blocked")
	}
	d.Text = "newer draft"
	d.Attachments = []session.ComposerAttachment{{Path: "new.txt"}}
	d.CompleteDetachedSend(first.ID, first.Attachments, errors.New("rejected"))
	if d.Text != "newer draft" || len(d.Attachments) != 2 || d.Attachments[0].Path != "one.txt" || d.Attachments[1].Path != "new.txt" || d.FailedSendCount() != 1 {
		t.Fatal("failure overwrote newer draft")
	}
	d.CompleteDetachedSend(second.ID, second.Attachments, nil)
	if d.Text != "newer draft" || len(d.Attachments) != 2 || d.Attachments[0].Path != "one.txt" || d.Attachments[1].Path != "new.txt" {
		t.Fatal("late success cleared newer draft")
	}
	d.RestoreFailedSends()
	if d.Text != "first\n\nnewer draft" || len(d.Attachments) != 2 || d.Attachments[0].Path != "one.txt" {
		t.Fatal("failed payload lost")
	}
	retry, ok := d.BeginDetachedSend()
	if !ok || retry.Text != "first\n\nnewer draft" || d.FailedSendCount() != 0 {
		t.Fatal("restored send not consumed exactly once")
	}
	d.CompleteDetachedSend(first.ID, nil, errors.New("duplicate callback"))
	if d.Text != "" {
		t.Fatal("duplicate settlement restored payload")
	}
}
func TestDetachedFailureOrderAndRecovery(t *testing.T) {
	d := NewDraftState("one")
	one, _ := d.BeginDetachedSend()
	d.Text = "two"
	two, _ := d.BeginDetachedSend()
	d.CompleteDetachedSend(two.ID, two.Attachments, errors.New("two failed"))
	d.CompleteDetachedSend(one.ID, one.Attachments, errors.New("one failed"))
	if d.Text != "one\n\ntwo" {
		t.Fatalf("failed order %q", d.Text)
	}
	d.BeginDetachedSend()
	d.Text = "newer"
	restored, err := RestoreDraftChecked(d.SaveDraft())
	if err != nil || restored.Text != "newer" || restored.FailedSendCount() != 1 {
		t.Fatalf("recovery: %+v %v", restored, err)
	}
	restored.RestoreFailedSends()
	if restored.Text != "one\n\ntwo\n\nnewer" {
		t.Fatal(restored.Text)
	}
}
func TestEnterDuringOrdinaryAdmission(t *testing.T) {
	d := NewDraftState("next")
	calls := 0
	m := Model{Draft: &d.Text, State: d, Snapshot: session.ViewState{PromptBusy: true}, OnSubmit: func() { calls++ }}
	if !draftInputEvent(d.Client, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEnter}) || calls != 1 {
		t.Fatal("ordinary admission swallowed Enter")
	}
	d.Sending = true
	draftInputEvent(d.Client, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEnter})
	if calls != 1 {
		t.Fatal("exclusive command/edit allowed concurrent send")
	}
}
