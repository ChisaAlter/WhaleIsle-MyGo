package composer

import (
	"errors"
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

// TestFirstErrorCarriesCopyNotTheRawText covers the attachment rail's tooltip,
// which is main copy: a staging failure explains the cause, it does not quote it.
func TestFirstErrorCarriesCopyNotTheRawText(t *testing.T) {
	zhForTest(t)
	display := &attachmentDisplay{err: errors.New(`open C:\clip.mp4: Access is denied.`)}
	got := firstError("", display)
	if !strings.Contains(got, "权限") {
		t.Fatalf("tooltip = %q", got)
	}
	if strings.Contains(got, "Access is denied") || strings.Contains(got, `C:\clip.mp4`) {
		t.Fatalf("tooltip leaked the raw text: %q", got)
	}
	if staged := firstError("请重试附件", nil); staged != "请重试附件" {
		t.Fatalf("staging error = %q", staged)
	}
}

// paintedCarrier matches the fields this package paints.
var paintedCarrier = regexp.MustCompile(`State\.Error|\.Error = |errorText|PrepareError|Tooltip\(|s\.err = |op\.Error`)

func TestComposerCarriersNeverTakeARawError(t *testing.T) {
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
			if !strings.Contains(line, ".Error()") || !paintedCarrier.MatchString(line) {
				continue
			}
			if strings.HasPrefix(strings.TrimSpace(line), "if ") {
				continue
			}

			t.Errorf("%s:%d paints a raw error: %s", name, index+1, strings.TrimSpace(line))
		}
	}
	if scanned == 0 {
		t.Fatal("scanned no source files")
	}
}
