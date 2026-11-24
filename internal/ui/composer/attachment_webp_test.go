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
		v := readSettled(a, full)
		if v.failure() != nil || !v.handle.Animated() {
			t.Fatalf("full%v: %+v", full, v)
		}
		start := time.Unix(100, 0)
		v.advance(start)
		if wait := v.advance(start.Add(110 * time.Millisecond)); wait != 90*time.Millisecond {
			t.Fatal("WebP did not advance", wait)
		}
		if wait := v.advance(start.Add(time.Second)); wait != 0 {
			t.Fatal("finite WebP did not stop")
		}
	}
	synctest.Test(t, func(t *testing.T) {
		v := readSettled(a, true)
		seat := attachmentDisplaySeat{open: true, full: &attachmentDisplaySeat{result: v}}
		tt := ui.NewTester(func(c *ui.Context) { paintAttachmentOriginal(c, &seat, a, a.Name, nil) }, 600, 400)
		tt.SetTime(time.Now().Add(300 * time.Millisecond))
		time.Sleep(110 * time.Millisecond)
		tt.Frame()
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
		current := v.bitmap
		for i := 0; i < 3; i++ {
			tt.Frame()
		}
		if seat.open || seat.full != nil || v.bitmap != current {
			t.Fatal("dismissed original retains or advances WebP motion")
		}
	})
}
