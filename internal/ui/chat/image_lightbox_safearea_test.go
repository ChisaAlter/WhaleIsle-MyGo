package chat

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
)

func TestImageLightboxCloseButtonReservesTitleBarRightSafeArea(t *testing.T) {
	dir := t.TempDir()
	imgPath := filepath.Join(dir, "sample.png")
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 60, G: 120, B: 220, A: 255})
		}
	}
	f, err := os.Create(imgPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()

	// Warm the async bitmap cache so the lightbox renders synchronously.
	done := make(chan struct{}, 1)
	bmp, _ := cachedChatBitmap(imgPath, func() {
		select {
		case done <- struct{}{}:
		default:
		}
	})
	if bmp == nil {
		<-done
	}

	const winW, winH = 800, 600
	const rightInset float32 = 104
	closed := false
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Fill().Children(func() {
			ImageLightboxOverlay(c, imgPath, "deepseek.light", false, func() { closed = true })
		})
	}, winW, winH)
	tt.SetTitleBar(ui.TitleBar{Height: 12, Right: rightInset})
	tt.Frame()

	closeRect, ok := tt.Find(i18n.T(i18n.KeyToolImageClosePreview))
	if !ok {
		t.Fatal("lightbox close button not found")
	}
	maxAllowedRight := float32(winW) - rightInset
	if closeRect.X+closeRect.W > maxAllowedRight {
		t.Fatalf("lightbox close button %+v intrudes into Windows caption safe area (right <= %v)", closeRect, maxAllowedRight)
	}
	if err := tt.Click(i18n.T(i18n.KeyToolImageClosePreview)); err != nil {
		t.Fatalf("clicking lightbox close button failed: %v", err)
	}
	if !closed {
		t.Fatal("expected lightbox close callback to fire")
	}
}
