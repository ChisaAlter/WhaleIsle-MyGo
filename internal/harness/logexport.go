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

// ExportLogsDialog prompts for a save location. Cancellation never writes a file.
func ExportLogsDialog(parent *mygo.Window, lines []string, title string) (string, error) {
	defaultDir := filepath.Join(paths.UserDataDir(), "logs")
	defaultPath := filepath.Join(defaultDir, fmt.Sprintf("harness-%s.log", time.Now().Format("20060102-150405")))
	return exportLogsWithSave(lines, func() (string, error) {
		return mygo.Dialog.Save(mygo.SaveDialogOptions{
			Parent:            parent,
			Title:             title,
			DefaultPath:       defaultPath,
			CreateDirectories: true,
		})
	})
}

func exportLogsWithSave(lines []string, save func() (string, error)) (string, error) {
	path, err := save()
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", nil
	}
	return ExportLogs(lines, path)
}

// BootLogDump formats the user-visible diagnostic snapshot, excluding endpoint
// authentication data. Unknown failure details remain explicitly unavailable.
func BootLogDump(version string, savedAt time.Time, status Status, logs []string) string {
	value := func(s string) string {
		if s == "" {
			return "-"
		}
		return s
	}
	errText := ""
	if status.Err != nil {
		errText = status.Err.Error()
	}
	recovery := "idle"
	if status.RecoveryActive {
		recovery = "active"
	}
	if status.RecoveryCancelled {
		recovery = "cancelled"
	}
	lines := []string{
		strings.TrimSpace("Whale Isle " + version),
		"savedAt: " + savedAt.Format(time.RFC3339Nano),
		"state: " + status.State.String(),
		"error: " + value(errText),
		"failure.phase: " + value(status.PluginLoad.Phase),
		"failure.message: " + value(errText),
		"failure.code: -", "failure.signal: -", "failure.occurredAt: -",
		"source: " + value(status.Source),
		fmt.Sprintf("generation: %d", status.Generation),
		"recovery.status: " + recovery,
		fmt.Sprintf("recovery.attempt: %d", status.Attempt),
		fmt.Sprintf("pluginRecovery.skipUserPlugins: %t", status.Recovery),
		fmt.Sprintf("pluginLoad.progress: %d/%d", status.PluginLoad.Ready, status.PluginLoad.Total),
		"pluginLoad.detail: " + value(status.PluginLoad.Detail),
		"", "--- logs ---",
	}
	return strings.Join(append(lines, logs...), "\n") + "\n"
}

// ExportBootLogsDialog captures version, time and lifecycle state with the log.
func ExportBootLogsDialog(parent *mygo.Window, version string, status Status, logs []string, title string) (string, error) {
	return ExportLogsDialog(parent, []string{BootLogDump(version, time.Now(), status, logs)}, title)
}
