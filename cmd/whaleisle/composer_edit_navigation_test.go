package main

import (
	"testing"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
)

func TestComposerEditNavigationLifecycle(t *testing.T) {
	for _, tc := range []struct {
		name      string
		rendered  bool
		visibleID string
		sending   bool
		keepEdit  bool
	}{
		{"still visible", true, "A", false, true},
		{"session switch", true, "B", false, false},
		{"global page teardown", false, "A", false, false},
		{"sending across session switch", true, "B", true, true},
		{"sending across global page", false, "A", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := composer.NewDraftState("saved draft")
			d.Attachments = []session.ComposerAttachment{{Path: "saved.png", Name: "saved.png"}}
			saved := string(d.SaveDraft())
			if !d.BeginEdit("A", 7, "original message") {
				t.Fatal("could not begin edit")
			}
			d.Text = "revised message"
			d.Sending = tc.sending
			other := composer.NewDraftState("other draft")
			s := &mainState{composerDrafts: map[string]*composer.DraftState{"A": d, "B": other}}
			// Repeated frames must preserve both the active edit and restored drafts.
			for range 2 {
				s.observeComposerEdit(tc.rendered, tc.visibleID)
			}
			if tc.keepEdit {
				if d.EditSeq != 7 || d.EditSessionID != "A" || d.TargetID != "A" || d.Text != "revised message" || len(d.Attachments) != 0 || d.Sending != tc.sending {
					t.Fatalf("visible or in-flight edit lost its owner/draft: %+v", d)
				}
			} else if d.EditSeq != 0 || d.EditSessionID != "" || d.TargetID != "" || d.Text != "saved draft" || len(d.Attachments) != 1 || d.Attachments[0].Path != "saved.png" {
				t.Fatalf("unmounted editor did not restore its stash: %+v", d)
			}
			if string(d.SaveDraft()) != saved || other.Text != "other draft" {
				t.Fatal("navigation changed the persisted stash or another session's draft")
			}
			if s.composerDrafts["A"] != d || s.composerDrafts["B"] != other {
				t.Fatal("navigation replaced a draft owner")
			}
		})
	}
}
