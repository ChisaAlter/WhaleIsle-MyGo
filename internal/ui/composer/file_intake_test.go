package composer

import (
	"context"
	"errors"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

func TestFileIntakePreservesEditingUntilExplicitApply(t *testing.T) {
	d := NewDraftState("😀 old tail")
	d.Client.SetSelectionUTF16(3, 6)
	ctx, cancel := context.WithCancel(context.Background())
	op := d.BeginFileIntake([]string{"fixture.txt"}, true, cancel)
	if op == nil || d.Preparing != 1 {
		t.Fatal("preparation not owned")
	}
	d.Client.ReplaceText(nil, "new")
	before := d.Text
	if d.CompleteFileIntake(op, nil) || d.Text != before || !op.Ready || ctx.Err() == nil {
		t.Fatal("background preparation changed newer edit or kept worker active")
	}
	d.Client.SetSelectionUTF16(ui.UTF16Len(d.Text), ui.UTF16Len(d.Text))
	indices, applied := d.ApplyFileIntake(op, []string{"@fixture.txt"}, []session.ComposerAttachment{{Path: "image.png"}})
	if !applied || d.Text != before+" @fixture.txt " || len(indices) != 1 || d.Preparing != 0 || d.FileIntake() != nil {
		t.Fatalf("apply: text=%q state=%+v", d.Text, d)
	}
	if !d.Client.Undo() || d.Text != before {
		t.Fatal("file reference not a single undoable edit")
	}
}

func TestFileIntakeOriginalSelectionCancelAndLateResult(t *testing.T) {
	d := NewDraftState("replace this")
	d.Client.SetSelectionUTF16(0, 7)
	_, cancel := context.WithCancel(context.Background())
	op := d.BeginFileIntake([]string{"fixture"}, false, cancel)
	if !d.CompleteFileIntake(op, nil) {
		t.Fatal("untouched draft not ready")
	}
	if _, ok := d.ApplyFileIntake(op, []string{"@file.txt"}, nil); !ok || d.Text != "@file.txt this" {
		t.Fatal(d.Text)
	}
	ctx, cancel := context.WithCancel(context.Background())
	op = d.BeginFileIntake([]string{"blocked"}, true, cancel)
	d.CancelFileIntake()
	if ctx.Err() == nil || d.Preparing != 0 {
		t.Fatal("cancel did not release its ownership")
	}
	if d.CompleteFileIntake(op, nil) || d.FileIntake() != nil {
		t.Fatal("late completion resurrected cancelled batch")
	}
}

func TestFileIntakeRecoveryRetainsPathsWithoutRestartingWork(t *testing.T) {
	d := NewDraftState("keep")
	_, cancel := context.WithCancel(context.Background())
	op := d.BeginFileIntake([]string{"fixture-one", "fixture-two"}, true, cancel)
	d.CompleteFileIntake(op, errors.New("file unavailable"))
	restored, err := RestoreDraftChecked(d.SaveDraft())
	if err != nil {
		t.Fatal(err)
	}
	pending := restored.FileIntake()
	if pending == nil || pending.Busy || pending.Ready || !pending.RegularOnly || len(pending.Paths) != 2 || restored.Text != "keep" || restored.Preparing != 1 {
		t.Fatalf("pending recovery lost: %+v", pending)
	}
	restored.CancelFileIntake()
	if restored.Preparing != 0 {
		t.Fatal("restored batch cannot be dismissed")
	}
}

func TestFileIntakeNeverConsumesNewIMEComposition(t *testing.T) {
	d := NewDraftState("keep")
	_, cancel := context.WithCancel(context.Background())
	op := d.BeginFileIntake([]string{"fixture"}, true, cancel)
	d.Client.SetMarkedText(nil, "拼", ui.TextInputRange{Start: 1, End: 1})
	before := d.Text
	if d.CompleteFileIntake(op, nil) {
		t.Fatal("new composition treated as unchanged")
	}
	if _, ok := d.ApplyFileIntake(op, []string{"@file"}, nil); ok || d.Text != before {
		t.Fatal("composition overwritten")
	}
	if _, marked := d.Client.MarkedRange(); !marked {
		t.Fatal("composition removed")
	}
	d.CancelFileIntake()
}
