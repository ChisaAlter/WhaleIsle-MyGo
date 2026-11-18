package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/paths"
	"whaleisle/internal/ui/composer"
	"whaleisle/internal/ui/frame"
)

func TestMainViewDiffWritebackPreservesConcurrentUpdatesAndBottom(t *testing.T) {
	s := &mainState{
		layout: frame.LayoutSizes{
			Sidebar: 240,
			Bottom:  180,
		},
		diffText:         "concurrent diff from worker",
		gitCommitOpen:    false,
		chatPaneTab:      "chat",
		thinkingExpanded: map[string]bool{"m1": true},
	}

	startExpanded, expandedNil := cloneBoolMapForDiff(map[string]bool{"m1": true})
	start := mainViewWritebackSnapshot{
		layout:              frame.LayoutSizes{Sidebar: 240, Bottom: 0},
		draft:               "initial",
		diffText:            "stale start diff",
		gitCommitOpen:       true,
		chatPaneTab:         "chat",
		thinkingExpanded:    startExpanded,
		thinkingExpandedNil: expandedNil,
	}

	cur := start
	cur.thinkingExpanded = map[string]bool{"m1": true}
	cur.draft = "edited draft"
	cur.layout.Sidebar = 300
	cur.layout.Bottom = 0 // frame-local copy did not observe concurrent toggleTerminal

	s.mu.Lock()
	prevTab := s.commitMainViewWritebackLocked(start, cur)
	s.mu.Unlock()

	if prevTab != "chat" {
		t.Fatalf("unexpected previous tab: %q", prevTab)
	}
	if s.diffText != "concurrent diff from worker" {
		t.Fatalf("untouched diffText clobbered concurrent update: %q", s.diffText)
	}
	if s.gitCommitOpen {
		t.Fatal("untouched gitCommitOpen clobbered callback close")
	}
	if s.draft != "edited draft" {
		t.Fatalf("modified draft not written back: %q", s.draft)
	}
	if s.layout.Sidebar != 300 || s.layout.Bottom != 180 {
		t.Fatalf("layout writeback wrong or clobbered Bottom: %+v", s.layout)
	}
}

func TestComposerDiffWritebackPreservesUntouchedCallbackState(t *testing.T) {
	s := &mainState{
		composerSkillsOpen:   true,
		composerSkillsFilter: "old",
	}
	start := captureComposerWriteback(ui.ListState{}, ui.ListState{}, false, "old", "", composer.ChipEdit{})
	cur := start
	cur.skillsFilter = "git"

	s.mu.Lock()
	s.commitComposerWritebackLocked(start, cur)
	s.mu.Unlock()

	if !s.composerSkillsOpen {
		t.Fatal("untouched composerSkillsOpen clobbered callback toggle")
	}
	if s.composerSkillsFilter != "git" {
		t.Fatalf("modified composerSkillsFilter not written back: %q", s.composerSkillsFilter)
	}
}

func TestCachedAssistantVendorsNonBlockingAsyncProbe(t *testing.T) {
	root := t.TempDir()
	t.Setenv(paths.EnvUserData, root)
	t.Setenv(paths.EnvVendor, filepath.Join(root, "vendor"))
	t.Setenv("DSH_HOME", filepath.Join(root, "dsh-home"))

	if err := os.MkdirAll(filepath.Join(paths.VendorDir(), "dsh-whale"), 0o755); err != nil {
		t.Fatal(err)
	}
	s := &mainState{}
	t.Cleanup(func() {
		s.mu.Lock()
		if s.lifecycleCancel != nil {
			s.lifecycleCancel()
		}
		s.mu.Unlock()
	})

	whale, dshbot := s.assistantService().CachedVendors()
	if whale || dshbot {
		t.Fatalf("expected cold first probe before async stat completes to return false/false, got %v/%v", whale, dshbot)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		whale, dshbot = s.assistantService().CachedVendors()
		if whale && !dshbot {
			return
		}
		time.Sleep(15 * time.Millisecond)
	}
	t.Fatalf("async vendor probe did not converge, got whale=%v dshbot=%v", whale, dshbot)
}

func TestEnsureCredentialsDraftSeededNonBlockingAndIsolated(t *testing.T) {
	root := t.TempDir()
	t.Setenv(paths.EnvUserData, root)
	t.Setenv(paths.EnvVendor, filepath.Join(root, "vendor"))
	t.Setenv("DSH_HOME", filepath.Join(root, "dsh-home"))

	s := &mainState{}
	t.Cleanup(func() {
		s.mu.Lock()
		if s.lifecycleCancel != nil {
			s.lifecycleCancel()
		}
		s.mu.Unlock()
	})

	s.ensureCredentialsDraftSeeded()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		seeded := s.credsDraft.seeded
		loading := s.credsSeedLoading
		s.mu.Unlock()
		if seeded && !loading {
			return
		}
		time.Sleep(15 * time.Millisecond)
	}
	t.Fatal("async credential seed did not finish within deadline")
}

func TestCachedWallpaperChoicesDeactivatesCleanly(t *testing.T) {
	root := t.TempDir()
	t.Setenv(paths.EnvUserData, root)
	t.Setenv(paths.EnvVendor, filepath.Join(root, "vendor"))
	t.Setenv("DSH_HOME", filepath.Join(root, "dsh-home"))

	s := &mainState{}
	t.Cleanup(func() {
		s.mu.Lock()
		if s.lifecycleCancel != nil {
			s.lifecycleCancel()
		}
		s.mu.Unlock()
	})

	_ = s.cachedWallpaperChoices(true)

	deadline := time.Now().Add(3 * time.Second)
	ready := false
	for time.Now().Before(deadline) {
		_ = s.cachedWallpaperChoices(true)
		s.mu.Lock()
		ready = s.wallpaperChoicesReady
		s.mu.Unlock()
		if ready {
			break
		}
		time.Sleep(15 * time.Millisecond)
	}
	if !ready {
		t.Fatal("async wallpaper choice load did not mark ready")
	}
	if got := s.cachedWallpaperChoices(false); got != nil {
		t.Fatalf("expected nil on inactive cachedWallpaperChoices, got %v", got)
	}
	s.mu.Lock()
	stillReady := s.wallpaperChoicesReady || s.wallpaperChoicesLoading
	s.mu.Unlock()
	if stillReady {
		t.Fatal("inactive cachedWallpaperChoices left ready/loading state set")
	}
}
