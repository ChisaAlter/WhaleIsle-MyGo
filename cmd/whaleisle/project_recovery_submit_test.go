package main

import (
	"testing"

	projectctl "whaleisle/internal/app/project"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
)

type projectRecoveryDraftTarget struct{ *composer.DraftState }

func (d projectRecoveryDraftTarget) Snapshot() projectctl.LeadDraftSnapshot {
	return projectctl.LeadDraftSnapshot{
		Text:           d.Text,
		HasAttachments: len(d.Attachments) > 0,
		Sending:        d.Sending,
		Preparing:      d.Preparing,
		EditSeq:        d.EditSeq,
		EditSessionID:  d.EditSessionID,
	}
}

func (d projectRecoveryDraftTarget) StageRecoveryText(text string) { d.Text = text }

func TestProjectRecoveryExplicitSend(t *testing.T) {
	d := composer.NewDraftState("")
	calls := 0
	err := projectctl.SubmitRecoveryDraft(projectRecoveryDraftTarget{d}, "verified original Lead request", func() {
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
		if err := projectctl.SubmitRecoveryDraft(projectRecoveryDraftTarget{d}, "recovery", func() { calls++ }); err == nil || calls != 0 {
			t.Fatal("existing draft overwritten or sent")
		}
	}
}
