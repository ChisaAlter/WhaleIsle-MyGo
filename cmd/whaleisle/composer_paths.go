package main

import (
	"time"

	"github.com/egoist/mygo"

	"whaleisle/internal/desktop/workspacefs"
)

// composerPathsRetryAfter is how long a failed index of a workspace root waits
// before the next attempt, so a root that cannot be read is not walked again on
// every frame.
const composerPathsRetryAfter = 5 * time.Second

// composerPathsLocked returns the indexed workspace files for @ suggestions. It
// never walks the tree itself: the walk reads every directory and asks git
// about each level, which took from seconds to half a minute on a large
// workspace while it ran inside the view build and froze the window. A missing
// index starts one worker and the suggestions appear when it finishes; callers
// hold s.mu.
func (s *mainState) composerPathsLocked(root string) []string {
	if root == "" {
		return nil
	}
	if s.composerPathsRoot == root {
		return s.composerPaths
	}
	if s.composerPathsLoading == root {
		return nil
	}
	if s.composerPathsFailedRoot == root && time.Now().Before(s.composerPathsRetry) {
		return nil
	}
	s.composerPathsLoading = root
	// Paths of another workspace must not be offered while this one is read.
	s.composerPaths, s.composerPathsRoot = nil, ""
	go s.indexComposerPaths(root, s.win)
	return nil
}

func (s *mainState) indexComposerPaths(root string, win *mygo.Window) {
	paths, err := workspacefs.IndexRelPaths(root, 4000)
	s.mu.Lock()
	if s.composerPathsLoading == root {
		s.composerPathsLoading = ""
		if err == nil {
			if paths == nil {
				paths = []string{}
			}
			s.composerPathsRoot, s.composerPaths = root, paths
			s.composerPathsFailedRoot = ""
		} else {
			s.composerPathsFailedRoot, s.composerPathsRetry = root, time.Now().Add(composerPathsRetryAfter)
		}
	}
	s.mu.Unlock()
	if win != nil {
		win.Update(func() {})
	}
}
