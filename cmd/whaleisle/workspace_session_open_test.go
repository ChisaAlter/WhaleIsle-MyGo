package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/egoist/mygo"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
)

type fakeWorkspaceSessionTarget struct {
	valid    bool
	selected string
	reported string
}

func (f *fakeWorkspaceSessionTarget) Create(context.Context) (string, error) { return "created", nil }
func (f *fakeWorkspaceSessionTarget) Select(id string) bool {
	if !f.valid {
		return false
	}
	f.valid = false
	f.selected = id
	return true
}
func (f *fakeWorkspaceSessionTarget) Current() bool { return f.valid }
func (f *fakeWorkspaceSessionTarget) ReportFailure(err error) bool {
	if !f.valid {
		return false
	}
	f.reported = err.Error()
	return true
}

func workspaceSessionOpenFixture() (*workspaceSessionOpen, *fakeWorkspaceSessionTarget, *composer.DraftState) {
	draft := composer.NewDraftState("original draft")
	store := session.New()
	win := &mygo.Window{}
	s := &mainState{sessions: store, win: win, composerDrafts: map[string]*composer.DraftState{"source": draft}}
	s.workspaceSessionOpen.generation = 3
	target := &fakeWorkspaceSessionTarget{valid: true}
	op := &workspaceSessionOpen{state: s, store: store, win: win, target: target, generation: 3,
		sourceID: "source", sourceDraft: draft, sourceSnapshot: append(json.RawMessage(nil), draft.SaveDraft()...),
		sourceRevision: workspaceSessionDraftRevision(draft), sourceEditSeq: draft.EditSeq, sourceSending: draft.Sending,
		sourcePreparing: draft.Preparing, sourceReply: draft.GroupReply}
	return op, target, draft
}

func TestWorkspaceSessionOpenProtectsDraftsAndNewerOwners(t *testing.T) {
	for _, change := range []string{"none", "new-gesture", "new-store", "new-window", "host-generation", "quitting", "source-replaced", "source-text", "source-aba", "source-attachment", "source-edit", "source-send", "source-preparing", "new-target-draft", "target-owner", "new-panel", "panel-aba"} {
		t.Run(change, func(t *testing.T) {
			op, target, draft := workspaceSessionOpenFixture()
			s := op.state
			switch change {
			case "new-gesture":
				s.workspaceSessionOpen.generation++
			case "new-store":
				s.sessions = session.New()
			case "new-window":
				s.win = &mygo.Window{}
			case "host-generation":
				s.bootStatus.Generation++
			case "quitting":
				s.quitting = true
			case "source-replaced":
				s.composerDrafts["source"] = composer.NewDraftState("replacement")
			case "source-text":
				draft.Text = "newer text"
			case "source-aba":
				draft.Text = "newer text"
				draft.Client.Sync()
				draft.Text = "original draft"
				draft.Client.Sync()
			case "source-attachment":
				draft.Attachments = append(draft.Attachments, session.ComposerAttachment{Path: "draft.png"})
			case "source-edit":
				draft.EditSeq = 4
			case "source-send":
				draft.Sending = true
			case "source-preparing":
				draft.Preparing = 1
			case "new-target-draft":
				s.composerDrafts["created"] = composer.NewDraftState("keep this target")
			case "target-owner":
				target.valid = false
			case "new-panel":
				s.marketOpen = true
				s.assistantConversationGeneration++
			case "panel-aba":
				s.assistantConversationGeneration += 2
			}
			selected := op.finish("created", nil)
			if selected != (change == "none") || (target.selected == "created") != (change == "none") {
				t.Fatalf("change %s selected=%v target=%q", change, selected, target.selected)
			}
			if change == "none" && (s.composerDrafts["source"] != draft || draft.Text != "original draft") {
				t.Fatal("source draft moved or overwritten")
			}
			if change == "new-target-draft" && s.composerDrafts["created"].Text != "keep this target" {
				t.Fatal("destination draft overwritten")
			}
		})
	}
}

func TestWorkspaceSessionOpenReportsOnlyOwnedErrors(t *testing.T) {
	op, target, draft := workspaceSessionOpenFixture()
	if op.finish("", errors.New("failed")) || target.reported != "failed" || draft.Text != "original draft" {
		t.Fatal("current failure lost or draft changed")
	}
	op, target, _ = workspaceSessionOpenFixture()
	op.state.workspaceSessionOpen.generation++
	if op.finish("", errors.New("late")) || target.reported != "" {
		t.Fatal("stale error replaced current status")
	}
}

func TestWorkspaceAdoptionCarriesBlankDraftOnlyAfterSelection(t *testing.T) {
	op, target, draft := workspaceSessionOpenFixture()
	draft.Attachments = []session.ComposerAttachment{{Path: "attachment.png"}}
	op.sourceSnapshot = append(json.RawMessage(nil), draft.SaveDraft()...)
	op.carryDraft = true
	if !op.finish("created", nil) || target.selected != "created" {
		t.Fatal("owned adoption was not selected")
	}
	if op.state.composerDrafts["created"] != draft || op.state.composerDrafts["source"] != nil || draft.Text != "original draft" || len(draft.Attachments) != 1 || draft.TargetID != "created" {
		t.Fatal("blank draft or attachment was not carried to the selected identity")
	}
	op, _, draft = workspaceSessionOpenFixture()
	op.carryDraft = true
	if op.finish("", errors.New("creation failed")) || op.state.composerDrafts["source"] != draft || draft.Text != "original draft" {
		t.Fatal("failure moved the source draft")
	}
}

func TestWorkspaceSessionDraftExclusion(t *testing.T) {
	if workspaceSessionDraftOccupied(nil) || workspaceSessionDraftOccupied(composer.NewDraftState("")) {
		t.Fatal("empty draft excluded")
	}
	for _, change := range []string{"text", "attachment", "editing", "sending", "preparing", "reply"} {
		d := composer.NewDraftState("")
		switch change {
		case "text":
			d.Text = "text"
		case "attachment":
			d.Attachments = []session.ComposerAttachment{{Path: "file"}}
		case "editing":
			d.EditSeq = 3
		case "sending":
			d.Sending = true
		case "preparing":
			d.Preparing = 1
		case "reply":
			d.GroupReply = &composer.GroupReplyDraft{ThreadID: "thread"}
		}
		if !workspaceSessionDraftOccupied(d) {
			t.Fatalf("excluded %s draft was reused", change)
		}
	}
}
