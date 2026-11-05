package composer

import (
	"context"
	"github.com/egoist/mygo/ui"
	"testing"
	"time"
	"whaleisle/internal/store/session"
)

func TestParityReferenceDiscoveryOffRenderer(t *testing.T) {
	d := NewDraftState("")
	d.Client.input.ReplaceText(nil, "@slow")
	started := make(chan string, 2)
	release := make(chan struct{})
	done := make(chan struct{}, 2)
	m := Model{State: d, Snapshot: session.ViewState{SelectedID: "s", ComposerHostGeneration: 1}, OnQueryReferences: func(ctx context.Context, q string, _ bool) ([]session.ReferenceCandidate, error) {
		started <- q
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-release:
			return []session.ReferenceCandidate{{Kind: "file", Path: q + ".txt"}}, nil
		}
	}, referenceInvalidate: func() { done <- struct{}{} }}
	m.Draft = &d.Text
	m.TextInput = d.Client
	selection := 0
	m.SuggestList = &ui.ListState{Selected: &selection}
	returned := make(chan struct{})
	go func() { d.references.query(m, "slow", false); close(returned) }()
	select {
	case <-returned:
	case <-time.After(time.Second):
		t.Fatal("render path blocked on discovery")
	}
	<-started
	if !handleSuggestKey(d.Client, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEnter}) {
		t.Fatal("pending reference Enter fell through to send")
	}
	d.references.query(m, "new", false)
	<-started
	close(release)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("query did not publish")
	}
	rows := d.references.query(m, "new", false)
	if len(rows) != 1 || rows[0].Path != "new.txt" {
		t.Fatalf("stale reference receipt: %+v", rows)
	}
	d.references.cancelQuery()
}
func TestParityChildAcceleratedAndStop(t *testing.T) {
	d := NewDraftState("child text")
	accelerated := false
	stops := 0
	m := Model{Draft: &d.Text, State: d, TextInput: d.Client, Editing: true, Child: true, Snapshot: session.ViewState{SelectedID: "child", ClientReady: true, StopSequenceMS: 500, StopTurnSeq: 3, Items: []session.Item{{SessionID: "child", Running: true}}}, OnSubmitGesture: func(value bool) { accelerated = value }, OnStop: func() { stops++ }}
	draftInputEvent(d.Client, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEnter, Mods: ui.Ctrl})
	if !accelerated {
		t.Fatal("child chord lost")
	}
	for i := 0; i < 2; i++ {
		draftInputEvent(d.Client, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEscape})
	}
	if stops != 1 {
		t.Fatal("literal child commands incorrectly suppressed double Escape")
	}
}
func TestParityUploadProgressBelongsToAttempt(t *testing.T) {
	d := NewDraftState("")
	d.Attachments = []session.ComposerAttachment{{Path: "one.txt"}}
	a, _ := d.BeginAttachmentPreparation(0, false)
	d.UpdateAttachmentProgress(a, 25, 100)
	if d.Attachments[0].SentBytes != 25 {
		t.Fatal("progress lost")
	}
	d.RemoveAttachment(0)
	d.Attachments = append(d.Attachments, session.ComposerAttachment{Path: "other.txt"})
	b, _ := d.BeginAttachmentPreparation(0, false)
	d.UpdateAttachmentProgress(a, 100, 100)
	if d.Attachments[0].PreparationID != b.PreparationID || d.Attachments[0].TotalBytes != 0 {
		t.Fatal("old upload touched replacement")
	}
}
func TestParityBindingRetirementKeepsAuthoredDraft(t *testing.T) {
	d := NewDraftState("first")
	d.Client.ReplaceText(nil, " second")
	text := d.Text
	revision := d.Client.Revision()
	d.Client.stop.target = stopTarget{sessionID: "old"}
	d.Suggestions.launcher = &commandLauncher{}
	d.RetireBinding()
	if d.Text != text || d.Client.Revision() != revision || d.Client.stop.target.sessionID != "" || d.Suggestions.launcher != nil {
		t.Fatal("binding retirement lost user history or retained old runtime owner")
	}
	if !d.Client.Undo() {
		t.Fatal("undo history was retired with Host binding")
	}
}
