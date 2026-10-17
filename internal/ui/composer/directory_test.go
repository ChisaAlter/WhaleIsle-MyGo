package composer

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestDirectoryMention(t *testing.T) {
	root := t.TempDir()
	abs := filepath.Join(root, "src", "pkg")
	got, err := DirectoryMention(root, abs)
	if err != nil {
		t.Fatal(err)
	}
	if got != "@src/pkg" {
		t.Fatalf("got %q want @src/pkg", got)
	}
}

func TestDirectoryMentionOutsideRoot(t *testing.T) {
	root := t.TempDir()
	_, err := DirectoryMention(root, `C:\other\dir`)
	if !errors.Is(err, ErrOutsideWorkspace) {
		t.Fatalf("err = %v", err)
	}
}

func TestInsertDirectoryMention(t *testing.T) {
	root := t.TempDir()
	abs := filepath.Join(root, "docs")
	draft := "hello"
	if err := InsertDirectoryMention(&draft, root, abs); err != nil {
		t.Fatal(err)
	}
	want := "hello @docs "
	if draft != want {
		t.Fatalf("got %q want %q", draft, want)
	}
}

func TestInsertDirectoryMentionEmptyDraft(t *testing.T) {
	root := t.TempDir()
	abs := filepath.Join(root, "a")
	draft := ""
	if err := InsertDirectoryMention(&draft, root, abs); err != nil {
		t.Fatal(err)
	}
	if draft != "@a " {
		t.Fatalf("got %q", draft)
	}
}

func TestInsertDirectoryMentionRejectsOutside(t *testing.T) {
	root := t.TempDir()
	draft := "x"
	err := InsertDirectoryMention(&draft, root, `C:\other\dir`)
	if !errors.Is(err, ErrOutsideWorkspace) {
		t.Fatalf("err = %v", err)
	}
	if draft != "x" {
		t.Fatalf("draft mutated: %q", draft)
	}
}
