package workspacefs

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// referenceWalk is the previous algorithm: one ListWorkspaceDir (so one gitignore
// query) per directory.
func referenceWalk(t *testing.T, root string, includeDirs bool) []string {
	t.Helper()
	var out []string
	pending := []string{""}
	for len(pending) > 0 {
		rel := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		entries, err := ListWorkspaceDir(context.Background(), root, rel)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			path := entry.Name
			if rel != "" {
				path = rel + "/" + entry.Name
			}
			if entry.IsDir {
				pending = append(pending, path)
			}
			if includeDirs || !entry.IsDir {
				out = append(out, path)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i]) < strings.ToLower(out[j]) })
	return out
}

func writeTreeFile(t *testing.T, root, rel, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLevelWalkMatchesPerDirectoryWalk(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	root := t.TempDir()
	if out, err := exec.Command("git", "-C", root, "init", "-q").CombinedOutput(); err != nil {
		t.Skipf("git init: %v %s", err, out)
	}
	writeTreeFile(t, root, ".gitignore", "build/\n*.log\nnested/secret/\n!keep.log\n")
	for _, rel := range []string{
		"main.go", "README.md", "debug.log", "keep.log",
		"src/a.go", "src/b.go", "src/deep/er/c.go", "src/deep/er/c.log",
		"build/out.bin", "build/sub/x.o",
		"nested/ok.txt", "nested/secret/token", "nested/pub/y.txt",
		".hidden/z", "docs/guide/one.md", "docs/guide/two.md",
	} {
		writeTreeFile(t, root, rel, "x")
	}
	if err := os.MkdirAll(filepath.Join(root, "empty", "inner"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, includeDirs := range []bool{true, false} {
		want := referenceWalk(t, root, includeDirs)
		got, err := walkRelPaths(context.Background(), root, "", nil, includeDirs)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Join(got, "\n") != strings.Join(want, "\n") {
			t.Fatalf("includeDirs=%v\n got: %q\nwant: %q", includeDirs, got, want)
		}
	}
	got, _ := walkRelPaths(context.Background(), root, "", nil, true)
	joined := "\n" + strings.Join(got, "\n") + "\n"
	for _, mustNot := range []string{"\nbuild\n", "\nbuild/out.bin\n", "\ndebug.log\n", "\nnested/secret\n", "\nnested/secret/token\n", "\n.git\n"} {
		if strings.Contains(joined, mustNot) {
			t.Fatalf("ignored path %q was listed:\n%s", strings.TrimSpace(mustNot), joined)
		}
	}
	for _, must := range []string{"\nkeep.log\n", "\n.hidden/z\n", "\nsrc/deep/er/c.go\n", "\nempty/inner\n", "\nnested/pub/y.txt\n"} {
		if !strings.Contains(joined, must) {
			t.Fatalf("visible path %q was dropped:\n%s", strings.TrimSpace(must), joined)
		}
	}
}

func TestLevelWalkHonoursCancellation(t *testing.T) {
	root := t.TempDir()
	writeTreeFile(t, root, "a/b/c.txt", "x")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := walkRelPaths(ctx, root, "", nil, true); err == nil {
		t.Fatal("a cancelled walk returned a result")
	}
}
