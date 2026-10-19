package files

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAVIFSequenceDoesNotExpandFilesEntry(t *testing.T) {
	data, err := os.ReadFile("../../desktop/imagepreview/testdata/test.avifs")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	for _, name := range []string{"sequence.avif", "sequence.gif"} {
		if err := os.WriteFile(filepath.Join(root, name), data, 0600); err != nil {
			t.Fatal(err)
		}
		preview := loadImagePreview(root, name)
		if preview.err == nil || preview.bitmap != nil || preview.animation != nil {
			t.Fatalf("Files gained sequence decoding via %s: %+v", name, preview)
		}
	}
	if isPreviewImage("sequence.avifs") {
		t.Fatal("new AVIS Files suffix was registered")
	}
}
