package main

import (
	"context"
	"sync"
	"testing"
	"time"

	desktopgit "whaleisle/internal/desktop/git"
)

func TestGitRefreshReturnsBeforeLocalReaderCompletes(t *testing.T) {
	s := &mainState{}
	started, release, returned := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var unblock sync.Once
	defer s.cancelGitRefresh()
	defer unblock.Do(func() { close(release) })
	reader := func(ctx context.Context, root string) (desktopgit.RepositoryStatus, error) {
		close(started)
		select {
		case <-release:
			return desktopgit.RepositoryStatus{IsRepo: true, Branch: "main", Dirty: true}, nil
		case <-ctx.Done():
			return desktopgit.RepositoryStatus{}, ctx.Err()
		}
	}
	go func() { s.refreshGitWith("owned-root", reader); close(returned) }()
	select {
	case <-returned:
	case <-time.After(time.Second):
		t.Fatal("frame blocked on the Git reader")
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("background reader did not start")
	}
	s.mu.Lock()
	owner := s.gitRefresh
	if s.gitBranch != "" || s.gitRepositoryStatus != nil {
		t.Error("read published before its result")
	}
	s.mu.Unlock()
	if owner == nil {
		t.Fatal("missing in-flight local read")
	}
	unblock.Do(func() { close(release) })
	deadline := time.Now().Add(3 * time.Second)
	for {
		s.mu.Lock()
		done := s.gitBranch == "main"
		s.mu.Unlock()
		if done {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("reader result was not published")
		}
		time.Sleep(time.Millisecond)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.gitBranch != "main" || !s.gitDirty || s.gitRefresh != nil {
		t.Fatal("accepted local result did not publish")
	}
}

func TestGitRefreshRejectsLateRootAndClosedWindowResults(t *testing.T) {
	old := &gitRefreshRead{root: "old", cancel: func() {}}
	current := &gitRefreshRead{root: "current", cancel: func() {}}
	status := &desktopgit.RepositoryStatus{IsRepo: true, Branch: "current"}
	s := &mainState{gitRefresh: current, gitCheckedRoot: "current", gitRepositoryStatus: status, gitBranch: "current"}
	s.applyGitRefresh(old, &desktopgit.RepositoryStatus{IsRepo: true, Branch: "late old", Dirty: true})
	if s.gitRefresh != current || s.gitBranch != "current" || s.gitRepositoryStatus != status || s.gitDirty {
		t.Fatal("a retired root replaced the current caption status")
	}
	s.cancelGitRefresh()
	s.applyGitRefresh(current, &desktopgit.RepositoryStatus{IsRepo: true, Branch: "after close"})
	if s.gitBranch != "current" || s.gitRepositoryStatus != status {
		t.Fatal("a closed window accepted the retired read")
	}
}

func TestGitRefreshCannotPublishAcrossMutationAndPreservesErrorDistinction(t *testing.T) {
	owner := &gitRefreshRead{root: "root", cancel: func() {}}
	s := &mainState{gitRefresh: owner, gitCheckedRoot: "root", gitBranch: "old", gitBusy: true}
	s.applyGitRefresh(owner, &desktopgit.RepositoryStatus{IsRepo: true, Branch: "racing read"})
	if s.gitBranch != "old" || s.gitRefresh != nil || !s.gitCheckedAt.IsZero() {
		t.Fatal("a racing mutation accepted its old read or prevented refresh")
	}
	s.gitBusy = false
	s.gitRefresh = owner
	s.applyGitRefresh(owner, nil) // Failed status is not a verified non-repository.
	if s.gitBranch != "" || s.gitRepositoryStatus != nil {
		t.Fatal("read error fabricated a repository state")
	}
	s.gitRefresh = owner
	s.applyGitRefresh(owner, &desktopgit.RepositoryStatus{})
	if s.gitRepositoryStatus == nil || s.gitRepositoryStatus.IsRepo {
		t.Fatal("verified non-repository lost Initialize admission")
	}
}
