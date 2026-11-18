package projectbridge

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
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

func TestConcurrentNonGitWorkspacePreparationRetainsOneReceipt(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node unavailable")
	}
	root, home := t.TempDir(), t.TempDir()
	b := New(Options{NodeBin: node, Home: home, Workspace: root})
	payload := map[string]any{"action": "prepare-workspace", "projectId": "project-concurrent", "workstreamId": "work-concurrent", "workingDirectory": root, "mode": "existing"}
	results := make(chan map[string]any, 4)
	var workers sync.WaitGroup
	for range 4 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			results <- b.Dispatch(context.Background(), payload)
		}()
	}
	workers.Wait()
	close(results)
	var receipt string
	for result := range results {
		if result["ok"] != true {
			t.Fatalf("workspace preparation: %v", result)
		}
		current, _ := result["receipt"].(string)
		if current == "" || receipt != "" && receipt != current {
			t.Fatalf("concurrent preparation changed receipt: first=%q current=%q", receipt, current)
		}
		receipt = current
	}
	b.workspaceMu.Lock()
	defer b.workspaceMu.Unlock()
	if len(b.workspaces) != 0 {
		t.Fatal("finished workspace binding retained its operation owner")
	}
}

func TestReadyMemoizesRepeatedNodeProbes(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node unavailable")
	}
	probes := 0
	prev := onReadyNodeProbe
	onReadyNodeProbe = func() { probes++ }
	t.Cleanup(func() { onReadyNodeProbe = prev })

	b := New(Options{NodeBin: node, Home: t.TempDir(), Workspace: t.TempDir()})
	for i := 0; i < 4; i++ {
		if !b.Ready(context.Background()) {
			t.Fatalf("Ready() returned false on call %d", i+1)
		}
	}
	if probes != 1 {
		t.Fatalf("expected 1 Node probe for repeated Ready checks, got %d", probes)
	}
}
