package frame

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestWorkspaceContentDiffLoadsAsynchronouslyWithoutBlockingFrame(t *testing.T) {
	root := filepath.Clean(t.TempDir())
	if _, err := exec.LookPath("git"); err == nil {
		cmd := exec.Command("git", "init")
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git init failed: %v (%s)", err, out)
		}
		target := filepath.Join(root, "hello.txt")
		if err := os.WriteFile(target, []byte("hello\n"), 0600); err != nil {
			t.Fatal(err)
		}
		add := exec.Command("git", "add", "hello.txt")
		add.Dir = root
		if out, err := add.CombinedOutput(); err != nil {
			t.Fatalf("git add failed: %v (%s)", err, out)
		}
		commit := exec.Command("git", "-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "-m", "init")
		commit.Dir = root
		if out, err := commit.CombinedOutput(); err != nil {
			t.Fatalf("git commit failed: %v (%s)", err, out)
		}
		if err := os.WriteFile(target, []byte("hello world\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}

	w := &WorkspacePanes{}
	w.syncRoute("diff")
	diffPane := w.Root
	diffPane.Diff = &WorkspaceDiffState{Root: root, Path: "hello.txt"}
	t.Cleanup(func() { diffPane.Diff.Workbench.ClosePane() })

	sizes := DefaultLayout()
	sizes.Workspace = w
	var invalidated atomic.Int32
	done := make(chan struct{}, 1)
	m := Model{
		Sizes:         &sizes,
		WorkspaceRoot: root,
		invalidate: func() {
			invalidated.Add(1)
			select {
			case done <- struct{}{}:
			default:
			}
		},
	}

	local := workspaceContentModel(m, diffPane)
	// First frame must return immediately without synchronously populating diff output.
	if *local.DiffText != "" || *local.DiffErr != "" {
		t.Fatalf("expected non-blocking frame 1 before async diff completion, got text=%q err=%q", *local.DiffText, *local.DiffErr)
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("async diff loader did not invoke invalidate")
	}

	local = workspaceContentModel(m, diffPane)
	if *local.DiffText == "" && *local.DiffErr == "" {
		t.Fatal("expected async diff result (text or error) after invalidate")
	}

	// Verify ui.NewTester wiring sets m.invalidate cleanly before workspaceContentModel.
	m2 := Model{Sizes: &sizes, WorkspaceRoot: root, RightPaneOpen: true}
	var cSeen ui.Element
	_ = cSeen
	tt := ui.NewTester(func(c *ui.Context) {
		workspaceNode(c, m2, diffPane)
	}, 600, 400)
	tt.Frame()
}
