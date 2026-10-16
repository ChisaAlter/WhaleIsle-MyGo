package workspacefs

import (
	"context"
	"strings"
	"sync"
)

type DirectorySnapshot struct {
	Entries    []Entry
	Busy       bool
	Error      string
	WatchError string
}
type directoryResult struct {
	entries                     []Entry
	busy, loaded, dirty, forced bool
	problem, watchProblem       string
	ctx                         context.Context
	cancel                      context.CancelFunc
	watchCancel                 context.CancelFunc
	remote                      RemoteDirectoryLister
	notify                      func()
}

// RemoteDirectoryWatcher observes one directory, not its descendants. Both
// readiness and changes invalidate its listing (ready closes the initial race).
type RemoteDirectoryWatcher func(context.Context, string, func()) error

// DirectoryTree owns cached listings and watches for the open directory tree.
// UI expansion and selection belong to the caller and survive refreshes.
type DirectoryTree struct {
	mu            sync.Mutex
	binding, root string
	ctx           context.Context
	cancel        context.CancelFunc
	remote        bool
	automatic     bool
	watch         RemoteDirectoryWatcher
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
	t.remote = false
	t.automatic = true
	t.directories = map[string]*directoryResult{}
}
func (t *DirectoryTree) Cancel() { t.mu.Lock(); defer t.mu.Unlock(); t.resetLocked() }

// Observe stops subscriptions while automatic refresh is disabled. Enabling
// refresh restarts cached directory watches and catches up changes missed while off.
func (t *DirectoryTree) Observe(watch RemoteDirectoryWatcher, automatic bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	wasAutomatic := t.automatic
	t.watch, t.automatic = watch, automatic
	for rel, result := range t.directories {
		if !automatic || watch == nil {
			if result.watchCancel != nil {
				result.watchCancel()
				result.watchCancel = nil
			}
			result.watchProblem = ""
		}
		if automatic {
			t.watchLocked(rel, result)
			if !wasAutomatic {
				result.dirty = true
			}
			if result.dirty {
				t.readLocked(rel, result)
			}
		}
	}
}

// Refresh rereads open levels without discarding their displayed entries.
func (t *DirectoryTree) Refresh() {
	t.mu.Lock()
	defer t.mu.Unlock()
	for rel, result := range t.directories {
		result.dirty, result.forced = true, true
		t.readLocked(rel, result)
	}
}

// Collapse retires the directory and all descendants, including late replies.
func (t *DirectoryTree) Collapse(rel string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.collapseLocked(rel)
}
func (t *DirectoryTree) collapseLocked(rel string) {
	for path, result := range t.directories {
		if path == rel || strings.HasPrefix(path, rel+"/") {
			result.cancel()
			delete(t.directories, path)
		}
	}
}

func (t *DirectoryTree) Get(rel string, remote RemoteDirectoryLister, notify func()) DirectorySnapshot {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.ctx == nil {
		t.resetLocked()
	}
	result := t.directories[rel]
	if result == nil {
		if rel != "" {
			parent := t.directories[""]
			if parent == nil || !parent.loaded || parent.problem != "" {
				return DirectorySnapshot{Busy: true}
			}
		}
		ctx, cancel := context.WithCancel(t.ctx)
		result = &directoryResult{ctx: ctx, cancel: cancel, remote: remote, notify: notify}
		t.directories[rel] = result
		t.readLocked(rel, result)
		t.watchLocked(rel, result)
	}
	return DirectorySnapshot{Entries: append([]Entry(nil), result.entries...), Busy: result.busy && !result.loaded, Error: result.problem, WatchError: result.watchProblem}
}

func (t *DirectoryTree) watchLocked(rel string, result *directoryResult) {
	if !t.automatic || t.watch == nil || result.watchCancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(result.ctx)
	result.watchCancel = cancel
	result.watchProblem = ""
	go t.follow(ctx, rel, result, t.watch)
}

func (t *DirectoryTree) follow(ctx context.Context, rel string, result *directoryResult, watch RemoteDirectoryWatcher) {
	err := watch(ctx, rel, func() {
		t.mu.Lock()
		if ctx.Err() != nil || t.directories[rel] != result {
			t.mu.Unlock()
			return
		}
		result.dirty = true
		if !result.loaded {
			result.forced = true
		}
		if t.automatic || !result.loaded {
			t.readLocked(rel, result)
		}
		t.mu.Unlock()
	})
	t.mu.Lock()
	if ctx.Err() != nil || t.directories[rel] != result {
		t.mu.Unlock()
		return
	}
	if err != nil && !strings.Contains(err.Error(), "workspace-file/watch-unsupported") {
		result.watchProblem = err.Error()
	}
	t.mu.Unlock()
	if result.notify != nil {
		result.notify()
	}
}

// A dirty bit coalesces events during a read into one subsequent read. The
// result identity and cancelled context prevent old work publishing on rebind.
func (t *DirectoryTree) readLocked(rel string, result *directoryResult) {
	if result.busy {
		return
	}
	result.busy, result.dirty, result.forced = true, false, false
	root, useRemote := t.root, t.remote
	go func() {
		var entries []Entry
		var err error
		if useRemote && result.remote != nil {
			entries, err = result.remote(result.ctx, rel)
		} else {
			entries, err = ListWorkspaceDir(result.ctx, root, rel)
			if rel == "" && err != nil && result.ctx.Err() == nil && result.remote != nil {
				entries, err = result.remote(result.ctx, rel)
				useRemote = true
			}
		}
		if err == nil {
			entries, err = validatedEntries(entries)
		}
		t.mu.Lock()
		if result.ctx.Err() != nil || t.directories[rel] != result {
			t.mu.Unlock()
			return
		}
		result.busy = false
		result.problem = ""
		if err != nil {
			result.problem = err.Error()
		} else {
			result.entries, result.loaded = entries, true
			// Removed directories must not keep invisible subscriptions alive.
			children := map[string]bool{}
			for _, entry := range entries {
				if entry.IsDir {
					child := entry.Name
					if rel != "" {
						child = rel + "/" + child
					}
					children[child] = true
				}
			}
			for path := range t.directories {
				if path == "" || path == rel {
					continue
				}
				parent := ""
				if i := strings.LastIndex(path, "/"); i >= 0 {
					parent = path[:i]
				}
				if parent == rel && !children[path] {
					t.collapseLocked(path)
				}
			}
		}
		if rel == "" {
			t.remote = useRemote
		}
		if result.dirty && (t.automatic || result.forced) {
			t.readLocked(rel, result)
		}
		t.mu.Unlock()
		if result.notify != nil {
			result.notify()
		}
	}()
}
