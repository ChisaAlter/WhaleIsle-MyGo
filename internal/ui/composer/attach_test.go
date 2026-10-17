package composer

import (
	"path/filepath"
	"testing"
)

func TestAppendDroppedFiles(t *testing.T) {
	root := t.TempDir()
	abs := filepath.Join(root, "docs", "readme.md")
	draft := "hello"
	AppendDroppedFiles(&draft, root, []string{abs})
	want := "hello @docs/readme.md"
	if draft != want {
		t.Fatalf("got %q want %q", draft, want)
	}
}

func TestAppendDroppedFilesEmptyDraft(t *testing.T) {
	root := t.TempDir()
	abs := filepath.Join(root, "a.go")
	draft := ""
	AppendDroppedFiles(&draft, root, []string{abs})
	if draft != "@a.go" {
		t.Fatalf("got %q", draft)
	}
}

func TestAppendDroppedFilesSkipsOutsideRoot(t *testing.T) {
	root := t.TempDir()
	draft := "x"
	AppendDroppedFiles(&draft, root, []string{`C:\other\file.txt`})
	if draft != "x" {
		t.Fatalf("got %q", draft)
	}
}

func TestAppendDroppedFilesImageMarkdown(t *testing.T) {
	root := t.TempDir()
	abs := filepath.Join(root, "shot.png")
	draft := ""
	AppendDroppedFiles(&draft, root, []string{abs})
	if draft != "![](shot.png)" {
		t.Fatalf("got %q", draft)
	}
}
