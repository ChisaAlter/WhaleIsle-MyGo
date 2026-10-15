package workspacefs

import (
	"context"
	"sync"
)

type DirectorySnapshot struct {
	Entries []Entry
	Busy    bool
	Error   string
}
type directoryResult struct {
	entries []Entry
	busy    bool
	problem string
}

// DirectoryTree caches asynchronous directory reads for the current binding.
// Refresh resets the cache, and cancellation/generations isolate late replies.
type DirectoryTree struct {
	mu            sync.Mutex
	binding, root string
	generation    uint64
	ctx           context.Context
	cancel        context.CancelFunc
	remote        bool
	directories   map[string]*directoryResult
}

func (t *DirectoryTree) Ensure(binding, root string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.ctx != nil && binding == t.binding && root == t.root {
		return
	}
	t.resetLocked()
	t.binding, t.root = binding, root
}
func (t *DirectoryTree) resetLocked() {
	if t.cancel != nil {
		t.cancel()
	}
	t.ctx, t.cancel = context.WithCancel(context.Background())
	t.generation++
	t.remote = false
	t.directories = map[string]*directoryResult{}
}
func (t *DirectoryTree) Cancel() { t.mu.Lock(); defer t.mu.Unlock(); t.resetLocked() }
func (t *DirectoryTree) Get(rel string, remote RemoteDirectoryLister, notify func()) DirectorySnapshot {
	t.mu.Lock()
	if result := t.directories[rel]; result != nil {
		out := DirectorySnapshot{Entries: append([]Entry(nil), result.entries...), Busy: result.busy, Error: result.problem}
		t.mu.Unlock()
		return out
	}
	if t.ctx == nil {
		t.resetLocked()
	}
	if rel != "" {
		root := t.directories[""]
		if root == nil || root.busy || root.problem != "" {
			t.mu.Unlock()
			return DirectorySnapshot{Busy: true}
		}
	}
	t.directories[rel] = &directoryResult{busy: true}
	ctx, generation, root, useRemote := t.ctx, t.generation, t.root, t.remote
	t.mu.Unlock()
	go func() {
		var entries []Entry
		var err error
		if useRemote {
			entries, err = remote(ctx, rel)
		} else {
			entries, err = ListWorkspaceDir(ctx, root, rel)
			if rel == "" && err != nil && ctx.Err() == nil && remote != nil {
				entries, err = remote(ctx, rel)
				useRemote = true
			}
		}
		// Remote entries must remain basenames and may never expose .git.
		if err == nil {
			entries, err = validatedEntries(entries)
		}
		t.mu.Lock()
		if generation != t.generation || ctx.Err() != nil {
			t.mu.Unlock()
			return
		}
		result := t.directories[rel]
		result.busy = false
		result.entries = entries
		if err != nil {
			result.problem = err.Error()
		}
		if rel == "" {
			t.remote = useRemote
		}
		t.mu.Unlock()
		if notify != nil {
			notify()
		}
	}()
	return DirectorySnapshot{Busy: true}
}
