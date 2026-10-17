package harness

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"

	"whaleisle/internal/paths"
)

// ExportLogs writes supervisor log lines to path. When path is empty, writes
// under userData/logs with a timestamped file name.
func ExportLogs(lines []string, path string) (written string, err error) {
	text := strings.Join(lines, "\n")
	if path == "" {
		dir := filepath.Join(paths.UserDataDir(), "logs")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
		path = filepath.Join(dir, fmt.Sprintf("harness-%s.log", time.Now().Format("20060102-150405")))
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// ExportLogsDialog prompts for a save location, falling back to userData/logs.
func ExportLogsDialog(parent *mygo.Window, lines []string, title string) (string, error) {
	defaultDir := filepath.Join(paths.UserDataDir(), "logs")
	_ = os.MkdirAll(defaultDir, 0o755)
	defaultPath := filepath.Join(defaultDir, fmt.Sprintf("harness-%s.log", time.Now().Format("20060102-150405")))
	path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
		Parent:            parent,
		Title:             title,
		DefaultPath:       defaultPath,
		CreateDirectories: true,
	})
	if err != nil {
		return "", err
	}
	if path == "" {
		return ExportLogs(lines, "")
	}
	return ExportLogs(lines, path)
}
