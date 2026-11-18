package diffsurface

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/git"
	"whaleisle/internal/ui/i18n"
)

func TestStageWriteSurvivesNavigationAndKeepsItsResult(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	entered, finish := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	withTestGitOps(t, func(ops *diffsurfaceGitOps) {
		ops.stagePaths = func(root string, paths ...string) (string, error) {
			calls.Add(1)
			close(entered)
			<-finish
			return "", errors.New("first stage result")
		}
	})
	root := `C:\mock\first`
	w := &Workbench{Root: root}
	m := Model{WorkspaceRoot: root, Workbench: w}
	w.requestStage(m, git.Change{Path: "first.txt", Worktree: 'M'}, true)
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("write did not start")
	}
	operation := w.Operation
	w.requestStage(m, git.Change{Path: "second.txt", Worktree: 'M'}, true)
	if w.Operation != operation || calls.Load() != 1 {
		t.Fatal("second action replaced an admitted write")
	}
	w.ClosePane()
	w.resetStateForRoot(`C:\mock\second`, nil)
	if w.Operation != operation {
		t.Fatal("navigation lost write ownership")
	}
	close(finish)
	waitUntil(t, "first write result after navigation", func() bool {
		finishGitOperation(Model{WorkspaceRoot: w.Root, Workbench: w})
		return w.Operation == nil
	}, nil)
	if !strings.Contains(w.Status, "first stage result") || !strings.Contains(w.Status, root) {
		t.Fatalf("lost write result or target: %q", w.Status)
	}
}

func withTestGitOps(t *testing.T, configure func(ops *diffsurfaceGitOps)) {
	t.Helper()
	prev := gitOps
	next := defaultDiffsurfaceGitOps()
	configure(&next)
	gitOps = next
	defaultDiffCache.mu.Lock()
	clear(defaultDiffCache.cache)
	for k, req := range defaultDiffCache.inFlight {
		req.cancel()
		delete(defaultDiffCache.inFlight, k)
	}
	defaultDiffCache.gen++
	defaultDiffCache.mu.Unlock()

	t.Cleanup(func() {
		gitOps = prev
		defaultDiffCache.mu.Lock()
		clear(defaultDiffCache.cache)
		for k, req := range defaultDiffCache.inFlight {
			req.cancel()
			delete(defaultDiffCache.inFlight, k)
		}
		defaultDiffCache.gen++
		defaultDiffCache.mu.Unlock()
	})
}

func waitUntil(t *testing.T, desc string, cond func() bool, step func()) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for: %s", desc)
		}
		time.Sleep(10 * time.Millisecond)
		if step != nil {
			step()
		}
	}
}

func TestAsyncLoaderSingleFlightCoalescing(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)

	release := make(chan struct{})
	var isRepoCalls, changesCalls, largeCalls, statusCalls, diffCalls, branchCalls atomic.Int32
	withTestGitOps(t, func(ops *diffsurfaceGitOps) {
		ops.isRepository = func(root string) bool {
			isRepoCalls.Add(1)
			<-release
			return true
		}
		ops.changes = func(root string) ([]git.Change, error) {
			changesCalls.Add(1)
			<-release
			return []git.Change{{Path: "main.go", Worktree: 'M'}}, nil
		}
		ops.checkLargeFiles = func(root string) ([]git.LargeFile, error) {
			largeCalls.Add(1)
			<-release
			return nil, nil
		}
		ops.readRepoStatus = func(ctx context.Context, root string) (git.RepositoryStatus, error) {
			statusCalls.Add(1)
			<-release
			return git.RepositoryStatus{IsRepo: true, Branch: "main"}, nil
		}
		ops.diffHEAD = func(root, relPath string) (string, error) {
			diffCalls.Add(1)
			<-release
			return "diff --git a/main.go b/main.go\n--- a/main.go\n+++ b/main.go\n@@ -1 +1 @@\n-old\n+coalesced\n", nil
		}
		ops.branchesContext = func(ctx context.Context, root string) ([]git.Branch, error) {
			branchCalls.Add(1)
			<-release
			return []git.Branch{{Name: "main", Current: true}, {Name: "dev"}}, nil
		}
	})

	root := `C:\mock\repo-single-flight`
	w := &Workbench{CompareOpen: true}
	defer w.ClosePane()

	var invCount atomic.Int32
	inv := func() { invCount.Add(1) }
	var text, errMsg string
	m := Model{WorkspaceRoot: root, Workbench: w, Text: &text, ErrMsg: &errMsg}

	tt := ui.NewTester(func(c *ui.Context) {
		EnsureDiffAsync(w, root, "", &text, &errMsg, c.Services().Invalidate)
		View(c, m)
		w.requestCompareBranches(false)
	}, 900, 500)

	// Repeated frames while background git calls are blocked must coalesce to 1 call each.
	for range 5 {
		tt.Frame()
		EnsureDiffAsync(w, root, "", &text, &errMsg, inv)
		w.requestReload(root, false)
		w.requestCompareBranches(false)
	}

	if w.Loaded {
		t.Fatal("workbench marked loaded before background reload completed")
	}
	if !tt.HasText("Loading diff…") {
		t.Fatalf("expected non-blocking loading diff placeholder, got texts=%v", tt.Texts())
	}
	waitUntil(t, "single-flight workers enter mocks", func() bool {
		return isRepoCalls.Load() == 1 && diffCalls.Load() == 1 && statusCalls.Load() == 1 && branchCalls.Load() == 1
	}, nil)
	if got := diffCalls.Load(); got != 1 {
		t.Fatalf("diffHEAD spawned %d times while in flight, want 1", got)
	}
	if got := isRepoCalls.Load(); got != 1 {
		t.Fatalf("isRepository spawned %d times while in flight, want 1", got)
	}
	if got := statusCalls.Load(); got != 1 {
		t.Fatalf("readRepoStatus spawned %d times while in flight, want 1", got)
	}
	if got := branchCalls.Load(); got != 1 {
		t.Fatalf("branchesContext spawned %d times while in flight, want 1", got)
	}

	close(release)
	waitUntil(t, "single-flight workbench load", func() bool {
		return w.Loaded && strings.Contains(text, "+coalesced") && len(w.CompareBranches) == 2 && w.RepositoryStatus != nil
	}, tt.Frame)

	if isRepoCalls.Load() != 1 || changesCalls.Load() != 1 || diffCalls.Load() != 1 || statusCalls.Load() != 1 || branchCalls.Load() != 1 {
		t.Fatalf("expected strict single-flight counts, got repo=%d changes=%d diff=%d status=%d branch=%d",
			isRepoCalls.Load(), changesCalls.Load(), diffCalls.Load(), statusCalls.Load(), branchCalls.Load())
	}
	if !w.Repository || len(w.Changes) != 1 || w.Changes[0].Path != "main.go" {
		t.Fatalf("unexpected workbench state: repo=%v changes=%+v", w.Repository, w.Changes)
	}
}

func TestAsyncLoaderGenerationAndRootSwitchDiscardsStaleResults(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)

	rootA := `C:\mock\repo-A`
	rootB := `C:\mock\repo-B`
	releaseA := make(chan struct{})
	var rootAWg sync.WaitGroup
	var staleReturned atomic.Int32

	withTestGitOps(t, func(ops *diffsurfaceGitOps) {
		ops.isRepository = func(root string) bool {
			if root == rootA {
				rootAWg.Add(1)
				defer rootAWg.Done()
				<-releaseA
			}
			return true
		}
		ops.changes = func(root string) ([]git.Change, error) {
			if root == rootA {
				rootAWg.Add(1)
				defer rootAWg.Done()
				<-releaseA
				staleReturned.Add(1)
				return []git.Change{{Path: "stale.txt", Worktree: 'M'}}, nil
			}
			return []git.Change{{Path: "fresh-b.txt", Worktree: 'M'}}, nil
		}
		ops.checkLargeFiles = func(root string) ([]git.LargeFile, error) {
			return nil, nil
		}
		ops.readRepoStatus = func(ctx context.Context, root string) (git.RepositoryStatus, error) {
			if root == rootA {
				rootAWg.Add(1)
				defer rootAWg.Done()
				<-releaseA
				return git.RepositoryStatus{IsRepo: true, Branch: "stale-a"}, nil
			}
			return git.RepositoryStatus{IsRepo: true, Branch: "fresh-b"}, nil
		}
		ops.diffHEAD = func(root, relPath string) (string, error) {
			if root == rootA {
				rootAWg.Add(1)
				defer rootAWg.Done()
				<-releaseA
				staleReturned.Add(1)
				return "diff --git a/stale.txt b/stale.txt\n--- a/stale.txt\n+++ b/stale.txt\n@@ -1 +1 @@\n-old\n+stale-a\n", nil
			}
			return "diff --git a/fresh-b.txt b/fresh-b.txt\n--- a/fresh-b.txt\n+++ b/fresh-b.txt\n@@ -1 +1 @@\n-old\n+fresh-b\n", nil
		}
		ops.branchesContext = func(ctx context.Context, root string) ([]git.Branch, error) {
			if root == rootA {
				rootAWg.Add(1)
				defer rootAWg.Done()
				<-releaseA
				return []git.Branch{{Name: "branch-a", Current: true}}, nil
			}
			return []git.Branch{{Name: "branch-b", Current: true}}, nil
		}
		ops.prepareDiscard = func(root, path string) (git.DiscardPlan, error) {
			if root == rootA {
				rootAWg.Add(1)
				defer rootAWg.Done()
				<-releaseA
				return git.DiscardPlan{Path: "stale.txt"}, nil
			}
			return git.DiscardPlan{Path: path}, nil
		}
	})

	w := &Workbench{}
	defer w.ClosePane()
	var text, errMsg, pathFilter string
	m := Model{WorkspaceRoot: rootA, Workbench: w, Text: &text, ErrMsg: &errMsg, PathFilter: &pathFilter}

	tt := ui.NewTester(func(c *ui.Context) {
		View(c, m)
	}, 900, 500)
	tt.Frame()

	// Also start compare-branch and discard-prepare on rootA while rootA is slow.
	w.requestCompareBranches(false)
	reviewDiscard(m, "stale.txt")

	// Switch to rootB before rootA finishes.
	m.WorkspaceRoot = rootB
	tt.Frame()
	waitUntil(t, "rootB load settles", func() bool {
		return w.Loaded && w.Root == rootB && strings.Contains(text, "+fresh-b") && w.RepositoryStatus != nil
	}, tt.Frame)

	// Now unblock slow rootA workers; stale results must be dropped.
	close(releaseA)
	rootAWg.Wait()
	for range 3 {
		tt.Frame()
		w.finishAsync(m)
	}

	if w.Root != rootB {
		t.Fatalf("root reverted to %q, want %q", w.Root, rootB)
	}
	if !strings.Contains(text, "+fresh-b") || strings.Contains(text, "+stale-a") {
		t.Fatalf("stale diff leaked after root switch: %q", text)
	}
	if len(w.Changes) != 1 || w.Changes[0].Path != "fresh-b.txt" {
		t.Fatalf("stale changes overwrote rootB: %+v", w.Changes)
	}
	if w.RepositoryStatus == nil || w.RepositoryStatus.Branch != "fresh-b" {
		t.Fatalf("stale repository status overwrote rootB: %+v", w.RepositoryStatus)
	}
	if w.Discard != nil {
		t.Fatalf("stale discard plan from rootA leaked into rootB: %+v", w.Discard)
	}
	for _, b := range w.CompareBranches {
		if b.Name == "branch-a" {
			t.Fatalf("stale branch-a leaked into rootB: %+v", w.CompareBranches)
		}
	}
	if staleReturned.Load() == 0 {
		t.Fatal("expected blocked rootA mocks to have executed before verification")
	}
}

func TestAsyncLoaderSameRootForceBumpDiscardsOldGeneration(t *testing.T) {
	root := `C:\mock\repo-same-root-gen`
	releaseGen1 := make(chan struct{})
	var gen1Wg sync.WaitGroup
	var diffAttempt atomic.Int32

	withTestGitOps(t, func(ops *diffsurfaceGitOps) {
		ops.isRepository = func(root string) bool { return true }
		ops.changes = func(root string) ([]git.Change, error) {
			return []git.Change{{Path: "app.go", Worktree: 'M'}}, nil
		}
		ops.checkLargeFiles = func(root string) ([]git.LargeFile, error) { return nil, nil }
		ops.readRepoStatus = func(ctx context.Context, root string) (git.RepositoryStatus, error) {
			return git.RepositoryStatus{IsRepo: true, Branch: "main"}, nil
		}
		ops.diffHEAD = func(root, relPath string) (string, error) {
			if diffAttempt.Add(1) == 1 {
				gen1Wg.Add(1)
				defer gen1Wg.Done()
				<-releaseGen1
				return "diff --git a/app.go b/app.go\n--- a/app.go\n+++ b/app.go\n@@ -1 +1 @@\n-old\n+gen1-stale\n", nil
			}
			return "diff --git a/app.go b/app.go\n--- a/app.go\n+++ b/app.go\n@@ -1 +1 @@\n-old\n+gen2-fresh\n", nil
		}
	})

	w := &Workbench{}
	defer w.ClosePane()
	var text, errMsg string
	m := Model{WorkspaceRoot: root, Workbench: w, Text: &text, ErrMsg: &errMsg}

	tt := ui.NewTester(func(c *ui.Context) {
		View(c, m)
	}, 900, 500)
	tt.Frame()

	// Bump generation via resetStateForRoot + forced reload (simulating newer workspace revision).
	w.resetStateForRoot(root, nil)
	refreshWorkbench(m)
	waitUntil(t, "gen2 diff settles", func() bool {
		return strings.Contains(text, "+gen2-fresh")
	}, tt.Frame)

	// Unblock slow gen-1 diff; it must not overwrite gen-2 diff or cache.
	close(releaseGen1)
	gen1Wg.Wait()
	for range 3 {
		tt.Frame()
		w.finishAsync(m)
	}
	if !strings.Contains(text, "+gen2-fresh") || strings.Contains(text, "+gen1-stale") {
		t.Fatalf("stale same-root generation overwrote fresh diff: %q", text)
	}
}

func TestExportedAsyncDiffAPIsAndPaneCopySafety(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)

	var calls atomic.Int32
	release := make(chan struct{})
	withTestGitOps(t, func(ops *diffsurfaceGitOps) {
		ops.diffHEAD = func(root, relPath string) (string, error) {
			n := calls.Add(1)
			if n == 1 {
				<-release
				return "first-diff", nil
			}
			return "second-diff", nil
		}
	})

	root := `C:\mock\repo-exported`
	w := &Workbench{Root: root, Repository: true, Loaded: true, AgainstHEAD: true}
	var text, errMsg string
	var invCount atomic.Int32
	inv := func() { invCount.Add(1) }

	// 1. EnsureDiffAsync is non-blocking and deduplicated while in-flight.
	if loading := EnsureDiffAsync(w, root, "a.go", &text, &errMsg, inv); !loading {
		t.Fatal("expected initial EnsureDiffAsync to report loading=true")
	}
	if loading := EnsureDiffAsync(w, root, "a.go", &text, &errMsg, inv); !loading {
		t.Fatal("expected repeated EnsureDiffAsync to remain loading=true")
	}
	waitUntil(t, "EnsureDiffAsync starts single diffHEAD worker", func() bool {
		return calls.Load() >= 1
	}, nil)
	if calls.Load() != 1 {
		t.Fatalf("expected coalesced single diffHEAD call, got %d", calls.Load())
	}

	close(release)
	waitUntil(t, "EnsureDiffAsync populates text", func() bool {
		loading := EnsureDiffAsync(w, root, "a.go", &text, &errMsg, inv)
		return !loading && text == "first-diff"
	}, nil)
	if invCount.Load() == 0 {
		t.Fatal("expected invalidate callback on EnsureDiffAsync completion")
	}
	// Cached read does not trigger another git diff.
	if loading := EnsureDiffAsync(w, root, "a.go", &text, &errMsg, inv); loading || calls.Load() != 1 {
		t.Fatalf("expected cache hit with 0 extra calls: loading=%v calls=%d", loading, calls.Load())
	}

	// 2. StartDiffAsync resets filters, clears strings, and forces reload.
	w.BaseRef = "origin/main"
	w.Staged = true
	w.AgainstHEAD = false
	StartDiffAsync(w, root, "a.go", &text, &errMsg, inv)
	if text != "" || errMsg != "" || w.BaseRef != "" || w.Staged || !w.AgainstHEAD {
		t.Fatalf("StartDiffAsync did not reset state cleanly: text=%q base=%q staged=%v againstHEAD=%v", text, w.BaseRef, w.Staged, w.AgainstHEAD)
	}
	waitUntil(t, "StartDiffAsync second diff", func() bool {
		out, e, ready, _ := w.PollDiffAsync(root, "a.go")
		if ready {
			text, errMsg = out, e
		}
		return ready && text == "second-diff"
	}, nil)

	// 3. nil Workbench EnsureDiffAsync uses fallbackDiffLoader without panic.
	var fbText, fbErr string
	if !EnsureDiffAsync(nil, root, "fallback.go", &fbText, &fbErr, inv) {
		t.Fatal("expected nil-workbench EnsureDiffAsync to load asynchronously")
	}
	waitUntil(t, "fallback EnsureDiffAsync settles", func() bool {
		return !EnsureDiffAsync(nil, root, "fallback.go", &fbText, &fbErr, inv) && fbText == "second-diff"
	}, nil)

	// 4. CopyPanePresentation yields clean value copy; ClosePane prevents stale invalidate.
	copied := w.CopyPanePresentation()
	if copied.loader != nil || copied.View.list != nil {
		t.Fatalf("CopyPanePresentation leaked live loader or list state: loader=%v list=%v", copied.loader, copied.View.list)
	}
	w.ClosePane()
	prevInv := invCount.Load()
	w.RequestDiffAsync(root, "after-close.go", true, inv)
	time.Sleep(30 * time.Millisecond)
	if invCount.Load() != prevInv {
		t.Fatalf("closed pane emitted unexpected invalidation: before=%d after=%d", prevInv, invCount.Load())
	}
}

func TestAsyncClickPathsAreNonBlocking(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)

	release := make(chan struct{})
	var stageCalls, discardPrepCalls, publishPrepCalls atomic.Int32
	withTestGitOps(t, func(ops *diffsurfaceGitOps) {
		ops.isRepository = func(root string) bool { <-release; return true }
		ops.changes = func(root string) ([]git.Change, error) {
			<-release
			return []git.Change{{Path: "file.txt", Worktree: 'M'}}, nil
		}
		ops.checkLargeFiles = func(root string) ([]git.LargeFile, error) { <-release; return nil, nil }
		ops.readRepoStatus = func(ctx context.Context, root string) (git.RepositoryStatus, error) {
			<-release
			return git.RepositoryStatus{IsRepo: true, Branch: "main"}, nil
		}
		ops.diffHEAD = func(root, relPath string) (string, error) { <-release; return "", nil }
		ops.stagePaths = func(root string, paths ...string) (string, error) {
			stageCalls.Add(1)
			<-release
			return "", nil
		}
		ops.prepareDiscard = func(root, path string) (git.DiscardPlan, error) {
			discardPrepCalls.Add(1)
			<-release
			return git.DiscardPlan{Path: path}, nil
		}
		ops.preparePublish = func(root string, input git.PublishInput) (git.PublishInput, error) {
			publishPrepCalls.Add(1)
			<-release
			input.Name = "resolved-repo"
			return input, nil
		}
	})

	root := `C:\mock\repo-clicks`
	w := &Workbench{
		Root:        root,
		Loaded:      true,
		Repository:  true,
		PublishOpen: true,
		PublishName: "draft",
		Changes:     []git.Change{{Path: "file.txt", Worktree: 'M'}},
	}
	defer w.ClosePane()
	var text, errMsg string
	m := Model{WorkspaceRoot: root, Workbench: w, Text: &text, ErrMsg: &errMsg}

	// Render frame with PublishOpen so "Review publish target" click runs async.
	tt := ui.NewTester(func(c *ui.Context) {
		paintPublish(c, m, c.Theme().Text, c.Theme().TextMuted)
	}, 700, 400)
	if err := tt.Click("Review publish target"); err != nil {
		t.Fatal(err)
	}
	waitUntil(t, "publish prepare enters mock", func() bool {
		return publishPrepCalls.Load() >= 1
	}, nil)
	if w.PublishReview || !w.isPublishPreparing() || publishPrepCalls.Load() != 1 {
		t.Fatalf("publish prepare blocked or failed to launch async: review=%v busy=%v calls=%d", w.PublishReview, w.isPublishPreparing(), publishPrepCalls.Load())
	}

	// Stage and Discard prepare must return immediately without blocking on release.
	stageCommitFile(m, git.Change{Path: "file.txt", Worktree: 'M'})
	reviewDiscard(m, "file.txt")
	waitUntil(t, "stage and discard enter mocks", func() bool {
		return stageCalls.Load() >= 1 && discardPrepCalls.Load() >= 1
	}, nil)
	if !w.isStagingBusy() || stageCalls.Load() != 1 {
		t.Fatalf("stage did not run asynchronously: busy=%v calls=%d", w.isStagingBusy(), stageCalls.Load())
	}
	if !w.isDiscardPreparing() || w.Discard != nil || discardPrepCalls.Load() != 1 {
		t.Fatalf("discard prepare did not run asynchronously: busy=%v plan=%v calls=%d", w.isDiscardPreparing(), w.Discard, discardPrepCalls.Load())
	}

	close(release)
	waitUntil(t, "async click results applied", func() bool {
		w.finishAsync(m)
		tt.Frame()
		return w.PublishReview && w.ReviewedPublish.Name == "resolved-repo" && w.Discard != nil && w.Discard.Path == "file.txt" && !w.isStagingBusy()
	}, nil)
}
