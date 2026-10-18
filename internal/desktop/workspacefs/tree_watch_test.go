package workspacefs

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

type treeWatchFixture struct {
	ctx     context.Context
	changed func()
}

func awaitTreeWatch[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case value := <-ch:
		return value
	case <-time.After(2 * time.Second):
		t.Fatal("directory watch did not settle")
	}
	var zero T
	return zero
}

func TestDirectoryTreeWatchRefreshToggleAndCancel(t *testing.T) {
	var tree DirectoryTree
	defer tree.Cancel()
	tree.Ensure("host-a/session-a", filepath.Join(t.TempDir(), "remote-workspace"))
	opened := make(chan treeWatchFixture, 8)
	watch := func(ctx context.Context, _ string, changed func()) error {
		opened <- treeWatchFixture{ctx, changed}
		<-ctx.Done()
		return ctx.Err()
	}
	var mu sync.Mutex
	name := "before.txt"
	reads := 0
	remote := func(context.Context, string) ([]Entry, error) {
		mu.Lock()
		defer mu.Unlock()
		reads++
		return []Entry{{Name: name}}, nil
	}
	updated := make(chan struct{}, 20)
	notify := func() { updated <- struct{}{} }
	tree.Observe(watch, true)
	tree.Get("", remote, notify)
	subscription := awaitTreeWatch(t, opened)
	awaitTreeWatch(t, updated)
	if got := tree.Get("", remote, notify); got.Busy || got.Entries[0].Name != "before.txt" {
		t.Fatalf("initial: %+v", got)
	}

	tree.Observe(watch, false)
	mu.Lock()
	name = "after.txt"
	count := reads
	mu.Unlock()
	subscription.changed()
	mu.Lock()
	unchanged := reads == count
	mu.Unlock()
	if !unchanged {
		t.Fatal("disabled auto refresh performed a read")
	}
	if subscription.ctx.Err() == nil {
		t.Fatal("disabled auto refresh retained subscription")
	}
	if got := tree.Get("", remote, notify); got.Entries[0].Name != "before.txt" {
		t.Fatalf("disabled snapshot: %+v", got)
	}
	tree.Observe(watch, true)
	restarted := awaitTreeWatch(t, opened)
	awaitTreeWatch(t, updated)
	subscription.changed()
	tree.mu.Lock()
	staleRead := tree.directories[""].busy || tree.directories[""].dirty
	tree.mu.Unlock()
	if staleRead {
		t.Fatal("retired subscription triggered a read after restart")
	}
	if got := tree.Get("", remote, notify); got.Entries[0].Name != "after.txt" {
		t.Fatalf("enable did not catch up: %+v", got)
	}

	tree.Observe(watch, false)
	if restarted.ctx.Err() == nil {
		t.Fatal("second disable retained restarted subscription")
	}
	mu.Lock()
	name = "manual.txt"
	mu.Unlock()
	tree.Refresh()
	awaitTreeWatch(t, updated)
	if got := tree.Get("", remote, notify); got.Entries[0].Name != "manual.txt" {
		t.Fatalf("manual refresh while disabled: %+v", got)
	}
	tree.Ensure("host-b/session-b", "/another-workspace")
	if subscription.ctx.Err() == nil {
		t.Fatal("workspace switch did not cancel watch")
	}
	subscription.changed() // late Host notification must not affect the new cache
	tree.mu.Lock()
	stale := len(tree.directories)
	tree.mu.Unlock()
	if stale != 0 {
		t.Fatal("old notification populated new workspace")
	}
}

func TestDirectoryTreeWatchCollapseAndCoalesce(t *testing.T) {
	var tree DirectoryTree
	defer tree.Cancel()
	tree.Ensure("host/session", filepath.Join(t.TempDir(), "remote-workspace"))
	opened := make(chan treeWatchFixture, 8)
	watch := func(ctx context.Context, _ string, changed func()) error {
		opened <- treeWatchFixture{ctx, changed}
		<-ctx.Done()
		return ctx.Err()
	}
	rootDone := make(chan struct{}, 8)
	childRead := make(chan chan []Entry, 8)
	remote := func(ctx context.Context, path string) ([]Entry, error) {
		if path == "" {
			return []Entry{{Name: "dir", IsDir: true}}, nil
		}
		response := make(chan []Entry, 1)
		childRead <- response
		// Deliberately emulate a transport returning a late response after cancellation.
		return <-response, nil
	}
	tree.Observe(watch, true)
	tree.Get("", remote, func() { rootDone <- struct{}{} })
	awaitTreeWatch(t, opened)
	awaitTreeWatch(t, rootDone)
	childDone := make(chan struct{}, 8)
	tree.Get("dir", remote, func() { childDone <- struct{}{} })
	childWatch := awaitTreeWatch(t, opened)
	first := awaitTreeWatch(t, childRead)
	childWatch.changed()
	childWatch.changed()
	first <- []Entry{{Name: "old.txt"}}
	second := awaitTreeWatch(t, childRead)
	awaitTreeWatch(t, childDone)
	// A refresh keeps the last entries visible rather than collapsing the subtree.
	if got := tree.Get("dir", remote, nil); got.Busy || got.Entries[0].Name != "old.txt" {
		t.Fatalf("refresh snapshot: %+v", got)
	}
	tree.Collapse("dir")
	if childWatch.ctx.Err() == nil {
		t.Fatal("collapse did not cancel directory watch")
	}
	second <- []Entry{{Name: "late.txt"}}
	childWatch.changed()
	tree.mu.Lock()
	_, retained := tree.directories["dir"]
	tree.mu.Unlock()
	if retained {
		t.Fatal("collapsed cache retained")
	}
}
