package main

import (
	"context"
	"testing"
	"time"

	"whaleisle/internal/desktop/proactive"
)

func TestProactiveSourceControlsKeepUnrelatedDraft(t *testing.T) {
	s := &mainState{}
	dir := t.TempDir()
	s.openProactive(dir)
	wait := func() {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			s.mu.Lock()
			busy := s.proactivePanel.sourceBusy
			s.mu.Unlock()
			if !busy {
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
		t.Fatal("source mutation did not settle")
	}
	s.proactiveModel().OnSourceEdit("owner/repo")
	s.proactiveModel().OnSourceAdd()
	wait()
	m := s.proactiveModel()
	if m.SourceRepository != "" || len(m.State.Sources) != 1 || !m.State.Sources[0].Enabled {
		t.Fatal("source not saved", m.SourceError)
	}
	m.OnSourceEdit("owner/pending-draft")
	s.proactiveModel().OnSourceToggle("owner/repo", false)
	wait()
	m = s.proactiveModel()
	if m.SourceRepository != "owner/pending-draft" || m.State.Sources[0].Enabled {
		t.Fatal("pause cleared another source draft")
	}
	reopened, err := proactive.Open(dir)
	if err != nil || reopened.Sources()[0].Enabled {
		t.Fatal("pause did not persist", err)
	}
	m.OnSourceEdit("owner/repo?unsafe")
	s.proactiveModel().OnSourceAdd()
	wait()
	m = s.proactiveModel()
	if m.SourceError == "" || m.SourceRepository != "owner/repo?unsafe" {
		t.Fatal("failed source configuration lost draft")
	}
	if status, err := s.deliverProactiveNotice(context.Background(), proactive.Notice{ID: "test"}); err != nil || status != "unavailable" {
		t.Fatal("no-window delivery did not remain in-app", status, err)
	}
}
