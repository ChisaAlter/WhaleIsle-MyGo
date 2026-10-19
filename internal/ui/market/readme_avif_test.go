package market

import (
	"bytes"
	"image"
	"image/draw"
	"math"
	"os"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestMarketReadmeAVIFSequenceNativePaintAndHide(t *testing.T) {
	data, err := os.ReadFile("../../desktop/imagepreview/testdata/test.avifs")
	if err != nil {
		t.Fatal(err)
	}
	cache := readmeImageCache(t)
	const source = "https://example.invalid/motion.avif"
	if err := os.WriteFile(cache.FilePath(source), data, 0600); err != nil {
		t.Fatal(err)
	}
	decoded := decodeReadmeMedia(data, true, false, false)
	if decoded.animation == nil || len(decoded.frames) != 17 {
		t.Fatal("README AVIF animation missing")
	}
	start := time.Unix(100, 0)
	decoded.advance(start)
	if wait := decoded.advance(start.Add(90 * time.Millisecond)); wait != 70*time.Millisecond || decoded.bitmap != decoded.frames[1] {
		t.Fatal("README AVIF timing", wait)
	}
	// Headless raster work stays outside synctest. Wait for the real mounted
	// decoder and observe changing pixels only inside its native image seat.
	visible := true
	tt := ui.NewTester(func(c *ui.Context) {
		if visible {
			marketReadme(c, Model{}, "![AVIF animation]("+source+")", c.Theme().Text, c.Theme().TextMuted, c.Theme().Surface, c.Theme().Accent)
		}
	}, 300, 300)
	var r ui.Rect
	ready := false
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); {
		tt.Frame()
		var ok bool
		r, ok = tt.Find("README image " + source)
		if ok && r.W > 0 && r.H > 0 && !tt.HasText("AVIF animation") {
			ready = true
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !ready || math.Abs(float64(r.W)-300) > .001 || math.Abs(float64(r.H)-216) > .001 {
		t.Fatalf("AVIF README missing or wrong native geometry: %+v", r)
	}
	pixels := func() []byte {
		picture := image.NewRGBA(image.Rect(0, 0, int(r.W), int(r.H)))
		draw.Draw(picture, picture.Bounds(), tt.Image(), image.Pt(int(r.X), int(r.Y)), draw.Src)
		return picture.Pix
	}
	before, changed := pixels(), false
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); {
		time.Sleep(5 * time.Millisecond)
		tt.Frame()
		if !bytes.Equal(before, pixels()) {
			changed = true
			break
		}
	}
	if !changed {
		t.Fatal("README AVIF remains first frame")
	}
	visible = false
	tt.Frame()
	if _, ok := tt.Find("README image " + source); ok {
		t.Fatal("hidden README retained AVIF image seat")
	}
	time.Sleep(time.Second)
	for i := 0; i < 3; i++ {
		tt.Frame()
		if _, ok := tt.Find("README image " + source); ok {
			t.Fatal("hidden README recreated AVIF image seat")
		}
	}
}
