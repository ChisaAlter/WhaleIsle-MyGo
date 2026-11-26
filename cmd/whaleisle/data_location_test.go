package main

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/desktop/importer"
	"whaleisle/internal/ui/i18n"
)

func TestImportLegacyDataNonBlockingAndRejectsConcurrentClick(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)

	src := t.TempDir()
	dst := t.TempDir()
	plan := importer.FirstRunPlan{Source: src, Dest: dst}

	entered := make(chan confirmAskRequest, 1)
	release := make(chan struct{})
	var calls atomic.Int32

	s := &mainState{
		confirmDataLocation: func(req confirmAskRequest) bool {
			calls.Add(1)
			entered <- req
			<-release
			return false
		},
	}

	// Must return immediately on the UI thread instead of blocking on confirm.
	start := time.Now()
	s.importLegacyData(nil, plan)
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("importLegacyData blocked calling thread for %v", elapsed)
	}

	select {
	case req := <-entered:
		if req.Title != i18n.T(i18n.KeyFirstRunImportTitle) {
			t.Fatalf("unexpected confirm title %q", req.Title)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("confirmation was not dispatched in background")
	}

	// Second click while confirmation is still open must be rejected immediately
	// without dispatching a second prompt.
	s.importLegacyData(nil, plan)
	s.mu.Lock()
	errText := s.dataLocationErr
	importing := s.dataLocationImporting
	s.mu.Unlock()
	if errText != i18n.T(i18n.KeyDataLocationBusy) {
		t.Fatalf("double-click error = %q, want %q", errText, i18n.T(i18n.KeyDataLocationBusy))
	}
	if !importing {
		t.Fatal("dataLocationImporting cleared prematurely while confirmation was pending")
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("confirm calls = %d, want 1", got)
	}

	// Declining the prompt must release the pending flag cleanly.
	close(release)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		stillImporting := s.dataLocationImporting
		s.mu.Unlock()
		if !stillImporting {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("dataLocationImporting was not released after decline")
}

func TestImportLegacyDataConfirmRunsImportAndReleasesFlag(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)

	src := t.TempDir()
	dst := t.TempDir()
	cfgPath := filepath.Join(src, "config.json")
	if err := os.WriteFile(cfgPath, []byte(`{"theme":"system"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	plan := importer.PlanFirstRun(dst, src)

	s := &mainState{
		legacyImportKnown: true,
		confirmDataLocation: func(confirmAskRequest) bool {
			return true
		},
	}

	s.importLegacyData(nil, plan)

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		done := !s.dataLocationImporting && s.dataLocationAction != "" && s.dataLocationAction != i18n.T(i18n.KeyDataLocationImportRunning)
		s.mu.Unlock()
		if done {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	s.mu.Lock()
	importing := s.dataLocationImporting
	errText := s.dataLocationErr
	action := s.dataLocationAction
	known := s.legacyImportKnown
	s.mu.Unlock()
	if importing {
		t.Fatal("dataLocationImporting stayed true after import finished")
	}
	if errText != "" {
		t.Fatalf("unexpected import error: %q", errText)
	}
	if action == "" {
		t.Fatal("expected import summary action text")
	}
	if known {
		t.Fatal("legacyImportKnown was not invalidated after import")
	}
}

func TestImportLegacyDataDefaultConfirmWithoutWindowIsNonBlocking(t *testing.T) {
	plan := importer.FirstRunPlan{Source: t.TempDir(), Dest: t.TempDir()}
	s := &mainState{}

	start := time.Now()
	s.importLegacyData(nil, plan)
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("importLegacyData blocked calling thread for %v", elapsed)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		stillImporting := s.dataLocationImporting
		s.mu.Unlock()
		if !stillImporting {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("dataLocationImporting was not released when no window owner existed")
}
