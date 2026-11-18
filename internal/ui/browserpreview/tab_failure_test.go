package browserpreview

import (
	"errors"
	"os"
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

// TestTabFailPaintsCopyNotTheRawError pins the chrome's carrier: the lead names
// the action, the cause names the next step, and the raw error stays in the log.
func TestTabFailPaintsCopyNotTheRawError(t *testing.T) {
	zhForTest(t)
	tab := &Tab{}
	tab.fail(browserChromeLabel("音视频预览不可用", "Media preview unavailable"), errors.New(`open C:\clip.mp4: Access is denied.`))
	if !strings.HasPrefix(tab.Error, "音视频预览不可用：") {
		t.Fatalf("chrome line does not name the action: %q", tab.Error)
	}
	if !strings.Contains(tab.Error, "权限") {
		t.Fatalf("chrome line does not explain the cause: %q", tab.Error)
	}
	for _, raw := range []string{"Access is denied", `C:\clip.mp4`} {
		if strings.Contains(tab.Error, raw) {
			t.Fatalf("chrome line leaked %q: %q", raw, tab.Error)
		}
	}
}

// TestTabFailWithoutALeadStillExplainsTheCause covers the bare call sites: the
// line stays readable instead of collapsing to an empty prefix.
func TestTabFailWithoutALeadStillExplainsTheCause(t *testing.T) {
	zhForTest(t)
	tab := &Tab{}
	tab.fail("", errors.New("open /work/report.pdf: no such file or directory"))
	if !strings.Contains(tab.Error, "找不到") || strings.Contains(tab.Error, "/work/report.pdf") {
		t.Fatalf("chrome line = %q", tab.Error)
	}
}

// assignsRawError reports whether a line fills any field straight from an
// error's text. Comparisons, returns and comments are not assignments.
func assignsRawError(line string) bool {
	trimmed := strings.TrimSpace(line)
	for _, prefix := range []string{"if ", "for ", "switch ", "case ", "return ", "//"} {
		if strings.HasPrefix(trimmed, prefix) {
			return false
		}
	}
	if !strings.Contains(line, ".Error()") {
		return false
	}
	operator := strings.Index(trimmed, "=")
	if operator < 0 {
		return false
	}
	left := trimmed[:operator]
	if strings.ContainsAny(left, "!<>") {
		return false
	}
	return !strings.HasPrefix(trimmed[operator+1:], "=")
}

// TestChromeCopyNeverCarriesARawError guards the whole carrier rather than the
// fields the 2026-11-18 audit happened to find: at HEAD the same scan reports
// 27 assignments of err.Error() into chrome copy across 15 files, and a guard
// that watched only Error and ActionMessage would have missed p.status.
func TestChromeCopyNeverCarriesARawError(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}
	scanned := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		scanned++
		source, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		for index, line := range strings.Split(string(source), "\n") {
			if assignsRawError(line) {
				t.Errorf("%s:%d paints a raw error: %s", name, index+1, strings.TrimSpace(line))
			}
		}
	}
	if scanned == 0 {
		t.Fatal("scanned no source files")
	}
}
