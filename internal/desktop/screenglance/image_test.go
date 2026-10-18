package screenglance

import (
	"bytes"
	"image/jpeg"
	"testing"
)

func TestThumbnailSizeIsBounded(t *testing.T) {
	for _, size := range [][2]int{{1920, 1080}, {3840, 2160}, {1080, 1920}, {320, 240}, {32768, 1}, {1, 32768}} {
		w, h, err := thumbnailSize(size[0], size[1])
		if err != nil || w < 1 || h < 1 || w > MaxWidth || h > MaxHeight {
			t.Fatalf("%v: %dx%d %v", size, w, h, err)
		}
	}
	for _, size := range [][2]int{{0, 10}, {1, 0}, {-1, 10}, {32769, 1}, {1, 32769}} {
		if _, _, err := thumbnailSize(size[0], size[1]); err == nil {
			t.Fatalf("accepted %v", size)
		}
	}
}

func TestSyntheticFrameJPEG(t *testing.T) {
	// Synthetic pixels only: this test never calls desktop capture.
	pixels := make([]byte, 4*10*8)
	for i := 0; i < len(pixels); i += 4 {
		pixels[i], pixels[i+1], pixels[i+2] = 0, 0, 255
	}
	encoded, err := encodeBGRA(pixels, 10, 8)
	if err != nil {
		t.Fatal(err)
	}
	img, err := jpeg.Decode(bytes.NewReader(encoded))
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 10 || img.Bounds().Dy() != 8 {
		t.Fatal(img.Bounds())
	}
	r, g, b, _ := img.At(4, 4).RGBA()
	if r < 60000 || g > 2000 || b > 2000 {
		t.Fatal("BGRA conversion")
	}
	if _, err := encodeBGRA(pixels, MaxWidth+1, 8); err == nil {
		t.Fatal("accepted oversized frame")
	}
}
