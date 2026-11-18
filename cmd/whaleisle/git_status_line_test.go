package main

import (
	"errors"
	"strings"
	"testing"

	"whaleisle/internal/ui/i18n"
)

func zhForTest(t *testing.T) {
	t.Helper()
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(previous) })
}

// TestGitStatusLineExplainsAFailureWithoutGitWording pins the strip's contract:
// the line names the cause and the next step, while git's own words (stderr and
// the exit status) only reach the log.
func TestGitStatusLineExplainsAFailureWithoutGitWording(t *testing.T) {
	zhForTest(t)
	out := "fatal: not a git repository (or any of the parent directories): .git"
	got := gitStatusLine("Git fetch OK", "Git 失败", out, errors.New("exit status 128"))
	if !strings.HasPrefix(got, "Git 失败：") {
		t.Fatalf("strip does not name the failed action: %q", got)
	}
	if !strings.Contains(got, "运行时进程已退出") {
		t.Fatalf("strip does not explain the cause: %q", got)
	}
	for _, raw := range []string{"fatal:", "exit status", "parent directories"} {
		if strings.Contains(got, raw) {
			t.Fatalf("strip leaked %q: %q", raw, got)
		}
	}
}

// TestGitStatusLineKeepsTheSuccessCopy leaves the successful paths alone: the
// strip still reports what git said when it worked.
func TestGitStatusLineKeepsTheSuccessCopy(t *testing.T) {
	if got := gitStatusLine("Git fetch OK", "Git 失败", "Already up to date.\nsecond line", nil); got != "Already up to date." {
		t.Fatalf("success line = %q", got)
	}
	if got := gitStatusLine("Git fetch OK", "Git 失败", "", nil); got != "Git fetch OK" {
		t.Fatalf("fallback ok line = %q", got)
	}
}
