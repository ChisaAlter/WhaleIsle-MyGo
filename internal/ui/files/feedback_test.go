package files

import (
	"errors"
	"strings"
	"testing"

	"whaleisle/internal/desktop/workspacefs"
	"whaleisle/internal/ui/i18n"
)

func zhForTest(t *testing.T) {
	t.Helper()
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(previous) })
}

// TestFeedbackEntryExplainsASaveFailure covers the preview toolbar's toast: the
// save queue's raw message never becomes the line a person reads.
func TestFeedbackEntryExplainsASaveFailure(t *testing.T) {
	zhForTest(t)
	text, kind := feedbackEntry("", workspacefs.SaveSnapshot{Error: `open C:\notes.md: Access is denied.`})
	if kind != "error" {
		t.Fatalf("kind = %q, want error", kind)
	}
	if !strings.Contains(text, "权限") {
		t.Fatalf("entry does not explain the cause: %q", text)
	}
	for _, raw := range []string{"Access is denied", `C:\notes.md`} {
		if strings.Contains(text, raw) {
			t.Fatalf("entry leaked %q: %q", raw, text)
		}
	}
}

func TestFeedbackEntryKeepsTheEditStatusAndHidesItWhileSaving(t *testing.T) {
	zhForTest(t)
	if text, kind := feedbackEntry("已复制路径", workspacefs.SaveSnapshot{}); text != "已复制路径" || kind != "info" {
		t.Fatalf("idle entry = %q / %q", text, kind)
	}
	if text, _ := feedbackEntry("已复制路径", workspacefs.SaveSnapshot{Running: true, Error: "boom"}); text != "" {
		t.Fatalf("saving entry = %q, want blank", text)
	}
}

// TestSetFileErrorLocalizesThePreviewStatus keeps the fourteen preview actions
// that used to pass err.Error() on the same copy rule.
func TestSetFileErrorLocalizesThePreviewStatus(t *testing.T) {
	zhForTest(t)
	status := ""
	setFileError(Model{EditStatus: &status}, errors.New("open /work/none.md: no such file or directory"))
	if !strings.Contains(status, "找不到") {
		t.Fatalf("status does not explain the cause: %q", status)
	}
	if strings.Contains(status, "/work/none.md") || strings.Contains(status, "no such file") {
		t.Fatalf("status leaked the raw error: %q", status)
	}
}
