package files

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNativeImageRoutesPreserveDirtySVG(t *testing.T) {
	original, draft := "<svg/>", "unsaved SVG source"
	m := Model{Preview: &original, EditDraft: &draft}
	if imageMode(m, "vector.svg") {
		t.Fatal("dirty source draft hidden")
	}
	if draft != "unsaved SVG source" {
		t.Fatal("draft changed")
	}
	draft = original
	for _, name := range []string{"a.gif", "a.ico", "a.svg", "a.avif", "a.png"} {
		if !imageMode(m, name) {
			t.Fatal("native image mode missing", name)
		}
	}
}

func TestNativeAVIFRouteDecodes(t *testing.T) {
	data, err := os.ReadFile("../../desktop/imagepreview/testdata/test10.avif")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err = os.WriteFile(filepath.Join(root, "native.avif"), data, 0600); err != nil {
		t.Fatal(err)
	}
	preview := loadImagePreview(root, "native.avif")
	if preview.err != nil {
		t.Fatal(preview.err)
	}
	if preview.bitmap == nil || preview.vector != nil {
		t.Fatal("AVIF did not use native image view")
	}
}
