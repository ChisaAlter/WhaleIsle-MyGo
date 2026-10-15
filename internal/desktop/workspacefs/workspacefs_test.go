package workspacefs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListDirSortAndVisibleDotfiles(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"b.txt", "a.go", "Sub", ".git", ".hidden"} {
		p := filepath.Join(dir, name)
		if name == "Sub" {
			if err := os.Mkdir(p, 0o755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ents, err := ListDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) != 4 {
		t.Fatalf("len = %d, want 4", len(ents))
	}
	if !ents[0].IsDir || ents[0].Name != "Sub" {
		t.Fatalf("first = %#v", ents[0])
	}
	if ents[1].Name != ".hidden" || ents[2].Name != "a.go" || ents[3].Name != "b.txt" {
		t.Fatalf("files order = %#v", ents[1:])
	}
}

func TestReadFileTextOK(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hello.txt")
	body := "line1\nline2\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := ReadFileText(path, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if res.Text != body || res.Truncated || res.Size != int64(len(body)) {
		t.Fatalf("got %#v", res)
	}
}

func TestReadFileTextTooLarge(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "big.txt")
	if err := os.WriteFile(path, []byte(strings.Repeat("a", 100)), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ReadFileText(path, 50)
	if err != ErrTooLarge {
		t.Fatalf("got %v", err)
	}
}

func TestReadFileTextBinary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bin.dat")
	if err := os.WriteFile(path, []byte{0x00, 0x01, 0x02}, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ReadFileText(path, 1024)
	if err != ErrBinary {
		t.Fatalf("got %v", err)
	}
}

func TestJoinRoot(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "pkg")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := JoinRoot(root, "pkg/main.go")
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(sub, "main.go") {
		t.Fatalf("got %q", got)
	}
	if _, err := JoinRoot(root, "../outside"); err == nil {
		t.Fatal("expected error for ..")
	}
}

func TestRelPath(t *testing.T) {
	root := t.TempDir()
	full := filepath.Join(root, "a", "b.txt")
	rel, err := RelPath(root, full)
	if err != nil {
		t.Fatal(err)
	}
	if rel != "a/b.txt" {
		t.Fatalf("rel = %q", rel)
	}
}

func TestWriteFileText(t *testing.T) {
	root := t.TempDir()
	full := filepath.Join(root, "n", "f.txt")
	if err := WriteFileText(full, "hello\n"); err != nil {
		t.Fatal(err)
	}
	got, err := ReadFileText(full, 0)
	if err != nil || got.Text != "hello\n" {
		t.Fatalf("got %+v err=%v", got, err)
	}
}
