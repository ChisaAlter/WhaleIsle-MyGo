package workspacefs

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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

func TestLocalEditorUnreadRereadRequiresExplicitDirtySnapshot(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "draft.txt")
	unknownDisk := strings.Repeat("unreviewed bytes\n", TextPreviewLimit/16+32)
	if len(unknownDisk) <= TextPreviewLimit {
		t.Fatal("fixture must exceed strict read limit")
	}
	if err := os.WriteFile(path, []byte(unknownDisk), 0600); err != nil {
		t.Fatal(err)
	}
	var writes atomic.Int32
	done := make(chan struct{}, 4)
	q := NewLocalEditorSaveQueue(root, "draft.txt", "previous complete baseline", func(_ string, text string) error {
		writes.Add(1)
		return os.WriteFile(path, []byte(text), 0600)
	}, func() { done <- struct{}{} })
	q.delay = time.Millisecond
	defer q.Dispose()
	q.Observe("retained edited draft", true)
	awaitSave(t, done)
	if s := q.Snapshot(); !s.Blocked || s.Error == "" || s.Confirmed != "previous complete baseline" || s.Revision != 0 || writes.Load() != 0 {
		t.Fatalf("incomplete reread authorized autosave: %+v writes=%d", s, writes.Load())
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != unknownDisk {
		t.Fatalf("automatic save overwrote unknown bytes: size=%d err=%v", len(data), err)
	}
	// Source FilePreview's existing dirty Save is an actual flush gesture. It
	// writes the captured draft, without treating the preview prefix as a baseline.
	q.Flush("retained edited draft")
	awaitSave(t, done)
	if s := q.Snapshot(); s.Blocked || s.Confirmed != "retained edited draft" || writes.Load() != 1 {
		t.Fatalf("explicit dirty save=%+v writes=%d", s, writes.Load())
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != "retained edited draft" {
		t.Fatalf("explicit captured bytes=%q err=%v", data, err)
	}

	if err := os.WriteFile(path, []byte(unknownDisk), 0600); err != nil {
		t.Fatal(err)
	}
	prefix, err := ReadFileTextPreview(path)
	if err != nil || !prefix.Truncated {
		t.Fatalf("prefix=%+v err=%v", prefix, err)
	}
	clean := NewLocalEditorSaveQueue(root, "draft.txt", prefix.Text, func(string, string) error { writes.Add(1); return nil }, nil)
	defer clean.Dispose()
	clean.Flush(prefix.Text)
	if s := clean.Snapshot(); s.Pending || s.Running || s.Revision != 0 || writes.Load() != 1 {
		t.Fatalf("clean prefix authorized write: %+v writes=%d", s, writes.Load())
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != unknownDisk {
		t.Fatal("clean prefix replaced full disk file")
	}
}

func TestNewerTypingCannotInheritQueuedExplicitSaveGesture(t *testing.T) {
	type receipt struct{ kind, text string }
	results := make(chan receipt, 4)
	done := make(chan struct{}, 4)
	started, release := make(chan struct{}), make(chan struct{})
	q := newSaveQueue("base", time.Millisecond, func(_ context.Context, _, text string) error { results <- receipt{"automatic", text}; return nil }, func() { done <- struct{}{} })
	q.persistExplicit = func(_ context.Context, _, text string) error {
		if text == "first" {
			close(started)
			<-release
		}
		results <- receipt{"explicit", text}
		return nil
	}
	defer q.Dispose()
	q.Flush("first")
	awaitSave(t, started)
	q.Flush("captured second gesture")
	q.Observe("newer automatic typing", true)
	close(release)
	awaitSave(t, done)
	awaitSave(t, done)
	first, second := <-results, <-results
	if first != (receipt{"explicit", "first"}) || second != (receipt{"automatic", "newer automatic typing"}) {
		t.Fatalf("save gestures=%+v %+v", first, second)
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
