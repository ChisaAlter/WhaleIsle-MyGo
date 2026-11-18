package files

import (
	"errors"
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestSetFileErrorCarriesCopyNotTheRawText covers the preview status line: a
// failed action names the cause and the next step.
func TestSetFileErrorCarriesCopyNotTheRawText(t *testing.T) {
	zhForTest(t)
	status := ""
	setFileError(Model{EditStatus: &status}, errors.New(`open C:\notes.md: Access is denied.`))
	if !strings.Contains(status, "权限") {
		t.Fatalf("preview status = %q", status)
	}
	if strings.Contains(status, "Access is denied") || strings.Contains(status, `C:\notes.md`) {
		t.Fatalf("preview status leaked the raw text: %q", status)
	}
}

// paintedCarrier matches the fields this package paints, plus the preview
// message returns that the pane paints as its own text.
var paintedCarrier = regexp.MustCompile(`tab\.Error|tab\.Status|\.Problem|EditStatus|setFileStatus\(|previewError|hint = |ui\.Text\(c,`)

func TestFilesCarriersNeverTakeARawError(t *testing.T) {
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
