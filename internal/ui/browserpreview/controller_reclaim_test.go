package browserpreview

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	desktop "whaleisle/internal/desktop/webview"
)

func waitReclaimCond(t *testing.T, timeout time.Duration, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal(msg)
}

func TestControllerHideReclaimsHostAfterTTLAndResumesOnSelect(t *testing.T) {
	b := NewController(nil)
	t.Cleanup(b.Close)
	b.hideTTL = 25 * time.Millisecond

	var closes atomic.Int32
	var resumedURL string
	var resumedPartition string
	b.closeHostFn = func(h *desktop.Host) error {
		closes.Add(1)
		return nil
	}
	b.newHostFn = func(opts desktop.HostOptions) (*desktop.Host, error) {
		resumedURL = opts.URL
		resumedPartition = opts.BrowserPartition
		return &desktop.Host{}, nil
	}

	firstHost := &desktop.Host{}
	secondHost := &desktop.Host{}
	first := &Tab{
		ID:          1,
		workspace:   browserWorkspace{key: "ws-alpha"},
		Draft:       "https://example.com/docs",
		resumeTitle: "Example Docs",
		Host:        firstHost,
	}
	second := &Tab{
		ID:        2,
		workspace: browserWorkspace{key: "ws-alpha"},
		Draft:     "https://example.com/other",
		Host:      secondHost,
	}
	b.Tabs = []*Tab{first, second}
	b.Active = 0

	// Switching to tab 1 hides tab 0 synchronously without immediately destroying it.
	b.Select(1)
	hostNow, reclaimedNow := b.tabHostSnapshot(first)
	if hostNow != firstHost || reclaimedNow {
		t.Fatalf("Select(1) destroyed background host synchronously: host=%v reclaimed=%v", hostNow, reclaimedNow)
	}
	if got := closes.Load(); got != 0 {
		t.Fatalf("unexpected immediate close calls = %d", got)
	}

	// After TTL, hidden tab 0 is reclaimed while active tab 1 stays untouched.
	waitReclaimCond(t, time.Second, func() bool {
		h, r := b.tabHostSnapshot(first)
		return h == nil && r && closes.Load() == 1
	}, "hidden tab host was not reclaimed after TTL")

	if host1, rec1 := b.tabHostSnapshot(second); host1 != secondHost || rec1 {
		t.Fatalf("active tab was unexpectedly reclaimed: host=%v reclaimed=%v", host1, rec1)
	}
	if first.resumeURL != "https://example.com/docs" || first.Draft != "https://example.com/docs" {
		t.Fatalf("resume state lost after reclaim: resumeURL=%q Draft=%q", first.resumeURL, first.Draft)
	}
	if title := browserTabDisplayTitle(first, false); title != "Example Docs" {
		t.Fatalf("tab title after reclaim = %q, want %q", title, "Example Docs")
	}

	// Re-selecting reclaimed tab recreates Host on demand with original URL and partition.
	b.Select(0)
	recreatedHost, recAfter := b.tabHostSnapshot(first)
	if recreatedHost == nil || recreatedHost == firstHost || recAfter {
		t.Fatalf("reclaimed tab did not recreate Host cleanly: host=%p reclaimed=%v", recreatedHost, recAfter)
	}
	if resumedURL != "https://example.com/docs" || resumedPartition != "ws-alpha" {
		t.Fatalf("unexpected resume opts: url=%q partition=%q", resumedURL, resumedPartition)
	}
	if first.resumeURL != "" {
		t.Fatalf("resumeURL not cleared after recreate: %q", first.resumeURL)
	}

	// Hide entire controller reclaims even the active tab after TTL, then resumes on view/Select.
	b.Hide()
	waitReclaimCond(t, time.Second, func() bool {
		_, rec0 := b.tabHostSnapshot(first)
		_, rec1 := b.tabHostSnapshot(second)
		return rec0 && rec1 && closes.Load() == 3
	}, "hidden controller did not reclaim both tabs after TTL")
}

func TestControllerHideExemptsMiniPlayerAndCancelsOnCloseOrSelect(t *testing.T) {
	b := NewController(nil)
	t.Cleanup(b.Close)
	b.hideTTL = 30 * time.Millisecond

	var closes atomic.Int32
	b.closeHostFn = func(h *desktop.Host) error {
		closes.Add(1)
		return nil
	}

	miniHost := &desktop.Host{}
	otherHost := &desktop.Host{}
	first := &Tab{ID: 1, Draft: "https://example.com/video", Host: miniHost}
	second := &Tab{ID: 2, Draft: "https://example.com/chat", Host: otherHost}
	b.Tabs = []*Tab{first, second}
	b.Active = 0
	b.mini = &browserMiniPlayer{tab: first, host: miniHost}
	b.setMiniTab(first)

	// Re-selecting within TTL cancels the reclaim timer for second.
	b.Hide()
	b.Select(1)
	time.Sleep(60 * time.Millisecond)

	if h, r := b.tabHostSnapshot(first); h != miniHost || r {
		t.Fatalf("mini-player tab was reclaimed while active: host=%v reclaimed=%v", h, r)
	}
	if h, r := b.tabHostSnapshot(second); h != otherHost || r {
		t.Fatalf("re-selected active tab was reclaimed: host=%v reclaimed=%v", h, r)
	}
	if closes.Load() != 0 {
		t.Fatalf("expected 0 closes while mini-player and active tab stayed live, got %d", closes.Load())
	}

	// Closing mini-player while inactive starts TTL reclaim for first.
	b.CloseMiniPlayer()
	waitReclaimCond(t, time.Second, func() bool {
		h, r := b.tabHostSnapshot(first)
		return h == nil && r && closes.Load() == 1
	}, "tab did not reclaim after closing inactive mini-player")

	// CloseTab / Close cancel any pending timers without double-closing Host.
	b.Hide()
	b.CloseTab(1)
	if closes.Load() != 2 {
		t.Fatalf("expected CloseTab to close secondHost once (total=2), got %d", closes.Load())
	}
	time.Sleep(60 * time.Millisecond)
	if closes.Load() != 2 {
		t.Fatalf("stale reclaim timer fired after CloseTab/Close: closes=%d", closes.Load())
	}
}

func TestControllerResumeFailureClearsReclaimedAndRestoredTabStaysLazy(t *testing.T) {
	b := NewController(nil)
	t.Cleanup(b.Close)

	var attempts atomic.Int32
	b.newHostFn = func(opts desktop.HostOptions) (*desktop.Host, error) {
		attempts.Add(1)
		return nil, errors.New("webview unavailable")
	}

	failed := &Tab{ID: 1, Draft: "https://example.com/a", resumeURL: "https://example.com/a", reclaimed: true}
	diskRestored := &Tab{ID: 2, Draft: "https://example.com/b", resumeURL: "https://example.com/b", reclaimed: false}
	b.Tabs = []*Tab{failed, diskRestored}
	b.Active = 1

	// Restored-from-disk tab (reclaimed == false) must stay lazy on Select.
	b.Select(1)
	if attempts.Load() != 0 {
		t.Fatalf("disk-restored tab triggered unexpected auto-navigation attempts=%d", attempts.Load())
	}

	// Reclaimed tab attempts recreation once and records Error instead of looping.
	b.Select(0)
	b.Select(0)
	if attempts.Load() != 1 {
		t.Fatalf("expected single recreate attempt on failure, got %d", attempts.Load())
	}
	if failed.Error != "webview unavailable" || failed.reclaimed {
		t.Fatalf("unexpected failed state: Error=%q reclaimed=%v", failed.Error, failed.reclaimed)
	}
	if failed.resumeURL != "https://example.com/a" {
		t.Fatalf("expected resumeURL preserved for manual refresh retry, got %q", failed.resumeURL)
	}
}

func TestControllerSuspendImmediatelyReleasesHostsAndRetainsTabs(t *testing.T) {
	b := NewController(nil)
	t.Cleanup(b.Close)
	b.hideTTL = time.Hour
	closes := 0
	b.closeHostFn = func(*desktop.Host) error { closes++; return nil }
	var resumed desktop.HostOptions
	b.newHostFn = func(opts desktop.HostOptions) (*desktop.Host, error) {
		resumed = opts
		return &desktop.Host{}, nil
	}
	firstHost, secondHost := &desktop.Host{}, &desktop.Host{}
	first := &Tab{ID: 1, Host: firstHost, Draft: "https://example.com/a", workspace: browserWorkspace{key: "workspace-a"}}
	second := &Tab{ID: 2, Host: secondHost, Draft: "https://example.com/b"}
	b.Tabs, b.Active = []*Tab{first, second}, 0
	b.Suspend()
	if closes != 2 || len(b.Tabs) != 2 || first.Host != nil || second.Host != nil || !first.reclaimed || first.hideTimer != nil || second.hideTimer != nil {
		t.Fatalf("route exit retained native hosts or discarded tabs: closes=%d tabs=%d", closes, len(b.Tabs))
	}
	if first.Draft != "https://example.com/a" || second.Draft != "https://example.com/b" {
		t.Fatal("route exit discarded tab drafts")
	}
	staleGeneration := first.hideGen
	b.Select(0)
	if first.Host == nil || resumed.URL != "https://example.com/a" || resumed.BrowserPartition != "workspace-a" {
		t.Fatal("explicit return failed to recreate the selected tab")
	}
	b.reclaimHiddenTab(first, firstHost, staleGeneration)
	if closes != 2 || first.Host == nil {
		t.Fatal("a stale hidden-tab timer closed the newly resumed page")
	}
}

func TestControllerSuspendKeepsExplicitMiniPlayerLease(t *testing.T) {
	b := NewController(nil)
	t.Cleanup(b.Close)
	closes := 0
	b.closeHostFn = func(*desktop.Host) error { closes++; return nil }
	host := &desktop.Host{}
	tab := &Tab{ID: 1, Host: host, Draft: "https://example.com/video"}
	b.Tabs, b.Active = []*Tab{tab}, 0
	b.mini = &browserMiniPlayer{tab: tab, host: host}
	b.setMiniTab(tab)
	b.Suspend()
	if tab.Host != host || tab.reclaimed || closes != 0 {
		t.Fatal("leaving the browser revoked its explicitly detached mini-player")
	}
}
