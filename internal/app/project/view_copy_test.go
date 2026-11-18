package projectctl

import (
	"os"
	"regexp"
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

// TestViewErrorPaintsCopyNotTheRawText pins the boundary the whole flow uses:
// the view model carries the cause and the next step, never the stored text.
func TestViewErrorPaintsCopyNotTheRawText(t *testing.T) {
	zhForTest(t)
	tests := []struct {
		raw  string
		want string
	}{
		{`open C:\notes.md: Access is denied.`, "权限"},
		{"dial tcp 10.0.0.1:443: connect: connection refused", "网络"},
		{"open /work/a.md: no such file or directory", "找不到"},
		{"weird failure", "操作没有完成"},
	}
	for _, tc := range tests {
		got := viewError(tc.raw)
		if !strings.Contains(got, tc.want) {
			t.Errorf("viewError(%q) = %q, want it to mention %q", tc.raw, got, tc.want)
		}
		if strings.Contains(got, tc.raw) {
			t.Errorf("viewError(%q) leaked the stored text", tc.raw)
		}
	}
	if got := viewError("  "); got != "" {
		t.Fatalf("blank stored text = %q", got)
	}
}

// viewCarrier matches the fields the projects package paints. The flow keeps
// its own raw text for logs and for classifiers, so only these carriers must
// never be filled from an error's message.
var viewCarrier = regexp.MustCompile(`Error:|RequestRevertError|refreshNote|SetLastAction\(|InteractiveError|snapshot\.Error|next\.Error`)

func TestViewCarriersNeverTakeARawError(t *testing.T) {
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
			if strings.Contains(line, ".Error()") && viewCarrier.MatchString(line) {
				t.Errorf("%s:%d paints a raw error: %s", name, index+1, strings.TrimSpace(line))
			}
		}
	}
	if scanned == 0 {
		t.Fatal("scanned no source files")
	}
}
