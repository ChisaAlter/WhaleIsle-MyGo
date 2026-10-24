package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	desktopupdate "whaleisle/internal/desktop/update"
	"whaleisle/internal/paths"
)

// TestUpdateDownloadProgressCopy keeps the visible download row honest: bytes
// are rendered in whole units and an unknown total shows activity, not 0 %.
func TestUpdateDownloadProgressCopy(t *testing.T) {
	if got := updateDownloadDetail(0, 0); got != "" {
		t.Fatalf("empty detail = %q", got)
	}
	if got := updateDownloadDetail(1536, 0); got != "2 KB" {
		t.Fatalf("received-only detail = %q", got)
	}
	if got := updateDownloadDetail(1500000, 29000000); got != "1.4 MB / 27.7 MB" {
		t.Fatalf("detail = %q", got)
	}
	if got := updateDownloadPercent(0, 0); got >= 0 {
		t.Fatalf("unknown total percent = %v", got)
	}
	if got := updateDownloadPercent(5, 10); got != 50 {
		t.Fatalf("percent = %v", got)
	}
	if got := updateDownloadPercent(20, 10); got != 100 {
		t.Fatalf("clamped percent = %v", got)
	}
	if got := formatUpdateBytes(3 * 1024 * 1024 * 1024); got != "3.00 GB" {
		t.Fatalf("gigabytes = %q", got)
	}
	if got := formatUpdateBytes(12); got != "12 B" {
		t.Fatalf("bytes = %q", got)
	}
}

// TestUpdateAvailableDetailKeepsTheEvidenceBounded mirrors the reference
// technicalDetails block: version, route, size and checksum, never the URL.
func TestUpdateAvailableDetailKeepsTheEvidenceBounded(t *testing.T) {
	res := desktopupdate.Result{
		Status:              desktopupdate.StatusUpdateAvailable,
		Version:             "2.0.0",
		ReleaseURL:          "https://example.com/secret-installer.exe",
		SHA256:              "abc123",
		Size:                2048,
		IncrementalPossible: true,
	}
	got := updateAvailableDetail(res, " CNB ")
	for _, want := range []string{"2.0.0", "cnb", "2 KB", "abc123"} {
		if !strings.Contains(got, want) {
			t.Fatalf("detail %q does not contain %q", got, want)
		}
	}
	if strings.Contains(got, "secret-installer") || strings.Contains(got, "example.com") {
		t.Fatalf("detail leaked the download URL: %q", got)
	}
	if got := updateAvailableDetail(desktopupdate.Result{}, ""); got != "" {
		t.Fatalf("empty detail = %q", got)
	}
}

// TestUpdateJournalRecordsCheckAndDownloadMilestones proves the evidence file
// stays outside the installation directory and keeps only whitelisted fields.
func TestUpdateJournalRecordsCheckAndDownloadMilestones(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvUserData, dir)
	s := &mainState{}
	s.recordUpdateAction(desktopupdate.JournalActionCheckRequested)
	s.recordUpdateCheckResult(desktopupdate.Result{Status: desktopupdate.StatusUpdateAvailable, Version: "2.0.0"}, nil)
	percent := 42.5
	s.recordUpdateState(desktopupdate.JournalState{Phase: desktopupdate.JournalPhaseDownloading, Version: "2.0.0", Percent: &percent})
	s.recordUpdateState(desktopupdate.JournalState{Phase: desktopupdate.JournalPhaseError, FailedOperation: "download", Message: "dial tcp: ETIMEDOUT"})
	files, err := filepath.Glob(filepath.Join(dir, updateJournalDirectory, "*.jsonl"))
	if err != nil || len(files) != 1 {
		t.Fatalf("journal files = %v (%v)", files, err)
	}
	data, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatalf("read journal: %v", err)
	}
	text := string(data)
	for _, want := range []string{
		`"event":"started"`,
		`"event":"check-requested"`,
		`"phase":"available"`,
		`"targetVersion":"2.0.0"`,
		`"percent":42`,
		`"errorCode":"ETIMEDOUT"`,
		`"failedOperation":"download"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("journal missing %s: %s", want, text)
		}
	}
	if strings.Contains(text, "dial tcp") {
		t.Fatalf("journal kept raw diagnostics: %s", text)
	}
}

// TestUpdateJournalStaysBestEffort keeps a journal storage failure from ever
// blocking an update: the helpers must not panic and must stop retrying.
func TestUpdateJournalStaysBestEffort(t *testing.T) {
	dir := t.TempDir()
	// A file where the journal directory belongs makes creation fail.
	blocked := filepath.Join(dir, updateJournalDirectory)
	if err := os.WriteFile(blocked, []byte("x"), 0o600); err != nil {
		t.Fatalf("fixture: %v", err)
	}
	t.Setenv(paths.EnvUserData, dir)
	s := &mainState{}
	s.recordUpdateAction(desktopupdate.JournalActionCheckRequested)
	s.recordUpdateAction(desktopupdate.JournalActionDownloadRequested)
	if !s.updateJournal.failed {
		t.Fatal("journal failure was not remembered")
	}
}
