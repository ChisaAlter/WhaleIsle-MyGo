package composer

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"testing"
	"whaleisle/internal/desktop/clipboardimage"
)

func TestClipboardImageIntakeFixture(t *testing.T) {
	var raw bytes.Buffer
	png.Encode(&raw, image.NewRGBA(image.Rect(0, 0, 3, 2)))
	path, err := clipboardimage.StorePNG(t.TempDir(), raw.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(saved, raw.Bytes()) {
		t.Fatal("image bytes changed", err)
	}
	dir := t.TempDir()
	for _, bad := range [][]byte{nil, []byte("not png"), raw.Bytes()[:30], make([]byte, clipboardimage.MaxBytes+1)} {
		if _, e := clipboardimage.StorePNG(dir, bad); e == nil {
			t.Fatal("invalid image accepted")
		}
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Fatal("rejected paste left attachments")
	}
	d := NewDraftState("keep draft")
	other := NewDraftState("other")
	if !PasteStillCurrent(d, d, "s", "s", 0) {
		t.Fatal("same draft rejected")
	}
	if PasteStillCurrent(d, other, "s", "s", 0) || PasteStillCurrent(d, d, "s", "other", 0) {
		t.Fatal("cross-session paste admitted")
	}
	d.EditSeq = 9
	if PasteStillCurrent(d, d, "s", "s", 0) {
		t.Fatal("late paste entered edit draft")
	}
	d.Sending = true
	if PasteStillCurrent(d, d, "s", "s", 9) {
		t.Fatal("paste mutated sending draft")
	}
	if d.Text != "keep draft" || other.Text != "other" {
		t.Fatal("text changed")
	}
}
