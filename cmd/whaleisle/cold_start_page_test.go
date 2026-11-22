package main

import (
	"testing"

	"whaleisle/internal/harness"
	"whaleisle/internal/hoststate"
	"whaleisle/internal/ui/boot"
	"whaleisle/internal/ui/i18n"
)

// TestColdPageKeepsTheDeclineReason covers every declined reason: the sentence
// the shell already carries is what the page shows.
func TestColdPageKeepsTheDeclineReason(t *testing.T) {
	reason := "上次启动未完整恢复，已暂停自动启动。请查看诊断后重试。"
	got := openedColdPage(boot.Model{}, harness.Status{State: harness.Stopped}, true, reason)
	if !got.Declined || got.StatusText != reason {
		t.Fatalf("declined page = %+v", got)
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
			got := openedColdPage(boot.Model{}, harness.Status{State: tc.state}, tc.skipped, "")
			if got.Declined || got.StatusText != "" {
				t.Fatalf("page model was rewritten: %+v", got)
			}
		})
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
