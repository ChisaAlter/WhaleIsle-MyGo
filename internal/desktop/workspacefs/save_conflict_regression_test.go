package workspacefs

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestSaveConflictReceiptRequiresAnotherExplicitSave(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "draft.txt")
	if err := os.WriteFile(path, []byte("external"), 0600); err != nil {
		t.Fatal(err)
	}
	var writes atomic.Int32
	done := make(chan struct{}, 4)
	q := newSaveQueue("original", time.Millisecond, func(ctx context.Context, base, text string) error {
		return SaveTextChecked(ctx, root, "draft.txt", base, text, func(_ string, text string) error {
			writes.Add(1)
			return os.WriteFile(path, []byte(text), 0600)
		})
	}, func() { done <- struct{}{} })
	defer q.Dispose()
	q.Flush("draft")
	awaitSave(t, done)
	s := q.Snapshot()
	if s.Confirmed != "external" || s.Revision != 1 || !s.Blocked || s.Error == "" || writes.Load() != 0 {
		t.Fatalf("conflict receipt = %+v writes=%d", s, writes.Load())
	}
	q.Observe("newer draft", true)
	q.Observe("newest draft", true)
	if s = q.Snapshot(); s.Pending || s.Running || !s.Blocked || writes.Load() != 0 {
		t.Fatalf("typing retried a conflict: %+v writes=%d", s, writes.Load())
	}
	if disk, err := os.ReadFile(path); err != nil || string(disk) != "external" {
		t.Fatalf("disk=%q err=%v", disk, err)
	}
	q.Flush("newest draft")
	awaitSave(t, done)
	if s = q.Snapshot(); s.Confirmed != "newest draft" || s.Revision != 2 || s.Blocked || s.Error != "" || writes.Load() != 1 {
		t.Fatalf("explicit retry = %+v writes=%d", s, writes.Load())
	}
}

func TestDisposedConflictCannotPublishIntoReplacementQueue(t *testing.T) {
	started, release, returned := make(chan struct{}), make(chan struct{}), make(chan struct{})
	old := newSaveQueue("old", time.Hour, func(context.Context, string, string) error {
		close(started)
		<-release
		close(returned)
		return &SaveConflictError{Current: "late external"}
	}, nil)
	old.Flush("old draft")
	awaitSave(t, started)
	old.Dispose()
	next := newSaveQueue("new owner", time.Hour, func(context.Context, string, string) error { t.Error("replacement queue wrote"); return nil }, nil)
	defer next.Dispose()
	close(release)
	awaitSave(t, returned)
	if s := next.Snapshot(); s.Confirmed != "new owner" || s.Revision != 0 || s.Blocked || s.Pending {
		t.Fatalf("late conflict replaced new baseline: %+v", s)
	}
	if s := old.Snapshot(); s.Confirmed != "old" || s.Revision != 0 {
		t.Fatalf("disposed receipt published: %+v", s)
	}
}
