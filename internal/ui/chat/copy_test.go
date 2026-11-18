package chat

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

// TestBlockFailPaintsCopyNotTheRawText covers the embedded UI block's error
// line: the transcript names the cause and the next step.
func TestBlockFailPaintsCopyNotTheRawText(t *testing.T) {
	zhForTest(t)
	block := &openUIBlock{}
	raw := "dial tcp 10.0.0.1:443: connect: connection refused"
	if !block.fail(errors.New(raw)) {
		t.Fatal("first failure did not change the line")
	}
	if !strings.Contains(block.err, "网络") {
		t.Fatalf("block error = %q", block.err)
	}
	if strings.Contains(block.err, "dial tcp") {
		t.Fatalf("block error leaked the raw text: %q", block.err)
	}
	if block.fail(errors.New(raw)) {
		t.Fatal("identical failure repainted the line")
	}
	if !block.fail(errors.New("open /x: no such file or directory")) {
		t.Fatal("different cause did not change the line")
	}
}

// paintedCarrier matches the fields this package paints. The openui response
// bodies carry the raw error back to the embedded client and are not copy.
var paintedCarrier = regexp.MustCompile(`\.err = |\.failed = |state\.error|p\.Err|previewError|state\.err|paintImageFailure|Tooltip\(`)

func TestChatCarriersNeverTakeARawError(t *testing.T) {
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
			if strings.HasPrefix(strings.TrimSpace(line), "if ") || strings.Contains(line, "body[") || strings.Contains(line, "body.") {
				continue
			}
			t.Errorf("%s:%d paints a raw error: %s", name, index+1, strings.TrimSpace(line))
		}
	}
	if scanned == 0 {
		t.Fatal("scanned no source files")
	}
}
