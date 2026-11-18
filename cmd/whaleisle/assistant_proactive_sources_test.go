package main

import (
	"context"
	"testing"
	"time"

	"whaleisle/internal/desktop/proactive"
)

func TestProactiveSourceControlsKeepUnrelatedDraft(t *testing.T) {
	s := &mainState{}
	svc := s.assistantService()
	t.Cleanup(svc.SuspendAll)
	dir := t.TempDir()
	s.openProactive(dir)
	wait := func() {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			if !svc.ProactiveSnapshot().SourceBusy {
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
		t.Fatal("source mutation did not settle")
	}
	svc.ProactiveModel().OnSourceEdit("owner/repo")
	svc.ProactiveModel().OnSourceAdd()
	wait()
	m := svc.ProactiveModel()
	if m.SourceRepository != "" || len(m.State.Sources) != 1 || !m.State.Sources[0].Enabled {
		t.Fatal("source not saved", m.SourceError)
	}
	m.OnSourceEdit("owner/pending-draft")
	svc.ProactiveModel().OnSourceToggle("owner/repo", false)
	wait()
	m = svc.ProactiveModel()
	if m.SourceRepository != "owner/pending-draft" || m.State.Sources[0].Enabled {
		t.Fatal("pause cleared another source draft")
	}
	reopened, err := proactive.Open(dir)
	if err != nil || reopened.Sources()[0].Enabled {
		t.Fatal("pause did not persist", err)
	}
	m.OnSourceEdit("owner/repo?unsafe")
	svc.ProactiveModel().OnSourceAdd()
	wait()
	m = svc.ProactiveModel()
	if m.SourceError == "" || m.SourceRepository != "owner/repo?unsafe" {
		t.Fatal("failed source configuration lost draft")
	}
	if status, err := svc.DeliverProactiveNotice(context.Background(), proactive.Notice{ID: "test"}); err != nil || status != "unavailable" {
		t.Fatal("no-window delivery did not remain in-app", status, err)
	}
}
