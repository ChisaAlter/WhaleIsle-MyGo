package workspacefs

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func awaitSave(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("save notification timeout")
	}
}

func TestSaveQueueDebouncesAndSerializes(t *testing.T) {
	var mu sync.Mutex
	var writes, expected []string
	firstStarted := make(chan struct{})
	release := make(chan struct{})
	done := make(chan struct{}, 10)
	persist := func(ctx context.Context, base, text string) error {
		mu.Lock()
		writes = append(writes, text)
		expected = append(expected, base)
		count := len(writes)
		mu.Unlock()
		if count == 1 {
			close(firstStarted)
			<-release
		}
		return nil
	}
	q := newSaveQueue("original", 25*time.Millisecond, persist, func() { done <- struct{}{} })
	defer q.Dispose()
	q.Observe("a", true)
	q.Observe("ab", true)
	q.Observe("abc", true)
	awaitSave(t, firstStarted)
	q.Observe("newer", true)
	q.Flush("latest")
	mu.Lock()
	if len(writes) != 1 {
		t.Fatalf("parallel write: %v", writes)
	}
	mu.Unlock()
	close(release)
	for q.Snapshot().Confirmed != "latest" {
		awaitSave(t, done)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(writes) != 2 || writes[0] != "abc" || writes[1] != "latest" || expected[1] != "abc" {
		t.Fatalf("writes=%v expected=%v", writes, expected)
	}
}

func TestSaveQueueFailureRetainsDraftAndStopsAutomaticRetry(t *testing.T) {
	done := make(chan struct{}, 10)
	calls := 0
	q := newSaveQueue("original", time.Millisecond, func(context.Context, string, string) error { calls++; return errors.New("conflict") }, func() { done <- struct{}{} })
	defer q.Dispose()
	q.Flush("draft")
	awaitSave(t, done)
	q.Observe("new draft", true)
	time.Sleep(10 * time.Millisecond)
	s := q.Snapshot()
	if calls != 1 || s.Confirmed != "original" || !s.Blocked || s.Error != "conflict" {
		t.Fatalf("failure state: %+v calls=%d", s, calls)
	}
	q.Flush("new draft")
	awaitSave(t, done)
	if calls != 2 {
		t.Fatalf("manual retry missing: %d", calls)
	}
}

func TestSaveQueueDisposeDoesNotFlushOrPublish(t *testing.T) {
	calls := 0
	q := newSaveQueue("base", time.Hour, func(context.Context, string, string) error { calls++; return nil }, nil)
	q.Observe("draft", true)
	q.Dispose()
	q.Flush("draft")
	if calls != 0 {
		t.Fatal("dispose flushed pending draft")
	}
	started := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan struct{})
	q = newSaveQueue("base", time.Millisecond, func(ctx context.Context, base, text string) error {
		close(started)
		<-release
		close(finished)
		return nil
	}, nil)
	q.Flush("draft")
	awaitSave(t, started)
	q.Dispose()
	close(release)
	awaitSave(t, finished)
	if q.Snapshot().Confirmed != "base" {
		t.Fatal("late completion published after dispose")
	}
}

func TestCheckedSaveConflictAndCancellation(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "file.txt")
	os.WriteFile(path, []byte("external"), 0600)
	calls := 0
	write := func(rel, text string) error { calls++; return os.WriteFile(path, []byte(text), 0600) }
	if err := SaveTextChecked(context.Background(), root, "file.txt", "base", "draft", write); err == nil {
		t.Fatal("external file overwritten")
	}
	if calls != 0 {
		t.Fatal("writer invoked on conflict")
	}
	// A completed old write can be acknowledged without overwriting anything.
	if err := SaveTextChecked(context.Background(), root, "file.txt", "base", "external", write); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatal("already-confirmed bytes rewritten")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := SaveTextChecked(ctx, root, "file.txt", "external", "new", write); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatal("cancelled write invoked")
	}
}

func TestProductionAutosaveDelay(t *testing.T) {
	if AutoSaveDelay != 500*time.Millisecond {
		t.Fatalf("autosave debounce = %v", AutoSaveDelay)
	}
}
