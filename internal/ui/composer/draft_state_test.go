package composer

import (
	"testing"
	"whaleisle/internal/store/session"
)

func TestDraftRecoveryChecked(t *testing.T) {
	d := NewDraftState("hello\nworld")
	d.Attachments = []session.ComposerAttachment{{Path: "/local/notes.txt", Name: "notes.txt", ImageData: "must not persist"}}
	restored, err := RestoreDraftChecked(d.SaveDraft())
	if err != nil || restored.Text != d.Text || len(restored.Attachments) != 1 || restored.Attachments[0].ImageData != "" {
		t.Fatalf("restore failed: %v", err)
	}
	if _, err = RestoreDraftChecked([]byte(`{"Files":[]}`)); err == nil {
		t.Fatal("missing draft text accepted")
	}
	if _, err = RestoreDraftChecked([]byte(`{broken`)); err == nil {
		t.Fatal("invalid JSON accepted")
	}
}
