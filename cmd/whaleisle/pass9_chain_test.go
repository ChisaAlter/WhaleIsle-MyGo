package main

import (
	"os"
	"path/filepath"
	"testing"

	"whaleisle/internal/ui/frame"
)

// TestTerminalOpenedFileKeepsEditorSaveWired is the B1/B2 code acceptance for the
// terminal link path: a file opened from a terminal's local workspace keeps the
// same editor save chain as the tree (real disk write through JoinRoot, and the
// workspace boundary still forbids escaping the root).
func TestTerminalOpenedFileKeepsEditorSaveWired(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("v1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	model := terminalFilesModel(frame.Model{}, root, "session-1")
	if model.WorkspaceRoot != root || model.SessionID != "session-1" {
		t.Fatalf("terminal model root=%q session=%q", model.WorkspaceRoot, model.SessionID)
	}
	if model.OnSaveEdit == nil {
		t.Fatal("a file opened from the terminal lost its editor writer")
	}
	if err := model.OnSaveEdit("notes.txt", "v2\n"); err != nil {
		t.Fatalf("save: %v", err)
	}
	disk, err := os.ReadFile(filepath.Join(root, "notes.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(disk) != "v2\n" {
		t.Fatalf("disk=%q, want the saved draft", disk)
	}
	if err := model.OnSaveEdit("../escape.txt", "x"); err == nil {
		t.Fatal("the writer accepted a path outside the workspace root")
	}
}
