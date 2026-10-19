package composer

import (
	"os"
	"testing"
	"testing/synctest"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestAttachmentNativeWebPThumbnailAndOriginal(t *testing.T) {
	data, err := os.ReadFile("../../desktop/imagepreview/testdata/animated-native.webp")
	if err != nil {
		t.Fatal(err)
	}
	a := attachmentFixture("moving.webp", data)
	for _, full := range []bool{false, true} {
		v := readAttachmentDisplay(a, full)
		if v.err != nil || v.motion == nil || len(v.motion.frames) != 4 || len(v.motion.animation.Frames) != 0 {
			t.Fatalf("full%v: %+v", full, v)
		}
		start := time.Unix(100, 0)
		v.advance(start)
		if wait := v.advance(start.Add(110 * time.Millisecond)); wait != 90*time.Millisecond || v.bitmap != v.motion.frames[2] {
			t.Fatal("WebP did not advance", wait)
		}
		if wait := v.advance(start.Add(time.Second)); wait != 0 || v.bitmap != v.motion.frames[3] {
			t.Fatal("finite WebP did not stop")
		}
	}
	// Global raster helpers must belong to the process, not the timed bubble.
	ui.NewTester(func(c *ui.Context) { ui.Box(c).Fill().Background(ui.RGB(255, 255, 255)) }, 600, 400)
	synctest.Test(t, func(t *testing.T) {
		v := readAttachmentDisplay(a, true)
		seat := attachmentDisplaySeat{open: true, full: &attachmentDisplaySeat{result: v}}
		tt := ui.NewTester(func(c *ui.Context) { paintAttachmentOriginal(c, &seat, a, a.Name) }, 600, 400)
		time.Sleep(110 * time.Millisecond)
		tt.Frame()
		if v.bitmap != v.motion.frames[2] {
			t.Fatal("original surface did not advance")
		}
		green := false
		img := tt.Image()
		for y := 0; y < img.Bounds().Dy(); y++ {
			for x := 0; x < img.Bounds().Dx(); x++ {
				pixel := img.RGBAAt(x, y)
				if pixel.G > 240 && pixel.R < 20 && pixel.B < 20 {
					green = true
				}
			}
		}
		if !green {
			t.Fatal("original surface did not paint current WebP pixels")
		}
		tt.Key(0, ui.KeyEscape)
		tt.Frame()
		if seat.open || seat.full != nil {
			t.Fatal("dismissed original retains WebP motion")
		}
	})
}
