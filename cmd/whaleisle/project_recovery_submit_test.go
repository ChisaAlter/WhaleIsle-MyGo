package main

import (
	"testing"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
)

func TestProjectRecoveryExplicitSend(t *testing.T) {
	d := composer.NewDraftState("")
	calls := 0
	err := submitProjectRecoveryDraft(d, "verified original Lead request", func() {
		calls++
		if d.Text != "verified original Lead request" {
			t.Fatal(d.Text)
		}
		d.Error = "Host rejected"
	})
	if err != nil || calls != 1 {
		t.Fatalf("explicit recovery click must submit once: calls=%d err=%v", calls, err)
	}
	if d.Text != "verified original Lead request" || d.Error != "Host rejected" {
		t.Fatal("failed send must keep standard draft", d)
	}
}
func TestProjectRecoveryPreservesExistingDraft(t *testing.T) {
	for _, d := range []*composer.DraftState{composer.NewDraftState("existing"), composer.NewDraftState("")} {
		if d.Text == "" {
			d.Attachments = []session.ComposerAttachment{{Name: "attachment"}}
		}
		calls := 0
		if err := submitProjectRecoveryDraft(d, "recovery", func() { calls++ }); err == nil || calls != 0 {
			t.Fatal("existing draft overwritten or sent")
		}
	}
}
