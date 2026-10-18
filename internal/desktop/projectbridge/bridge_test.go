package projectbridge

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// One bounded fixture, temporary directories only: no Git, Host, credentials or PRs.
func TestProjectBoundaryFixture(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node unavailable")
	}
	home, root, outside := t.TempDir(), t.TempDir(), t.TempDir()
	b := New(Options{NodeBin: node, Home: home, Workspace: root})
	if !b.Ready(context.Background()) {
		t.Fatal("embedded source closure did not load")
	}
	call := func(action, p string) map[string]any {
		return b.Dispatch(context.Background(), map[string]any{"action": action, "workingDirectory": p})
	}
	if result := call("canonicalize", root); result["ok"] != true {
		t.Fatalf("registered root: %v", result)
	}
	for _, action := range []string{"canonicalize", "select-directory", "run-command"} {
		if result := call(action, outside); result["ok"] != false {
			t.Fatalf("%s allowed outside root", action)
		}
	}
	metadata := filepath.Join(root, ".git")
	if err := os.Mkdir(metadata, 0700); err != nil {
		t.Fatal(err)
	}
	if call("canonicalize", metadata)["ok"] != false {
		t.Fatal("Git metadata allowed")
	}
	link := filepath.Join(root, "escape")
	if os.Symlink(outside, link) == nil && call("canonicalize", link)["ok"] != false {
		t.Fatal("symlink escape allowed")
	}
	if New(Options{Home: home}).Ready(context.Background()) {
		t.Fatal("missing node advertised ready")
	}
	result := b.Dispatch(context.Background(), map[string]any{"action": "open-path", "path": root})
	if result["ok"] != false {
		t.Fatal("missing opener silently succeeded")
	}
}
