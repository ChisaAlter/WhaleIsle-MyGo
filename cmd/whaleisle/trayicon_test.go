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

// TestTrayIconResizeTarget pins the native tray raster: tray.js:20 resizes to
// 24x24, and tray.js:19 only does it on win32.
func TestTrayIconResizeTarget(t *testing.T) {
	src := encodeTestPNG(96, 96, func(x, y int) color.Color {
		return color.RGBA{byte(x % 4 * 60), byte(y % 4 * 60), 30, 255}
	})
	original := append([]byte(nil), src...)

	tray, err := resizeTrayIcon("windows", src)
	if err != nil {
		t.Fatal(err)
	}
	w, h, err := decodePNGSize(tray)
	if err != nil {
		t.Fatal(err)
	}
	if w != trayIconSize || h != trayIconSize || trayIconSize != 24 {
		t.Fatalf("windows tray icon = %dx%d, want %dx%d (native 24)", w, h, trayIconSize, trayIconSize)
	}
	if !bytes.Equal(src, original) {
		t.Fatal("resizeTrayIcon modified its source bytes")
	}
	for _, platform := range []string{"darwin", "linux"} {
		kept, err := resizeTrayIcon(platform, src)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(kept, original) {
			t.Fatalf("%s tray icon changed; the native shell only resizes on win32", platform)
		}
	}
	// An already-24 source is passed through rather than re-encoded.
	if again, err := resizeTrayIcon("windows", tray); err != nil || !bytes.Equal(again, tray) {
		t.Fatalf("24x24 tray icon was re-encoded: %v", err)
	}
}

// TestWindowIconResizeTarget pins the native window raster: window.js:185 uses
// 48x48 on win32, and other platforms draw the application icon themselves.
func TestWindowIconResizeTarget(t *testing.T) {
	src := encodeTestPNG(96, 96, func(x, y int) color.Color {
		switch {
		case x < 48 && y < 48:
			return color.RGBA{220, 30, 30, 255}
		case x >= 48 && y < 48:
			return color.RGBA{30, 200, 60, 255}
		case x < 48:
			return color.RGBA{30, 60, 220, 255}
		default:
			return color.RGBA{240, 240, 40, 255}
		}
	})
	original := append([]byte(nil), src...)
	icon, err := windowIconPNG("windows", src)
	if err != nil {
		t.Fatal(err)
	}
	w, h, err := decodePNGSize(icon)
	if err != nil {
		t.Fatal(err)
	}
	if w != windowIconSize || h != windowIconSize || windowIconSize != 48 {
		t.Fatalf("windows window icon = %dx%d, want %dx%d (native 48)", w, h, windowIconSize, windowIconSize)
	}
	if !bytes.Equal(src, original) {
		t.Fatal("windowIconPNG modified its source bytes")
	}
	for _, platform := range []string{"darwin", "linux"} {
		none, err := windowIconPNG(platform, src)
		if err != nil || none != nil {
			t.Fatalf("%s window icon = %d bytes, want none (the toolkit keeps the app icon)", platform, len(none))
		}
	}
	// Catmull-Rom keeps the four quadrants distinct rather than averaging them
	// into one color.
	decoded, err := png.Decode(bytes.NewReader(icon))
	if err != nil {
		t.Fatal(err)
	}
	colors := map[color.RGBA]bool{}
	for _, p := range []image.Point{{X: 8, Y: 8}, {X: 39, Y: 8}, {X: 8, Y: 39}, {X: 39, Y: 39}} {
		colors[color.RGBAModel.Convert(decoded.At(p.X, p.Y)).(color.RGBA)] = true
	}
	if len(colors) != 4 {
		t.Fatalf("resized icon holds %d quadrant colors, want 4", len(colors))
	}
}

// TestScalePNGRejectsBrokenInput keeps a corrupt asset from panicking the shell.
func TestScalePNGRejectsBrokenInput(t *testing.T) {
	if _, err := scalePNG([]byte("not a png"), trayIconSize); err == nil {
		t.Fatal("scalePNG accepted non-PNG bytes")
	}
	if _, err := resizeTrayIcon("windows", []byte("not a png")); err == nil {
		t.Fatal("resizeTrayIcon accepted non-PNG bytes")
	}
	if _, err := windowIconPNG("windows", []byte("not a png")); err == nil {
		t.Fatal("windowIconPNG accepted non-PNG bytes")
	}
}
