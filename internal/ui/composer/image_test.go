package composer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInsertImageMention(t *testing.T) {
	root := t.TempDir()
	img := filepath.Join(root, "shot.png")
	if err := os.WriteFile(img, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	draft := "hello"
	if err := InsertImageMention(&draft, root, img); err != nil {
		t.Fatal(err)
	}
	if draft != "hello ![](shot.png) " {
		t.Fatalf("%q", draft)
	}
}
