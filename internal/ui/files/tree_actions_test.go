package files

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"whaleisle/internal/desktop/workspacefs"
)

func TestTreeDirectoryActions(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "nested folder"), 0700); err != nil {
		t.Fatal(err)
	}
	var revealed string
	m := Model{WorkspaceRoot: root, OnReveal: func(path string) { revealed = path }}
	values, ok := treePathCopies(m, "nested folder")
	if !ok || values.relative != "nested folder" || values.absolute != filepath.ToSlash(root)+"/nested folder" || values.reference != "[nested folder](nested%20folder)" {
		t.Fatalf("copies: %+v, %v", values, ok)
	}
	revealTreePath(m, "nested folder")
	if revealed != filepath.Join(root, "nested folder") {
		t.Fatalf("reveal: %q", revealed)
	}
	// Host paths need no corresponding local directory. Resolve them lexically
	// against the selected workspace, never against the Host process cwd.
	m.WorkspaceRoot = `C:\remote\project\`
	m.ListRemoteContext = func(context.Context, string) ([]workspacefs.Entry, error) { return nil, nil }
	values, ok = treePathCopies(m, "nested folder")
	if !ok || values.absolute != "C:/remote/project/nested folder" {
		t.Fatalf("host copies: %+v", values)
	}
	revealTreePath(m, "nested folder")
	if revealed != "C:/remote/project/nested folder" {
		t.Fatalf("host reveal: %q", revealed)
	}
}

func TestTreeDirectoryActionsRejectNonTreePaths(t *testing.T) {
	called := false
	m := Model{WorkspaceRoot: t.TempDir(), OnReveal: func(string) { called = true }}
	for _, path := range []string{"", "../other", ".git", "/absolute"} {
		if _, ok := treePathCopies(m, path); ok {
			t.Errorf("accepted %q", path)
		}
		revealTreePath(m, path)
	}
	if called {
		t.Fatal("invalid tree path reached reveal callback")
	}
}
