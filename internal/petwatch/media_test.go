package petwatch

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// Synthetic raster-only fixture; no Host, GUI, account, log or network access.
func TestStickerMediaBoundary(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, "data", "whale", "stickers")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "ok.png")
	f, err := os.Create(target)
	if err != nil {
		t.Fatal(err)
	}
	pic := image.NewRGBA(image.Rect(0, 0, 2, 2))
	pic.Set(0, 0, color.RGBA{255, 0, 0, 255})
	if err = png.Encode(f, pic); err != nil {
		t.Fatal(err)
	}
	f.Close()
	raw, mime, err := ReadSticker(home, "", target)
	if err != nil || len(raw) == 0 || mime != "image/png" {
		t.Fatalf("valid sticker: %s %v", mime, err)
	}
	outside := filepath.Join(home, "outside.png")
	if err = os.WriteFile(outside, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err = ReadSticker(home, "", outside); err == nil {
		t.Fatal("outside sticker accepted")
	}
	fake := filepath.Join(root, "fake.png")
	os.WriteFile(fake, []byte("<html><script>alert(1)</script></html>"), 0600)
	if _, _, err = ReadSticker(home, "", fake); err == nil {
		t.Fatal("non-raster accepted")
	}
	linked := filepath.Join(root, "linked.png")
	if err = os.Symlink(outside, linked); err == nil {
		if _, _, err = ReadSticker(home, "", linked); err == nil {
			t.Fatal("symlink accepted")
		}
	}
	large := filepath.Join(root, "large.png")
	os.WriteFile(large, make([]byte, 2*1024*1024+1), 0600)
	if _, _, err = ReadSticker(home, "", large); err == nil {
		t.Fatal("oversized sticker accepted")
	}
	if w, h := webpDimensions([]byte("<svg/>")); w != 0 || h != 0 {
		t.Fatal("invalid WebP accepted")
	}
}
