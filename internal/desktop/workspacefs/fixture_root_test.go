package workspacefs

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// gitIsolatedRoot creates a workspace fixture that git treats as a standalone
// directory rather than a subfolder of an enclosing repository.
//
// scripts/env.ps1 redirects GOTMPDIR into <repo>/.toolchain/gotmp, which is
// git-ignored; t.TempDir() therefore lands inside this repository's worktree.
// `git check-ignore --no-index` then reports every fixture file as ignored
// (because the directory itself is ignored), so plain listing and search come
// back empty. GIT_CEILING_DIRECTORIES stops repository discovery above the
// fixture, which is exactly the standalone-directory state these tests model.
func gitIsolatedRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(root))
	return root
}

// TestSearchFixtureIsGitIsolated pins the fixture contract: files in the helper
// root must be visible even when the enclosing tree has a repository that
// ignores them, and git must not see that enclosing repository from the root.
func TestSearchFixtureIsGitIsolated(t *testing.T) {
	root := gitIsolatedRoot(t)
	if err := os.WriteFile(filepath.Join(root, "keep.txt"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "src", "client"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "client", "FilePreview.tsx"), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	paths, err := SearchRelPaths(context.Background(), root, "", nil)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	for _, want := range []string{"keep.txt", "src/client/FilePreview.tsx"} {
		if !slices.Contains(paths, want) {
			t.Fatalf("fixture file %q hidden: %v", want, paths)
		}
	}

	out, err := exec.Command("git", "-C", root, "rev-parse", "--show-toplevel").CombinedOutput()
	if err == nil && strings.TrimSpace(string(out)) != "" {
		t.Fatalf("fixture root still belongs to repository %q", strings.TrimSpace(string(out)))
	}
}
