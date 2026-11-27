package diffsurface

import (
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/git"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
)

func TestDiffSurfaceSemanticTokensLightAndDark(t *testing.T) {
	hardcodedGreen := ui.RGB(0x22, 0xc5, 0x5e)
	for _, dark := range []bool{false, true} {
		var gotAdd, gotDel ui.Color
		tt := ui.NewTester(func(c *ui.Context) {
			tokens := design.For(c)
			gotAdd = tokens.SuccessForeground
			gotDel = tokens.DestructiveForeground
		}, 400, 300)
		tt.SetDark(dark)
		tt.Frame()
		if !dark && gotAdd == hardcodedGreen {
			t.Fatalf("light mode SuccessForeground unexpectedly equals low-contrast hardcoded green %+v", hardcodedGreen)
		}
		if gotAdd.A == 0 || gotDel.A == 0 {
			t.Fatalf("dark=%v semantic diff tokens must be opaque: add=%+v del=%+v", dark, gotAdd, gotDel)
		}
	}
}

func TestCommitDialogFileRowClickTogglesSelection(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })

	w := &Workbench{
		Root:       "/repo",
		Loaded:     true,
		Repository: true,
		CommitOpen: true,
		Changes: []git.Change{
			{Path: "alpha.go", Index: 'M', Worktree: ' ', Added: 5, Deleted: 2},
			{Path: "beta.go", Index: ' ', Worktree: 'M', Added: 1, Deleted: 0},
		},
	}
	m := Model{Workbench: w, WorkspaceRoot: "/repo", ThemeID: "deepseek"}
	tt := ui.NewTester(func(c *ui.Context) {
		commitDialog(c, m, c.Theme().Text, c.Theme().TextMuted)
	}, 800, 640)

	if len(w.selectedCommitPaths()) != 2 {
		t.Fatalf("initial selected = %v, want 2", w.selectedCommitPaths())
	}
	// Click on the file path text area (right side of the row, away from the checkbox).
	rect, ok := tt.Find("alpha.go")
	if !ok {
		t.Fatalf("missing alpha.go row; texts=%q", tt.Texts())
	}
	tt.ClickAt(rect.X+rect.W/2, rect.Y+rect.H/2)
	tt.Frame()
	if got := w.selectedCommitPaths(); len(got) != 1 || got[0] != "beta.go" {
		t.Fatalf("after clicking row, selected = %v, want [beta.go]", got)
	}

	// Clicking again restores selection.
	tt.ClickAt(rect.X+rect.W/2, rect.Y+rect.H/2)
	tt.Frame()
	if got := w.selectedCommitPaths(); len(got) != 2 {
		t.Fatalf("after second click, selected = %v, want 2", got)
	}
}

func TestCaptionBranchesAutoFocusAndProgressEsc(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })

	w := &Workbench{
		Root:   "/repo",
		Loaded: true,
		Branches: []git.Branch{
			{Name: "main", Current: true},
			{Name: "feat/ux-polish"},
		},
	}
	open := true
	m := Model{Workbench: w, WorkspaceRoot: "/repo", ThemeID: "deepseek"}
	tt := ui.NewTester(func(c *ui.Context) {
		anchor := ui.ButtonBase(c).Size(120, 28).Label("Branches")
		CaptionBranches(c, m, anchor, &open)
	}, 720, 600)

	// Filter input is auto-focused on open so typing updates BranchQuery immediately.
	tt.Type("ux")
	tt.Frame()
	if w.BranchQuery != "ux" {
		t.Fatalf("BranchQuery = %q, want %q", w.BranchQuery, "ux")
	}

	// Completed CaptionActivity progress overlay sits clear of the bottom composer
	// send area at 720x600 and dismisses on Escape.
	w.Progress = git.NewProgress("/repo", "Switching branch…", func() {})
	w.Progress.Finish("Switched to feat/ux-polish", false)
	ttProg := ui.NewTester(func(c *ui.Context) {
		CaptionActivity(c, m)
	}, 720, 600)

	rect, ok := ttProg.Find("Dismiss progress")
	if !ok {
		t.Fatalf("missing Dismiss progress button; texts=%q", ttProg.Texts())
	}
	// Must stay in upper area, well clear of bottom composer send button (bottom 120 DIP).
	if rect.Y+rect.H > 600-120 {
		t.Fatalf("progress overlay overlaps bottom composer area at 720x600: rect=%+v", rect)
	}

	ttProg.Key(0, ui.KeyEscape)
	ttProg.Frame()
	if w.Progress != nil {
		t.Fatal("Escape did not dismiss completed caption-branch-progress")
	}
}

func TestWorkbenchLoaderDiffCacheLRUBounds(t *testing.T) {
	t.Run("entry-limit-and-lru-touch", func(t *testing.T) {
		l := newWorkbenchLoader()
		l.diffCacheMaxEntries = 4
		l.diffCacheMaxBytes = 1 << 20

		keyFor := func(i int) diffRequestKey {
			return diffRequestKey{root: "/repo", gen: 1, path: fmt.Sprintf("f%d.go", i), againstHEAD: true}
		}
		l.mu.Lock()
		for i := 0; i < 4; i++ {
			l.putDiffCacheLocked(keyFor(i), diffCacheEntry{text: "patch", seq: uint64(i + 1)})
		}
		// Touch f0 so f1 becomes the oldest LRU entry.
		l.touchDiffCacheKeyLocked(keyFor(0))
		l.putDiffCacheLocked(keyFor(4), diffCacheEntry{text: "patch", seq: 5})
		gotLen := len(l.diffCache)
		_, has0 := l.diffCache[keyFor(0)]
		_, has1 := l.diffCache[keyFor(1)]
		_, has4 := l.diffCache[keyFor(4)]
		l.mu.Unlock()

		if gotLen != 4 {
			t.Fatalf("diffCache len = %d, want 4", gotLen)
		}
		if !has0 || has1 || !has4 {
			t.Fatalf("LRU eviction mismatch: has0=%v has1=%v has4=%v", has0, has1, has4)
		}
	})

	t.Run("default-32-entries-budget", func(t *testing.T) {
		l := newWorkbenchLoader()
		l.mu.Lock()
		for i := 0; i < 60; i++ {
			k := diffRequestKey{root: "/repo", gen: 1, path: fmt.Sprintf("file-%02d.go", i), againstHEAD: true}
			l.putDiffCacheLocked(k, diffCacheEntry{text: "diff --git a/x b/x\n+hello\n", seq: uint64(i + 1)})
		}
		gotLen := len(l.diffCache)
		gotOrderLen := len(l.diffCacheOrder)
		gotBytes := l.diffCacheBytes
		l.mu.Unlock()

		if gotLen > defaultDiffCacheMaxEntries {
			t.Fatalf("diffCache len = %d, exceeds default max %d", gotLen, defaultDiffCacheMaxEntries)
		}
		if gotOrderLen != gotLen {
			t.Fatalf("diffCacheOrder len = %d != diffCache len = %d", gotOrderLen, gotLen)
		}
		if gotBytes <= 0 || gotBytes > defaultDiffCacheMaxBytes {
			t.Fatalf("diffCacheBytes = %d out of (0, %d]", gotBytes, defaultDiffCacheMaxBytes)
		}
	})

	t.Run("byte-budget-and-oversized-entry", func(t *testing.T) {
		l := newWorkbenchLoader()
		l.diffCacheMaxEntries = 32
		l.diffCacheMaxBytes = 4096

		payload := strings.Repeat("a", 1500)
		l.mu.Lock()
		for i := 0; i < 5; i++ {
			k := diffRequestKey{root: "/repo", gen: 1, path: fmt.Sprintf("big-%d.go", i), againstHEAD: true}
			l.putDiffCacheLocked(k, diffCacheEntry{text: payload, seq: uint64(i + 1)})
		}
		bytesAfter := l.diffCacheBytes
		lenAfter := len(l.diffCache)

		// Oversized single entry (> budget) must not exceed byte budget.
		huge := diffRequestKey{root: "/repo", gen: 1, path: "huge.go", againstHEAD: true}
		l.putDiffCacheLocked(huge, diffCacheEntry{text: strings.Repeat("z", 8192), seq: 99})
		bytesHuge := l.diffCacheBytes
		lenHuge := len(l.diffCache)
		l.mu.Unlock()

		if bytesAfter > 4096 || lenAfter != 2 {
			t.Fatalf("byte budget violated: bytes=%d len=%d (want <=4096, len=2)", bytesAfter, lenAfter)
		}
		if bytesHuge != 0 || lenHuge != 0 {
			t.Fatalf("oversized entry retained in cache: bytes=%d len=%d", bytesHuge, lenHuge)
		}
	})
}
