package main

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	desktopgit "whaleisle/internal/desktop/git"
)

type gitRefreshRead struct {
	root   string
	cancel context.CancelFunc
}

func (s *mainState) refreshGit(root string) bool {
	return s.refreshGitWith(root, desktopgit.ReadRepositoryStatus)
}

// A frame or titlebar menu only admits a local status read. Git processes run
// off the UI thread, with one current root owner and the existing ten-second
// deadline. A replacement cancels the subprocess and rejects even a late result.
func (s *mainState) refreshGitWith(root string, read func(context.Context, string) (desktopgit.RepositoryStatus, error)) bool {
	root = strings.TrimSpace(root)
	s.mu.Lock()
	if s.gitBusy || s.gitWorkbench.Operation != nil {
		if s.gitRefresh != nil {
			s.gitRefresh.cancel()
			s.gitRefresh = nil
		}
		s.gitCheckedAt = time.Time{}
		s.mu.Unlock()
		return false
	}
	if s.quitting || s.gitRefresh != nil && s.gitRefresh.root == root {
		s.mu.Unlock()
		return false
	}
	if root == s.gitCheckedRoot && time.Since(s.gitCheckedAt) < gitRefreshInterval {
		s.mu.Unlock()
		return false
	}
	if s.gitRefresh != nil {
		s.gitRefresh.cancel()
		s.gitRefresh = nil
	}
	changed := false
	if root != s.gitCheckedRoot {
		s.gitBranch, s.gitDirty, s.gitRepositoryStatus = "", false, nil
		changed = true
	}
	s.gitCheckedRoot, s.gitCheckedAt = root, time.Now()
	if root == "" {
		s.mu.Unlock()
		return changed
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	owner := &gitRefreshRead{root: root, cancel: cancel}
	s.gitRefresh = owner
	s.mu.Unlock()
	go func() {
		defer cancel()
		status, err := read(ctx, root)
		if ctx.Err() != nil {
			err = ctx.Err()
		}
		var next *desktopgit.RepositoryStatus
		if err == nil {
			next = &status
		} else if errors.Is(err, desktopgit.ErrNotRepo) {
			next = &desktopgit.RepositoryStatus{}
		}
		s.applyGitRefresh(owner, next)
	}()
	return changed
}

func (s *mainState) applyGitRefresh(owner *gitRefreshRead, next *desktopgit.RepositoryStatus) {
	s.mu.Lock()
	if s.gitRefresh != owner || s.quitting {
		s.mu.Unlock()
		return
	}
	s.gitRefresh = nil
	if s.gitBusy || s.gitWorkbench.Operation != nil {
		s.gitCheckedAt = time.Time{}
		s.mu.Unlock()
		return
	}
	branch := ""
	if next != nil {
		branch = next.Branch
		if next.IsRepo && branch == "" {
			branch = "HEAD"
		}
	}
	s.gitCheckedAt = time.Now()
	s.gitBranch, s.gitDirty, s.gitRepositoryStatus = branch, next != nil && next.Dirty, next
	win := s.win
	s.mu.Unlock()
	if win != nil {
		ui.FrameTraceCause("git-status")
		win.Invalidate()
	}
}

func (s *mainState) cancelGitRefresh() {
	s.mu.Lock()
	if s.gitRefresh != nil {
		s.gitRefresh.cancel()
		s.gitRefresh = nil
	}
	s.mu.Unlock()
}
