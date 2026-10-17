package htmldocument

import (
	"bytes"
	"image"
	"image/png"
	"testing"
)

func TestFloatingImageBytesAreInertAndHaveNoLocalIdentity(t *testing.T) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	d, err := OpenImageBytes("/remote/image.png", buf.Bytes(), func() bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if d.CanRunScripts() {
		t.Fatal("image enabled scripts")
	}
	if path, err := d.ExternalPath(); err == nil || path != "" {
		t.Fatal("remote image exposed a local path")
	}
	if _, err := OpenImageBytes("image.png", buf.Bytes(), func() bool { return false }); err == nil {
		t.Fatal("stale source accepted")
	}
}
