package harness

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestB29BootDiagnosticEnvelope(t *testing.T) {
	st := Status{State: Failed, Err: errors.New("plugin failed"), Attempt: 2, Recovery: true, RecoveryCancelled: true, Source: "bundled", PluginLoad: PluginProgress{Phase: "runtime", Ready: 1, Total: 3}}
	dump := BootLogDump("1.2.3", time.Date(2026, 10, 16, 13, 0, 0, 0, time.UTC), st, []string{"line one", "line two"})
	for _, value := range []string{"Whale Isle 1.2.3", "savedAt: 2026-10-16T13:00:00Z", "state: failed", "error: plugin failed", "failure.phase: runtime", "recovery.attempt: 2", "recovery.status: cancelled", "pluginRecovery.skipUserPlugins: true", "--- logs ---\nline one\nline two\n"} {
		if !strings.Contains(dump, value) {
			t.Errorf("missing %q in %s", value, dump)
		}
	}
}

func TestB29SaveCancellationAndSelection(t *testing.T) {
	dir := t.TempDir()
	path, err := exportLogsWithSave([]string{"data"}, func() (string, error) { return "", nil })
	if err != nil || path != "" {
		t.Fatalf("cancel: %q %v", path, err)
	}
	files, _ := os.ReadDir(dir)
	if len(files) != 0 {
		t.Fatal("cancel wrote a file")
	}
	selected := filepath.Join(dir, "chosen.log")
	path, err = exportLogsWithSave([]string{"diagnostic\n"}, func() (string, error) { return selected, nil })
	if err != nil || path != selected {
		t.Fatalf("save: %q %v", path, err)
	}
	raw, err := os.ReadFile(selected)
	if err != nil || string(raw) != "diagnostic\n" {
		t.Fatalf("contents: %q %v", raw, err)
	}
	failure := errors.New("dialog unavailable")
	_, err = exportLogsWithSave(nil, func() (string, error) { return "", failure })
	if !errors.Is(err, failure) {
		t.Fatal(err)
	}
}
