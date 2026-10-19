package composer

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"testing"
	"testing/synctest"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/imagepreview"
	"whaleisle/internal/store/session"
)

func attachmentJPEGFixture(t *testing.T, orientation uint16) []byte {
	t.Helper()
	src := image.NewRGBA(image.Rect(0, 0, 480, 240))
	for y := 0; y < 240; y++ {
		for x := 0; x < 480; x++ {
			src.Set(x, y, color.RGBA{R: 220, A: 255})
		}
	}
	var b bytes.Buffer
	if err := jpeg.Encode(&b, src, nil); err != nil {
		t.Fatal(err)
	}
	tiff := make([]byte, 26)
	copy(tiff, "II")
	binary.LittleEndian.PutUint16(tiff[2:], 42)
	binary.LittleEndian.PutUint32(tiff[4:], 8)
	binary.LittleEndian.PutUint16(tiff[8:], 1)
	binary.LittleEndian.PutUint16(tiff[10:], 0x112)
	binary.LittleEndian.PutUint16(tiff[12:], 3)
	binary.LittleEndian.PutUint32(tiff[14:], 1)
	binary.LittleEndian.PutUint16(tiff[18:], orientation)
	exif := append([]byte("Exif\x00\x00"), tiff...)
	header := []byte{0xff, 0xd8, 0xff, 0xe1, 0, 0}
	binary.BigEndian.PutUint16(header[4:], uint16(len(exif)+2))
	return append(append(header, exif...), b.Bytes()[2:]...)
}
func attachmentGIFFixture(t *testing.T, count, width, height, loops int) []byte {
	t.Helper()
	palette := color.Palette{color.RGBA{R: 255, A: 255}, color.RGBA{B: 255, A: 255}}
	g := &gif.GIF{LoopCount: loops, Config: image.Config{ColorModel: palette, Width: width, Height: height}}
	for i := 0; i < count; i++ {
		f := image.NewPaletted(image.Rect(0, 0, 2, 1), palette)
		f.Pix[0] = uint8(i % 2)
		f.Pix[1] = uint8(i % 2)
		g.Image = append(g.Image, f)
		g.Delay = append(g.Delay, 5)
	}
	var b bytes.Buffer
	if err := gif.EncodeAll(&b, g); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
func attachmentFixture(name string, data []byte) session.ComposerAttachment {
	return session.ComposerAttachment{Path: name, Name: name, ImageData: base64.StdEncoding.EncodeToString(data)}
}
func TestAttachmentJPEGThumbnailOrientation(t *testing.T) {
	for _, o := range []uint16{1, 2, 3, 4, 5, 6, 7, 8} {
		a := attachmentFixture("photo.jpg", attachmentJPEGFixture(t, o))
		thumb, full := readAttachmentDisplay(a), readAttachmentDisplay(a, true)
		if thumb.err != nil || full.err != nil || thumb.bitmap == nil || full.bitmap == nil {
			t.Fatalf("orientation %d: thumbnail=%+v full=%+v", o, thumb, full)
		}
		w, h := thumb.bitmap.Size()
		fw, fh := full.bitmap.Size()
		if o < 5 {
			if w != 240 || h != 120 || fw != 480 || fh != 240 {
				t.Fatalf("orientation %d: thumb %dx%d full %dx%d", o, w, h, fw, fh)
			}
		} else if w != 80 || h != 160 || fw != 240 || fh != 480 {
			t.Fatalf("orientation %d: thumb %dx%d full %dx%d", o, w, h, fw, fh)
		}
	}
}
func TestAttachmentGIFRejectsFrameAndCanvasBudgets(t *testing.T) {
	for _, data := range [][]byte{attachmentGIFFixture(t, imagepreview.MaxFrames+1, 2, 1, 0), attachmentGIFFixture(t, 2, 4000, 4001, 0)} {
		for _, full := range []bool{false, true} {
			if v := readAttachmentDisplay(attachmentFixture("moving.gif", data), full); v.err == nil || v.bitmap != nil {
				t.Fatalf("over-budget GIF accepted (full=%t)", full)
			}
		}
	}
}
func TestAttachmentOriginalGIFAdvancesAndDismisses(t *testing.T) {
	a := attachmentFixture("moving.gif", attachmentGIFFixture(t, 2, 2, 1, -1))
	v := readAttachmentDisplay(a, true)
	if v.err != nil {
		t.Fatal(v.err)
	}
	seat := attachmentDisplaySeat{open: true, full: &attachmentDisplaySeat{result: v}}
	tt := ui.NewTester(func(c *ui.Context) { paintAttachmentOriginal(c, &seat, a, a.Name) }, 600, 400)
	count := func(red bool) int {
		n := 0
		img := tt.Image()
		for y := 0; y < img.Bounds().Dy(); y++ {
			for x := 0; x < img.Bounds().Dx(); x++ {
				c := img.RGBAAt(x, y)
				if red && c.R > 220 && c.G < 40 && c.B < 40 || !red && c.B > 220 && c.R < 40 && c.G < 40 {
					n++
				}
			}
		}
		return n
	}
	if count(true) == 0 {
		t.Fatal("first GIF frame was not displayed")
	}
	time.Sleep(110 * time.Millisecond)
	tt.Frame()
	if count(false) == 0 {
		t.Fatal("original preview did not advance to second GIF frame")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if seat.open || seat.full != nil {
		t.Fatal("dismissed original retains full animation")
	}
}

func TestAttachmentGIFTimingAndThumbnailBudget(t *testing.T) {
	for _, full := range []bool{false, true} {
		for _, loops := range []int{-1, 0} {
			v := readAttachmentDisplay(attachmentFixture("moving.data", attachmentGIFFixture(t, 2, 480, 240, loops)), full)
			if v.err != nil {
				t.Fatal(v.err)
			}
			if v.motion == nil || len(v.motion.frames) != 2 || len(v.motion.animation.Frames) != 0 {
				t.Fatal("frames flattened or staging canvases retained")
			}
			w, h := v.bitmap.Size()
			if full {
				if w != 480 || h != 240 {
					t.Fatal("original dimensions changed")
				}
			} else if w != 240 || h != 120 {
				t.Fatalf("thumbnail exceeds bounds: %dx%d", w, h)
			}
			start := time.Unix(100, 0)
			if wait := v.advance(start); wait != 50*time.Millisecond || v.bitmap != v.motion.frames[0] {
				t.Fatal("initial frame timing", wait)
			}
			if wait := v.advance(start.Add(60 * time.Millisecond)); wait != 40*time.Millisecond || v.bitmap != v.motion.frames[1] {
				t.Fatal("second frame timing", wait)
			}
			wait := v.advance(start.Add(110 * time.Millisecond))
			if loops < 0 {
				if wait != 0 || v.bitmap != v.motion.frames[1] {
					t.Fatal("finite GIF did not stop")
				}
			} else if wait != 40*time.Millisecond || v.bitmap != v.motion.frames[0] {
				t.Fatal("loop/missed-frame timing changed", wait)
			}
		}
	}
}

func TestAttachmentGIFThumbnailRailAnimates(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := NewDraftState("")
		d.Attachments = []session.ComposerAttachment{attachmentFixture("moving.gif", attachmentGIFFixture(t, 2, 2, 1, -1))}
		tt := ui.NewTester(func(c *ui.Context) { paintAttachmentRail(c, Model{State: d}, ui.RGB(0, 0, 0), ui.RGB(80, 80, 80)) }, 600, 400)
		// The preview label exists during loading. Drain its asynchronous decode
		// before the first rendered frame starts the animation's paint clock.
		synctest.Wait()
		tt.Frame()
		sample := func() color.RGBA {
			t.Helper()
			rect, ok := tt.Find("Preview attachment moving.gif")
			if !ok {
				t.Fatal("GIF thumbnail disappeared")
			}
			return tt.Image().RGBAAt(int(rect.X+rect.W/2), int(rect.Y+rect.H/2))
		}
		first := sample()
		if first.R < 220 || first.B > 40 {
			t.Fatalf("initial GIF frame was not rendered: %+v", first)
		}
		time.Sleep(110 * time.Millisecond)
		tt.Frame()
		last := sample()
		if last.B < 220 || last.R > 40 {
			t.Fatalf("rail thumbnail remains first frame: %+v", last)
		}
		if err := tt.Click("Remove attachment moving.gif"); err != nil {
			t.Fatal(err)
		}
		if len(d.Attachments) != 0 {
			t.Fatal("animated attachment removal failed")
		}
	})
}
