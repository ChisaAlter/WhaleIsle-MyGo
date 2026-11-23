package composer

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
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
	thumb := readSettled(a)
	if thumb.failure() != nil || !thumb.handle.Animated() || thumb.mediaType != "image/avif" {
		t.Fatalf("AVIS byte detection / native thumbnail: %+v", thumb)
	}
	if w, h := thumb.bitmap.Size(); w != 222 || h != 160 {
		t.Fatal("thumbnail budget", w, h)
	}
	start := time.Unix(100, 0)
	thumb.advance(start)
	if wait := thumb.advance(start.Add(90 * time.Millisecond)); wait != 70*time.Millisecond {
		t.Fatal("thumbnail not animated", wait)
	}
	v := readSettled(attachmentFixture("moving.avif", data), true)
	if v.failure() != nil || !v.handle.Animated() {
		t.Fatalf("native original/staging cleanup: %+v", v)
	}
	if w, h := v.bitmap.Size(); w != 500 || h != 360 {
		t.Fatal("original dimensions", w, h)
	}
	v.advance(start)
	if wait := v.advance(start.Add(90 * time.Millisecond)); wait != 70*time.Millisecond {
		t.Fatal("original frame timing", wait)
	}
	// Native rendering and its process-global raster workers stay outside
	// synctest. Exact timing is checked above; observe a real subsequent frame.
	seat := attachmentDisplaySeat{open: true, full: &attachmentDisplaySeat{result: v}}
	tt := ui.NewTester(func(c *ui.Context) { paintAttachmentOriginal(c, &seat, a, a.Name, nil) }, 700, 500)
	before := append([]byte(nil), tt.Image().Pix...)
	changed := false
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); {
		time.Sleep(5 * time.Millisecond)
		tt.Frame()
		if !bytes.Equal(before, tt.Image().Pix) {
			changed = true
			break
		}
	}
	if !changed {
		t.Fatal("native AVIF original did not visibly advance")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	current := v.bitmap
	for i := 0; i < 3; i++ {
		tt.Frame()
	}
	if seat.open || seat.full != nil || v.bitmap != current {
		t.Fatal("dismissed original retains or advances AVIF animation")
	}
}
