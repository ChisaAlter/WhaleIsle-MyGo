package composer

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

func TestAttachmentAVIFSequenceThumbnailOriginalAndDismissal(t *testing.T) {
	data, err := os.ReadFile("../../desktop/imagepreview/testdata/test.avifs")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "moving.avifs")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	a := session.ComposerAttachment{Path: path, Name: "moving.avifs"}
	thumb := readAttachmentDisplay(a)
	if thumb.err != nil || thumb.motion == nil || len(thumb.motion.frames) != 17 || thumb.mediaType != "image/avif" {
		t.Fatalf("AVIS byte detection / native thumbnail: %+v", thumb)
	}
	if w, h := thumb.bitmap.Size(); w != 222 || h != 160 {
		t.Fatal("thumbnail budget", w, h)
	}
	start := time.Unix(100, 0)
	thumb.advance(start)
	if wait := thumb.advance(start.Add(90 * time.Millisecond)); wait != 70*time.Millisecond || thumb.bitmap != thumb.motion.frames[1] {
		t.Fatal("thumbnail not animated", wait)
	}
	v := readAttachmentDisplay(attachmentFixture("moving.avif", data), true)
	if v.err != nil || v.motion == nil || len(v.motion.animation.Frames) != 0 {
		t.Fatalf("native original/staging cleanup: %+v", v)
	}
	if w, h := v.bitmap.Size(); w != 500 || h != 360 {
		t.Fatal("original dimensions", w, h)
	}
	// The raster helpers are process-global and intentionally stay alive. Hire
	// the ordinary drawing team outside this bubble at its largest viewport.
	ui.NewTester(func(c *ui.Context) { ui.Box(c).Fill().Background(ui.RGB(255, 255, 255)) }, 700, 500)
	synctest.Test(t, func(t *testing.T) {
		seat := attachmentDisplaySeat{open: true, full: &attachmentDisplaySeat{result: v}}
		tt := ui.NewTester(func(c *ui.Context) { paintAttachmentOriginal(c, &seat, a, a.Name) }, 700, 500)
		before := append([]byte(nil), tt.Image().Pix...)
		time.Sleep(90 * time.Millisecond)
		tt.Frame()
		if v.bitmap != v.motion.frames[1] || bytes.Equal(before, tt.Image().Pix) {
			t.Fatal("native AVIF original did not visibly advance")
		}
		tt.Key(0, ui.KeyEscape)
		tt.Frame()
		if seat.open || seat.full != nil {
			t.Fatal("dismissed original retains AVIF animation")
		}
	})
}
