package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func encodeTestPNG(w, h int, fill func(x, y int) color.Color) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, fill(x, y))
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func decodePNGSize(data []byte) (int, int, error) {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return 0, 0, err
	}
	b := img.Bounds()
	return b.Dx(), b.Dy(), nil
}

func TestScalePNGNoOpWhenAlreadySized(t *testing.T) {
	in := encodeTestPNG(32, 32, func(x, y int) color.Color {
		return color.RGBA{byte(x), byte(y), 0, 255}
	})
	out, err := scalePNG(in, 32)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, in) {
		t.Fatal("expected original bytes when size matches")
	}
}

func TestScalePNGDimensions(t *testing.T) {
	in := encodeTestPNG(64, 64, func(x, y int) color.Color {
		return color.RGBA{255, 0, 0, 255}
	})
	out, err := scalePNG(in, 32)
	if err != nil {
		t.Fatal(err)
	}
	w, h, err := decodePNGSize(out)
	if err != nil {
		t.Fatal(err)
	}
	if w != 32 || h != 32 {
		t.Fatalf("size: got %dx%d", w, h)
	}
}

func TestScalePNGUsesInterpolation(t *testing.T) {
	// 2×2 checkerboard → 1×1 center sample should blend, not pick a corner only.
	in := encodeTestPNG(2, 2, func(x, y int) color.Color {
		if x^y == 0 {
			return color.RGBA{0, 0, 0, 255}
		}
		return color.RGBA{255, 255, 255, 255}
	})
	out, err := scalePNG(in, 1)
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	r, g, b, _ := img.At(0, 0).RGBA()
	// Nearest-neighbor would be fully black or white (0 or 65535).
	if r == 0 && g == 0 && b == 0 {
		t.Fatal("expected blended pixel, got pure black")
	}
	if r == 65535 && g == 65535 && b == 65535 {
		t.Fatal("expected blended pixel, got pure white")
	}
}
