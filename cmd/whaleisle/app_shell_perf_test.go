package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/config"
	"whaleisle/internal/ui/composer"
	"whaleisle/internal/ui/diffsurface"
	"whaleisle/internal/ui/files"
	"whaleisle/internal/ui/settings"
)

func TestWritebackMutationRevisionsReplaceDeepEqual(t *testing.T) {
	// AST guard: main_hotpath_cache.go must not import reflect or call DeepEqual/cloneBoolMapForDiff.
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, "main_hotpath_cache.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, imp := range parsed.Imports {
		if strings.Trim(imp.Path.Value, `"`) == "reflect" {
			t.Fatal("main_hotpath_cache.go still imports reflect")
		}
	}
	ast.Inspect(parsed, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok && (id.Name == "DeepEqual" || id.Name == "cloneBoolMapForDiff") {
			t.Fatalf("main_hotpath_cache.go still references %s", id.Name)
		}
		return true
	})

	s := &mainState{
		fileTabs:         files.Tabs{Root: "/initial"},
		gitWorkbench:     diffsurface.Workbench{Root: "/initial"},
		fileExpanded:     map[string]bool{"a": true},
		thinkingExpanded: map[string]bool{"m1": true},
	}

	start := mainViewWritebackSnapshot{
		revs:             s.writebackRevs,
		fileTabs:         s.fileTabs,
		gitWorkbench:     s.gitWorkbench,
		fileExpanded:     s.fileExpanded,
		thinkingExpanded: s.thinkingExpanded,
	}

	// Simulate an uncontested UI-frame mutation to fileTabs and fileExpanded,
	// plus a concurrent external mutation to gitWorkbench that bumps its revision.
	cur := start
	cur.fileTabs.Root = "/ui-updated"
	cur.fileExpanded = map[string]bool{"a": true, "b": true}
	cur.gitWorkbench.Root = "/stale-frame"

	s.mu.Lock()
	s.gitWorkbench.Root = "/concurrent-worker"
	s.markGitWorkbenchMutatedLocked()
	s.commitMainViewWritebackLocked(start, cur)
	s.mu.Unlock()

	if s.fileTabs.Root != "/ui-updated" {
		t.Fatalf("uncontested fileTabs writeback failed: got %q", s.fileTabs.Root)
	}
	if !s.fileExpanded["b"] {
		t.Fatalf("uncontested fileExpanded writeback failed: %+v", s.fileExpanded)
	}
	if s.gitWorkbench.Root != "/concurrent-worker" {
		t.Fatalf("concurrent gitWorkbench mutation was clobbered: got %q", s.gitWorkbench.Root)
	}

	// Also verify commitGitWorkbenchByRev respects revision gating.
	rev := s.writebackRevs.gitWorkbench
	s.commitGitWorkbenchByRev(rev, diffsurface.Workbench{Root: "/workspace-page-update"})
	if s.gitWorkbench.Root != "/workspace-page-update" {
		t.Fatalf("commitGitWorkbenchByRev did not commit matching revision: %q", s.gitWorkbench.Root)
	}
	s.mu.Lock()
	s.gitWorkbench.Root = "/newer-external"
	s.markGitWorkbenchMutatedLocked()
	s.mu.Unlock()
	s.commitGitWorkbenchByRev(rev, diffsurface.Workbench{Root: "/stale-overwrite"})
	if s.gitWorkbench.Root != "/newer-external" {
		t.Fatalf("commitGitWorkbenchByRev clobbered newer revision: %q", s.gitWorkbench.Root)
	}

	// Composer ChipEdit uses direct struct comparison without reflection.
	compStart := captureComposerWriteback(s.writebackRevs.composerSuggest, s.writebackRevs.composerSkills, ui.ListState{}, ui.ListState{}, false, "", "", composer.ChipEdit{})
	compCur := compStart
	compCur.chipEdit = composer.ChipEdit{Open: true, Start: 2, Orig: "@a", Draft: "@ab"}
	s.mu.Lock()
	s.commitComposerWritebackLocked(compStart, compCur)
	s.mu.Unlock()
	if s.composerChipEdit != compCur.chipEdit {
		t.Fatalf("composerChipEdit writeback mismatch: %+v", s.composerChipEdit)
	}
}

func TestStoreRefreshCoalescerBatchesStreamingAndFlushesTerminalImmediately(t *testing.T) {
	var flushes atomic.Int32
	done := make(chan struct{}, 8)
	c := newStoreRefreshCoalescer(25*time.Millisecond, func() {
		flushes.Add(1)
		select {
		case done <- struct{}{}:
		default:
		}
	})
	defer c.Close()

	// Leading-edge notification flushes immediately.
	c.Notify(false)
	if got := flushes.Load(); got != 1 {
		t.Fatalf("leading edge flushes = %d, want 1", got)
	}

	// Rapid burst of 20 streaming tokens within the 25ms window must coalesce into 1 trailing flush.
	for i := 0; i < 20; i++ {
		c.Notify(false)
	}
	if got := flushes.Load(); got != 1 {
		t.Fatalf("burst within window flushed synchronously: got %d, want 1", got)
	}

	// Wait for the coalesced trailing flush.
	select {
	case <-done: // drain leading signal
	default:
	}
	deadline := time.After(500 * time.Millisecond)
	for flushes.Load() < 2 {
		select {
		case <-done:
		case <-deadline:
			t.Fatalf("timed out waiting for coalesced flush; flushes=%d", flushes.Load())
		}
	}
	if got := flushes.Load(); got != 2 {
		t.Fatalf("expected exactly 2 flushes (leading + coalesced), got %d", got)
	}

	// Terminal/urgent notification within the window must flush immediately.
	c.Notify(false) // schedules deferred flush
	c.Notify(true)  // terminal state -> immediate flush
	if got := flushes.Load(); got != 3 {
		t.Fatalf("urgent terminal notification did not flush immediately: got %d, want 3", got)
	}
}

func TestAsyncConfigWriterOptimisticCoalescingAndFlush(t *testing.T) {
	dir := t.TempDir()
	store, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Long debounce proves UI calls do not block waiting on the timer and that
	// Flush() persists all queued edits deterministically before exit/restart.
	w := newAsyncConfigWriter(store, 2*time.Second, func(err error) {
		t.Errorf("unexpected async config error: %v", err)
	})
	defer w.Close()

	w.EnqueueUpdate(func(c *config.Config) { c.Theme = "midnight"; c.AppearanceExplicit = true })
	w.EnqueueUpdate(func(c *config.Config) { c.MinimalMode = true })
	w.EnqueueUpdate(func(c *config.Config) { v := true; c.GlassTheme = &v })
	w.EnqueueUpdate(func(c *config.Config) { c.Locale = "en" })
	w.EnqueueMerge(func(c *config.Config) { c.Workspace = "/opt/project" })

	// Effective() sees all optimistic mutations immediately before disk flush.
	eff := w.Effective(store)
	if eff.Theme != "midnight" || !eff.MinimalMode || !eff.GlassThemeOn() || eff.Locale != "en" || eff.Workspace != "/opt/project" {
		t.Fatalf("optimistic Effective config missing queued edits: %+v", eff)
	}

	// Flush writes everything to disk synchronously.
	if err := w.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	reopened, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := reopened.Get()
	if got.Theme != "midnight" || !got.MinimalMode || !got.GlassThemeOn() || got.Locale != "en" || got.Workspace != "/opt/project" {
		t.Fatalf("flushed config on disk mismatch: %+v", got)
	}
}

func TestAsyncWorkspaceViewPreferencesOptimisticAndFlush(t *testing.T) {
	dir := t.TempDir()
	store, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	vp := newWorkspaceViewPreferences(store)
	vp.enableAsync(2*time.Second, func(err error) {
		t.Errorf("unexpected async workspace preference error: %v", err)
	})

	prefs, err := vp.WorkspaceNavigationPreferences("host-1")
	if err != nil {
		t.Fatal(err)
	}
	prefs.GroupBy = "flat"
	prefs.Collapsed = map[string]bool{"ws-a": true}
	if err := vp.SaveWorkspaceNavigationPreferences("host-1", prefs); err != nil {
		t.Fatalf("async save returned error: %v", err)
	}
	// Immediate read sees optimistic collapsed/group state before disk flush.
	optimistic, err := vp.WorkspaceNavigationPreferences("host-1")
	if err != nil || optimistic.GroupBy != "flat" || !optimistic.Collapsed["ws-a"] {
		t.Fatalf("optimistic workspace preferences mismatch: %+v err=%v", optimistic, err)
	}
	if err := vp.Flush(); err != nil {
		t.Fatalf("workspace preference Flush failed: %v", err)
	}
	reopened, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, err := reopened.WorkspaceNavigationPreferences("host-1")
	if err != nil || got.GroupBy != "flat" || !got.Collapsed["ws-a"] {
		t.Fatalf("flushed workspace preferences on disk mismatch: %+v err=%v", got, err)
	}
}

func TestSettingsTabSWRCacheTTLAndOwnerInvalidation(t *testing.T) {
	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	cache := newSettingsTabSWR(30 * time.Second)
	cache.now = func() time.Time { return now }

	ownerA := settingsTabOwner{gen: 1, ref: "storeA"}
	ownerB := settingsTabOwner{gen: 2, ref: "storeB"}

	// First visit refreshes.
	if !cache.ShouldRefresh("models", ownerA) {
		t.Fatal("first visit to models must refresh")
	}
	// Switching back within 30s reuses cached store state without re-fetching.
	now = now.Add(15 * time.Second)
	if cache.ShouldRefresh("models", ownerA) {
		t.Fatal("revisit within 30s TTL should not refresh")
	}
	// Explicit refresh advances the TTL window.
	cache.MarkRefreshed("models", ownerA)
	now = now.Add(20 * time.Second)
	if cache.ShouldRefresh("models", ownerA) {
		t.Fatal("revisit within 30s of explicit refresh should not refresh")
	}
	// After 30s TTL expires, next visit refreshes in background.
	now = now.Add(11 * time.Second)
	if !cache.ShouldRefresh("models", ownerA) {
		t.Fatal("visit after 30s TTL expiry must refresh")
	}
	// Owner change (Host restart / session scope change) immediately refreshes.
	if !cache.ShouldRefresh("models", ownerB) {
		t.Fatal("owner change must bypass cached entry and refresh")
	}

	// Leaving usage section must NOT wipe usagePageReads loaded marker.
	s := &mainState{}
	usagePageReads.Store(s, "owner-1")
	t.Cleanup(func() { usagePageReads.Delete(s) })
	s.leaveSettingsSection(settings.Model{}, settings.SectionUsageStats)
	if _, ok := usagePageReads.Load(s); !ok {
		t.Fatal("leaveSettingsSection cleared usagePageReads marker instead of retaining SWR cache")
	}
}

func TestProcessApplicationMenuEmptyQueueEarlyExitAndCachedCommands(t *testing.T) {
	cfg, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s := &mainState{cfgStore: cfg, applicationMenu: &desktopApplicationMenu{}}
	// Calling with nil context must safely early-exit when pending queues are empty.
	s.processApplicationMenuFor(nil, nil)

	// Cached native commands slice must be reused across calls without rebuilding.
	if snap := s.pageShortcutPreferences().Get(); snap.Status != "ready" {
		t.Fatalf("shortcut prefs not ready: %+v", snap)
	}
	first := s.pageShortcutState()
	second := s.pageShortcutState()
	if len(first.nativeCommands) == 0 {
		t.Fatal("expected non-empty cached native commands")
	}
	if &first.nativeCommands[0] != &second.nativeCommands[0] {
		t.Fatal("pageShortcutState rebuilt nativeCommands slice instead of reusing cache")
	}
}

func TestWindowsStandaloneRouteTopCaptionModes(t *testing.T) {
	// Standalone routes on Windows use a borderless 12 DIP transparent drag strip
	// (never TitleBarWithActions, which paints a 1px hairline slicing "- □ ×").
	for _, route := range []mainRoute{routeBrowser, routeIM, routeRemote, routeUsage} {
		if got := mainTopCaptionModeForOS("windows", true, route); got != topCaptionWindowsStandaloneStrip {
			t.Fatalf("windows ready route %d caption mode = %v, want topCaptionWindowsStandaloneStrip", route, got)
		}
		if got := mainTopCaptionModeForOS("windows", false, route); got != topCaptionWindowsStandaloneStrip {
			t.Fatalf("windows offline route %d caption mode = %v, want topCaptionWindowsStandaloneStrip", route, got)
		}
	}
	// WebView and Workspace-managed routes reserve 0 DIP on Windows.
	for _, route := range []mainRoute{routeWebView, routeConversation, routeSettings, routeMarket, routePlugins, routeSchedule, routeAssistant, routeComponents} {
		if got := mainTopCaptionModeForOS("windows", true, route); got != topCaptionNone {
			t.Fatalf("windows ready route %d caption mode = %v, want topCaptionNone", route, got)
		}
	}
	// Offline maintenance routes use the borderless strip on Windows.
	for _, route := range []mainRoute{routeComponents, routeImport, routeStartupDiagnostics, routeUpdatePreferences} {
		if got := mainTopCaptionModeForOS("windows", false, route); got != topCaptionWindowsStandaloneStrip {
			t.Fatalf("windows offline maintenance route %d caption mode = %v, want topCaptionWindowsStandaloneStrip", route, got)
		}
	}
	// Non-Windows retains locked TitleBarWithActions for exclusive routes.
	if got := mainTopCaptionModeForOS("darwin", true, routeBrowser); got != topCaptionNonWindowsLocked {
		t.Fatalf("darwin routeBrowser caption mode = %v, want topCaptionNonWindowsLocked", got)
	}
	if got := mainTopCaptionModeForOS("darwin", true, routeConversation); got != topCaptionNonWindowsDragStrip {
		t.Fatalf("darwin routeConversation caption mode = %v, want topCaptionNonWindowsDragStrip", got)
	}
}

func TestNoRedundantEmptyWinUpdateInSettingsTabSelectionCallbacks(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "main.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	// Guard against regression of one-click-two-frames in Settings tab/toggle callbacks.
	checkedFields := map[string]bool{
		"OnSelectDesktop":      true,
		"OnSelectCredentials":  true,
		"OnSelectAbout":        true,
		"OnSelectAppearance":   true,
		"OnSelectInterface":    true,
		"OnSelectConversation": true,
		"OnSelectFeatures":     true,
		"OnSelectModels":       true,
		"OnSelectMCP":          true,
		"OnSelectPlugins":      true,
		"OnSelectSkills":       true,
		"OnSelectAgentPresets": true,
		"OnSelectVoice":        true,
		"OnSelectDeepSeekAPIs": true,
		"OnSelectNamespace":    true,
		"OnToggleTools":        true,
		"OnToggleTerminal":     true,
		"OnCloseTerminal":      true,
		"OnRestartTerminal":    true,
		"OnOpenImageLightbox":  true,
		"OnCloseImageLightbox": true,
	}
	found := map[string]int{}
	ast.Inspect(file, func(n ast.Node) bool {
		kv, ok := n.(*ast.KeyValueExpr)
		if !ok {
			return true
		}
		ident, ok := kv.Key.(*ast.Ident)
		if !ok || !checkedFields[ident.Name] {
			return true
		}
		fn, ok := kv.Value.(*ast.FuncLit)
		if !ok {
			return true
		}
		found[ident.Name]++
		ast.Inspect(fn.Body, func(inner ast.Node) bool {
			// Skip nested goroutines (async updates inside goroutines are valid).
			if _, isGo := inner.(*ast.GoStmt); isGo {
				return false
			}
			call, ok := inner.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "Update" || len(call.Args) != 1 {
				return true
			}
			argFn, ok := call.Args[0].(*ast.FuncLit)
			if ok && len(argFn.Body.List) == 0 {
				pos := fset.Position(call.Pos())
				t.Errorf("%s at %s:%d has redundant synchronous win.Update(func() {})", ident.Name, pos.Filename, pos.Line)
			}
			return true
		})
		return true
	})
	for name := range checkedFields {
		if found[name] == 0 {
			t.Errorf("expected to audit callback %s in main.go, found 0", name)
		}
	}
}

func TestAccountQuotaAndUpdaterModalUseCossButtons(t *testing.T) {
	for _, filename := range []string{"account_quota_notice.go", "updater_ui.go"} {
		src, err := os.ReadFile(filename)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(src), "ui.Button(") {
			t.Errorf("%s still contains bare ui.Button( instead of coss.Button", filename)
		}
		if !strings.Contains(string(src), "coss.Button(") {
			t.Errorf("%s does not use coss.Button", filename)
		}
	}
	// Ensure concurrency safety of settingsTabSWR under -race.
	cache := newSettingsTabSWR(30 * time.Second)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_ = cache.ShouldRefresh("models", settingsTabOwner{gen: 1})
				cache.MarkRefreshed("plugins", settingsTabOwner{gen: 1})
			}
		}()
	}
	wg.Wait()
}
