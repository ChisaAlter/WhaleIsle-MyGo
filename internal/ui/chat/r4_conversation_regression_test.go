package chat

import (
	"context"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestTrajectoryFoldsKeepTimelineAndRevealSelectedRecord(t *testing.T) {
	selected := ""
	foldTurns, foldCalls, actualTime := true, true, true
	messages := []session.Message{
		{ID: "user", TurnSeq: 1, Role: "user", Kind: session.MessageKindText, Body: "request"},
		{ID: "parent", TurnSeq: 1, Kind: session.MessageKindTool, EventType: "tool/call", Tool: &session.ToolCallView{CallID: "parent-call", Name: "read", State: "running"}, TrajectoryTime: &session.TrajectoryTiming{OperationID: "parent", StartedAt: trajectoryTestTime(1000), CompletedAt: trajectoryTestTime(2000)}},
		{ID: "child", TurnSeq: 1, Kind: session.MessageKindTool, EventType: "tool/call", Tool: &session.ToolCallView{CallID: "child-call", ParentCallID: "parent-call", Name: "grep", State: "running"}, TrajectoryTime: &session.TrajectoryTiming{OperationID: "child", StartedAt: trajectoryTestTime(1500), CompletedAt: trajectoryTestTime(2500)}},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		TrajectoryView(c, TrajectoryModel{Snapshot: session.ViewState{SelectedID: t.Name(), Messages: messages}, SelectedID: &selected, TurnsCollapsed: &foldTurns, CallsCollapsed: &foldCalls, ActualTime: &actualTime})
	}, 1100, 650)
	if _, ok := tt.Find("Trajectory row child"); ok {
		t.Fatal("initial turn did not fold its ledger rows")
	}
	if _, ok := tt.Find("Trajectory span child"); !ok {
		t.Fatal("ledger fold removed the complete timeline operation")
	}
	if err := tt.Click("Trajectory span child"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if selected != "child" {
		t.Fatalf("timeline selected=%q", selected)
	}
	if _, ok := tt.Find("Trajectory row child"); !ok {
		t.Fatal("timeline navigation did not reveal the folded record")
	}
	if err := tt.Click("Parent tool"); err != nil {
		t.Fatalf("complete-row relationship missing: %v; %q", err, tt.Texts())
	}
	tt.Frame()
	if selected != "parent" {
		t.Fatalf("parent link selected=%q", selected)
	}
	if _, ok := tt.Find("Trajectory row parent"); !ok {
		t.Fatal("parent link did not reveal the complete-row target")
	}
}

func TestQueueImageScopeAndCorruptImageRetry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "same-path.png")
	if err := os.WriteFile(path, []byte("broken image"), 0600); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	load := func(context.Context, string) (string, error) {
		if calls.Add(1) > 1 {
			f, err := os.Create(path)
			if err != nil {
				return "", err
			}
			err = png.Encode(f, image.NewRGBA(image.Rect(0, 0, 4, 3)))
			if closeErr := f.Close(); err == nil {
				err = closeErr
			}
			if err != nil {
				return "", err
			}
		}
		return path, nil
	}
	model := PlanBoardModel{Board: session.PlanBoard{QueueActions: session.QueueActionState{Scope: t.Name() + ":child-a"}}, LoadQueueImage: load}
	item := session.PlanQueueItem{Attachments: []session.QueueAttachment{{Kind: "image", AttachmentID: "same-opaque-id", Width: 4, Height: 3}}}
	tt := ui.NewTester(func(c *ui.Context) { paintQueueAttachments(c, model, item, c.Theme().Text, c.Theme().TextMuted) }, 400, 180)
	waitFor := func(done func() bool) {
		t.Helper()
		deadline := time.Now().Add(2 * time.Second)
		for !done() {
			if time.Now().After(deadline) {
				t.Fatalf("queue image did not settle: calls=%d texts=%q", calls.Load(), tt.Texts())
			}
			time.Sleep(time.Millisecond)
			tt.Frame()
		}
	}
	retry := i18n.T(i18n.KeyChatImageRetry)
	waitFor(func() bool { return tt.HasText(retry) })
	if calls.Load() != 1 {
		t.Fatal("failed image automatically replayed")
	}
	if err := tt.Click(retry); err != nil {
		t.Fatal(err)
	}
	waitFor(func() bool { return calls.Load() == 2 && !tt.HasText(retry) && loadChatBitmap(path) != nil })
	model.Board.QueueActions.Scope = t.Name() + ":child-b"
	tt.Frame()
	waitFor(func() bool { return calls.Load() == 3 })
}
