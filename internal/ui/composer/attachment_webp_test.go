package composer

import (
	"os"
	"testing"
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
	// Keep real raster work outside synctest. The pure timeline checks above
	// retain exact delays/finite completion; this seat must actually paint green.
	v := readAttachmentDisplay(a, true)
	seat := attachmentDisplaySeat{open: true, full: &attachmentDisplaySeat{result: v}}
	tt := ui.NewTester(func(c *ui.Context) { paintAttachmentOriginal(c, &seat, a, a.Name) }, 600, 400)
	green := false
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); {
		time.Sleep(5 * time.Millisecond)
		tt.Frame()
		if v.bitmap != v.motion.frames[2] {
			continue
		}
		img := tt.Image()
		for y := 0; y < img.Bounds().Dy(); y++ {
			for x := 0; x < img.Bounds().Dx(); x++ {
				pixel := img.RGBAAt(x, y)
				if pixel.G > 240 && pixel.R < 20 && pixel.B < 20 {
					green = true
				}
			}
		}
		if green {
			break
		}
	}
	if !green {
		t.Fatal("original surface did not paint current WebP pixels")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	current := v.bitmap
	for i := 0; i < 3; i++ {
		tt.Frame()
	}
	if seat.open || seat.full != nil || v.bitmap != current {
		t.Fatal("dismissed original retains or advances WebP motion")
	}
}
