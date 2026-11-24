package config

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestAsyncUpdateCoalescesDoesNotHoldStoreLockAndFlushesDurably(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	s.SetAsyncDebounce(40 * time.Millisecond)

	var renames atomic.Int32
	inRename := make(chan struct{}, 1)
	releaseRename := make(chan struct{})
	var releaseOnce sync.Once
	s.rename = func(oldpath, newpath string) error {
		n := renames.Add(1)
		if n == 1 {
			select {
			case inRename <- struct{}{}:
			default:
			}
			<-releaseRename
		}
		return os.Rename(oldpath, newpath)
	}

	// Burst 10 rapid updates; all should return immediately and update memory.
	start := time.Now()
	for i := 1; i <= 10; i++ {
		port := 4000 + i
		if err := s.UpdateAsync(func(c *Config) { c.Port = port }); err != nil {
			t.Fatalf("UpdateAsync %d: %v", i, err)
		}
	}
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("UpdateAsync burst blocked caller for %v", elapsed)
	}
	if got := s.Get().Port; got != 4010 {
		t.Fatalf("optimistic memory state = %d, want 4010", got)
	}

	// Wait until the single background writer enters disk rename, and prove
	// Store.mu is NOT held across disk I/O by calling IsFirstRun() and
	// UpdateAsync() concurrently while rename is blocked.
	select {
	case <-inRename:
	case <-time.After(2 * time.Second):
		t.Fatal("background debounced writer did not start")
	}
	doneCheck := make(chan struct{})
	go func() {
		_ = s.IsFirstRun()
		_ = s.UpdateAsync(func(c *Config) { c.Locale = "en" })
		close(doneCheck)
	}()
	select {
	case <-doneCheck:
	case <-time.After(500 * time.Millisecond):
		releaseOnce.Do(func() { close(releaseRename) })
		t.Fatal("Store.mu was held across disk rename I/O")
	}
	releaseOnce.Do(func() { close(releaseRename) })

	if err := s.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if n := renames.Load(); n > 3 {
		t.Fatalf("expected coalesced writes (<=3), got %d", n)
	}

	// Verify durability by reopening from disk.
	reopened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := reopened.Get(); got.Port != 4010 || got.Locale != "en" {
		t.Fatalf("flushed config lost edits: port=%d locale=%q", got.Port, got.Locale)
	}

	// Synchronous Update right after an async write must flush cleanly without ErrConflict.
	if err := s.UpdateAsync(func(c *Config) { c.SidebarWidth = 340 }); err != nil {
		t.Fatal(err)
	}
	if err := s.Update(func(c *Config) { c.TerminalHeight = 300 }); err != nil {
		t.Fatalf("sync Update after UpdateAsync failed: %v", err)
	}
	reopened2, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := reopened2.Get(); got.SidebarWidth != 340 || got.TerminalHeight != 300 {
		t.Fatalf("mixed async+sync updates lost fields: %+v", got)
	}
}

func TestSaveWorkspaceNavigationPreferencesAsyncAndFlush(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	s.SetAsyncDebounce(30 * time.Millisecond)

	prefs := WorkspaceNavigationPreferences{
		GroupBy:        "workspace-tree",
		OrderBy:        "updated",
		ArchivedFilter: "show",
		Collapsed:      map[string]bool{"ws-1": true, "ws-2": false},
	}
	if err := s.SaveWorkspaceNavigationPreferencesAsync("host-a", prefs); err != nil {
		t.Fatal(err)
	}
	// Immediate in-memory read sees the optimistic state.
	got, err := s.WorkspaceNavigationPreferences("host-a")
	if err != nil {
		t.Fatal(err)
	}
	if got.GroupBy != "workspace-tree" || !got.Collapsed["ws-1"] {
		t.Fatalf("optimistic workspace prefs missing: %+v", got)
	}
	if err := s.Flush(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, FileName)); err != nil {
		t.Fatalf("config.json not written after Flush: %v", err)
	}
	reopened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	persisted, err := reopened.WorkspaceNavigationPreferences("host-a")
	if err != nil {
		t.Fatal(err)
	}
	if persisted.GroupBy != "workspace-tree" || persisted.OrderBy != "updated" || !persisted.Collapsed["ws-1"] {
		t.Fatalf("persisted workspace prefs mismatch: %+v", persisted)
	}
}

func TestAsyncUpdateConcurrentSafety(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	s.SetAsyncDebounce(5 * time.Millisecond)

	var wg sync.WaitGroup
	for g := 0; g < 6; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < 25; i++ {
				_ = s.UpdateAsync(func(c *Config) {
					c.Port = 3200 + id*100 + i
				})
				_ = s.Get()
			}
		}(g)
	}
	wg.Wait()
	if err := s.Flush(); err != nil {
		t.Fatalf("Flush after concurrent writes: %v", err)
	}
}
