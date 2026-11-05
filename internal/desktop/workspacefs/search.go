package workspacefs

import (
	"context"
	"sync"
)

const SearchPageSize = 200

type SearchSnapshot struct {
	Paths []string
	Total int
	Busy  bool
	Error string
}

// Search owns cancellable enumeration and presentation paging. Background work
// never mutates UI pointers; only the current generation can publish a result.
type Search struct {
	mu               sync.Mutex
	key, root, query string
	generation       uint64
	cancel           context.CancelFunc
	paths            []string
	visible          int
	busy             bool
	hostOnly         bool
	problem          string
}

func (s *Search) Ensure(binding, root, query string, remote RemoteDirectoryLister, notify func()) {
	s.EnsureHostOnly(binding, root, query, remote, notify, false)
}

// EnsureHostOnly keeps a remote workspace search on its captured Host even if
// a matching path happens to exist on the desktop. Ensure remains local-first.
func (s *Search) EnsureHostOnly(binding, root, query string, remote RemoteDirectoryLister, notify func(), hostOnly bool) {
	s.mu.Lock()
	if s.key == binding && s.root == root && s.query == query && s.hostOnly == hostOnly {
		s.mu.Unlock()
		return
	}
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	s.generation++
	s.key, s.root, s.query = binding, root, query
	s.hostOnly = hostOnly
	s.paths = nil
	s.visible = SearchPageSize
	s.problem = ""
	s.busy = query != "" && root != ""
	if !s.busy {
		s.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	generation := s.generation
	s.mu.Unlock()
	go func() {
		defer cancel()
		paths, err := walkRelPathsWithSource(ctx, root, normalizeFileSearch(query), remote, false, hostOnly)
		s.mu.Lock()
		if s.generation != generation || ctx.Err() != nil {
			s.mu.Unlock()
			return
		}
		s.paths = paths
		s.busy = false
		s.cancel = nil
		if err != nil {
			s.problem = err.Error()
		}
		s.mu.Unlock()
		if notify != nil {
			notify()
		}
	}()
}

// Cancel is used on binding/navigation changes, including when Files is hidden.
func (s *Search) Cancel() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	s.generation++
	s.key = ""
	s.root = ""
	s.query = ""
	s.paths = nil
	s.busy = false
	s.problem = ""
}
func (s *Search) Snapshot() SearchSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := min(len(s.paths), s.visible)
	return SearchSnapshot{Paths: append([]string(nil), s.paths[:n]...), Total: len(s.paths), Busy: s.busy, Error: s.problem}
}
func (s *Search) More() { s.mu.Lock(); defer s.mu.Unlock(); s.visible += SearchPageSize }
