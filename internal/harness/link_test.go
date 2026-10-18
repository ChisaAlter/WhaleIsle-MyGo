package harness

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestEnsureDirectoryLinkIdempotent(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("junction semantics are Windows-specific")
	}
	root := t.TempDir()
	src := filepath.Join(root, "src")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "package.json"), []byte(`{"name":"x"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(root, "node_modules", "x")
	created, err := EnsureDirectoryLink(src, dst)
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatal("first EnsureDirectoryLink should create the junction")
	}
	// Second call must treat the existing mklink /J junction as a match
	// (not "unknown content"). Go reports junctions as ModeIrregular.
	created, err = EnsureDirectoryLink(src, dst)
	if err != nil {
		t.Fatalf("idempotent EnsureDirectoryLink: %v", err)
	}
	if created {
		t.Fatal("matching junction should not report created=true")
	}
	if _, err := os.Stat(filepath.Join(dst, "package.json")); err != nil {
		t.Fatal(err)
	}
}

func TestEnsureDirectoryLinkRetargetsJunction(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("junction semantics are Windows-specific")
	}
	root := t.TempDir()
	srcA := filepath.Join(root, "a")
	srcB := filepath.Join(root, "b")
	for _, src := range []string{srcA, srcB} {
		if err := os.MkdirAll(src, 0o755); err != nil {
			t.Fatal(err)
		}
		name := filepath.Base(src)
		if err := os.WriteFile(filepath.Join(src, "package.json"), []byte(`{"name":"`+name+`"}`), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	dst := filepath.Join(root, "link")
	if _, err := EnsureDirectoryLink(srcA, dst); err != nil {
		t.Fatal(err)
	}
	if _, err := EnsureDirectoryLink(srcB, dst); err != nil {
		t.Fatalf("retarget: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dst, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"name":"b"}` {
		t.Fatalf("got %s", data)
	}
}

func TestEnsureDirectoryLinkRefusesRealDir(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dst := filepath.Join(root, "dst")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dst, "keep.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := EnsureDirectoryLink(src, dst)
	if err == nil {
		t.Fatal("expected refuse real directory")
	}
}
