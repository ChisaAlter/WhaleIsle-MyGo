package harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"whaleisle/internal/paths"
)

func TestExportLogsDefaultPath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvUserData, dir)
	path, err := ExportLogs([]string{"line1", "line2"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(path, filepath.Join(dir, "logs")) {
		t.Fatalf("path %s", path)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "line1\nline2" {
		t.Fatalf("content %q", raw)
	}
}
