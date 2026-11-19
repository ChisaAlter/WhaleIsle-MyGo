package main

import (
	"os"
	"path/filepath"
	"testing"

	"whaleisle/internal/harness"
	"whaleisle/internal/hoststate"
	"whaleisle/internal/ui/boot"
	"whaleisle/internal/ui/i18n"
)

// TestColdPageAsksForTheFolderWhenThereIsNone is the dead front door this
// change removes: a launch with automatic start and no workspace must open on
// the page with the one action that resolves it, not on a workbench whose
// service never started.
func TestColdPageAsksForTheFolderWhenThereIsNone(t *testing.T) {
	i18n.SetLocale(i18n.LocaleZH)
	defer i18n.SetLocale(i18n.DefaultLocale)
	report := hoststate.NeedWorkspace()
	got := openedColdPage(boot.Model{}, harness.Status{State: harness.Stopped}, true, true, "")
	if !got.Declined {
		t.Fatal("a declined automatic start does not stay on the page")
	}
	if got.StatusText != report.Line {
		t.Fatalf("status = %q, want %q", got.StatusText, report.Line)
	}
	if got.ActionLabel != report.Action {
		t.Fatalf("action = %q, want %q", got.ActionLabel, report.Action)
	}
}

// TestColdPageKeepsTheDeclineReason covers every other declined reason: the
// sentence the shell already carries is what the page shows.
func TestColdPageKeepsTheDeclineReason(t *testing.T) {
	reason := "上次启动未完整恢复，已暂停自动启动。请查看诊断后重试。"
	got := openedColdPage(boot.Model{}, harness.Status{State: harness.Stopped}, true, false, reason)
	if !got.Declined || got.StatusText != reason {
		t.Fatalf("declined page = %+v", got)
	}
	if got.ActionLabel != "" {
		t.Fatalf("the page overrode the retry action it did not own: %q", got.ActionLabel)
	}
}

// TestColdPageLeavesARunningStartAlone keeps the page's own model when the
// launch is not a declined start: a starting Host must keep its progress copy.
func TestColdPageLeavesARunningStartAlone(t *testing.T) {
	for _, tc := range []struct {
		name    string
		skipped bool
		state   harness.State
	}{
		{"not declined", false, harness.Stopped},
		{"declined but starting", true, harness.Starting},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := openedColdPage(boot.Model{}, harness.Status{State: tc.state}, tc.skipped, true, "")
			if got.Declined || got.StatusText != "" || got.ActionLabel != "" {
				t.Fatalf("page model was rewritten: %+v", got)
			}
		})
	}
}

// TestWorkspaceUsable separates the first start from a broken one: an empty or
// vanished path is something to ask for, not a failure to report.
func TestWorkspaceUsable(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "a-file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		path string
		want bool
	}{
		{"empty", "   ", false},
		{"missing", filepath.Join(dir, "gone"), false},
		{"a file, not a folder", file, false},
		{"folder", dir, true},
	} {
		if got := workspaceUsable(tc.path); got != tc.want {
			t.Fatalf("%s: workspaceUsable(%q) = %v, want %v", tc.name, tc.path, got, tc.want)
		}
	}
}

// TestWorkbenchStripNeverSaysTheHostIsNotRunning is the copy the operator saw:
// the strip must state the service state and the next step, and never fall back
// to a bare state name or the transport's words.
func TestWorkbenchStripNeverSaysTheHostIsNotRunning(t *testing.T) {
	i18n.SetLocale(i18n.LocaleZH)
	defer i18n.SetLocale(i18n.DefaultLocale)
	for _, st := range []harness.Status{
		{State: harness.Stopped},
		{State: harness.Starting},
		{State: harness.Failed},
		{State: harness.Failed, RuntimeFailure: true},
	} {
		line := boot.StatusLine(st)
		if line == "" || line == i18n.T(i18n.KeyBootStopped) {
			t.Fatalf("state %v paints %q", st.State, line)
		}
	}
	stopped := boot.StatusLine(harness.Status{State: harness.Stopped})
	if stopped != hoststate.Of(harness.Status{State: harness.Stopped}).Line {
		t.Fatalf("strip disagrees with the shared report: %q", stopped)
	}
}
