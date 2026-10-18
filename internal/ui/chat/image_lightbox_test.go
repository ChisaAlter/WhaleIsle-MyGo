package chat

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestDisplayImagePathLocalPNG(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "shot.png")
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.Rect(0, 0, 1, 1)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := DisplayImagePath(p); got != p {
		t.Fatalf("DisplayImagePath = %q", got)
	}
	if loadChatBitmap(p) == nil {
		t.Fatal("expected decodable bitmap for lightbox")
	}
}
