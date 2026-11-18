package diffsurface

import (
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/git"
	"whaleisle/internal/ui/i18n"
)

func TestAsyncDiffLoaderPrebuildsFilesWidthsAndListOffUIThread(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	clearPrebuiltPatchCache()
	t.Cleanup(clearPrebuiltPatchCache)

	patch := buildLargeSyntheticPatch(2, 200)
	root := t.TempDir()
	ready := make(chan struct{})
	withTestGitOps(t, func(ops *diffsurfaceGitOps) {
		ops.isRepository = func(string) bool { return true }
		ops.changes = func(string) ([]git.Change, error) {
			return []git.Change{{Path: "pkg/file_00.go", Worktree: 'M'}}, nil
		}
		ops.diffHEAD = func(string, string) (string, error) {
			return patch, nil
		}
	})

	w := &Workbench{
		Root:        root,
		Loaded:      true,
		Repository:  true,
		AgainstHEAD: true,
		Changes:     []git.Change{{Path: "pkg/file_00.go", Worktree: 'M'}},
	}
	defer w.ClosePane()

	var invalidations atomic.Int32
	w.RequestDiffAsync(root, "", true, func() {
		invalidations.Add(1)
		select {
		case <-ready:
		default:
			close(ready)
		}
	})

	waitUntil(t, "background diff prebuild", func() bool {
		return invalidations.Load() > 0
	}, nil)

	var text, errMsg string
	gotText, gotErr, ok, loading := w.PollDiffAsync(root, "")
	if !ok || loading || gotErr != "" || gotText != patch {
		t.Fatalf("unexpected PollDiffAsync state: ok=%v loading=%v err=%q len=%d", ok, loading, gotErr, len(gotText))
	}
	text, errMsg = gotText, gotErr

	if len(w.View.cachedItems) == 0 || len(w.View.files) != 2 {
		t.Fatalf("expected PollDiffAsync to install prebuilt files and immutable item list: files=%d items=%d", len(w.View.files), len(w.View.cachedItems))
	}
	for i, f := range w.View.files {
		if !f.metricsReady || f.uniMinW <= 0 || f.splitColW <= 0 || len(f.uniLines) == 0 {
			t.Fatalf("file %d missing pre-measured metrics: %+v", i, f)
		}
	}

	m := Model{
		WorkspaceRoot: root,
		Workbench:     w,
		Text:          &text,
		ErrMsg:        &errMsg,
	}
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, m)
	}, 900, 420)
	tt.Frame()

	if w.View.uiParseCalls != 0 || w.View.uiMeasureCalls != 0 || w.View.uiBuildCalls != 0 {
		t.Fatalf("UI frame performed work that should stay off-thread: parse=%d measure=%d build=%d", w.View.uiParseCalls, w.View.uiMeasureCalls, w.View.uiBuildCalls)
	}

	// Switching to Split + Wrap reuses prebuilt parse/metrics with 0 UI parses or width scans.
	if err := tt.Click(i18n.T(i18n.KeySurfacesDiffSplit)); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(i18n.T(i18n.KeySurfacesDiffWrap)); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if w.View.uiParseCalls != 0 || w.View.uiMeasureCalls != 0 {
		t.Fatalf("mode toggle re-parsed or re-measured on UI thread: parse=%d measure=%d", w.View.uiParseCalls, w.View.uiMeasureCalls)
	}

	// Fallback loader (Workbench == nil) also prebuilds immutable doc off-thread.
	clearPrebuiltPatchCache()
	var fbText, fbErr string
	fbDone := make(chan struct{})
	loadingFB := EnsureDiffAsync(nil, root, "", &fbText, &fbErr, func() {
		select {
		case <-fbDone:
		default:
			close(fbDone)
		}
	})
	if !loadingFB {
		t.Fatal("expected fallback loader to start async read")
	}
	waitUntil(t, "fallback loader prebuild", func() bool {
		return !EnsureDiffAsync(nil, root, "", &fbText, &fbErr, nil)
	}, nil)
	var standalone patchViewState
	items, maxW := buildPatchListItems(&standalone, fbText)
	if len(items) == 0 || maxW <= 0 {
		t.Fatalf("expected fallback prebuilt items, got len=%d maxW=%v", len(items), maxW)
	}
	if standalone.uiParseCalls != 0 || standalone.uiMeasureCalls != 0 || standalone.uiBuildCalls != 0 {
		t.Fatalf("standalone view did redundant UI work after background prebuild: parse=%d measure=%d build=%d", standalone.uiParseCalls, standalone.uiMeasureCalls, standalone.uiBuildCalls)
	}
}
