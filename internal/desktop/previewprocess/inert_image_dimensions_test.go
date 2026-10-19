package previewprocess

import (
	"bytes"
	"math"
	"strings"
	"testing"
)

func TestInertImageDimensionsRejectOldHelperAndUnboundedMetadata(t *testing.T) {
	for _, size := range [][2]float64{{0, 0}, {80, 0}, {-1, 40}, {math.NaN(), 40}, {math.Inf(1), 40}, {32769, 1}, {6000, 6000}} {
		s := &Session{}
		if err := s.acceptImageDimensions(Document{InertImage: true}, packet{ImageWidth: size[0], ImageHeight: size[1]}); err == nil {
			t.Fatalf("accepted missing/invalid intrinsic dimensions %v", size)
		}
		if w, h := s.ImageSize(); w != 0 || h != 0 {
			t.Fatal("failed metadata mutated session")
		}
	}
	for _, size := range [][2]float64{{80, 160}, {.25, .5}, {32768, 1}, {8000, 4000}} {
		s := &Session{}
		if err := s.acceptImageDimensions(Document{InertImage: true}, packet{ImageWidth: size[0], ImageHeight: size[1]}); err != nil {
			t.Fatal(err)
		}
		if w, h := s.ImageSize(); w != size[0] || h != size[1] {
			t.Fatal("lost trusted fractional dimensions")
		}
	}
	if err := (&Session{}).acceptImageDimensions(Document{}, packet{ImageWidth: 80, ImageHeight: 160}); err == nil {
		t.Fatal("ordinary HTML accepted image-only metadata")
	}
}

func TestInertImageDocumentWireAndModeBoundary(t *testing.T) {
	d := Document{HTML: `<img src="data:image/svg+xml;base64,PHN2Zy8+">`, Width: 300, Appearance: "light", Inline: true, InertImage: true}
	if err := validateDocument(d); err != nil {
		t.Fatal(err)
	}
	nonce := strings.Repeat("a", 64)
	var wire bytes.Buffer
	if err := sendDocument(&wire, nonce, d); err != nil {
		t.Fatal(err)
	}
	got, err := receiveDocument(&wire, nonce)
	if err != nil || got != d {
		t.Fatalf("image-only wire state lost: %#v %v", got, err)
	}
	for _, mutate := range []func(*Document){func(d *Document) { d.Inline = false }, func(d *Document) { d.capture = true }, func(d *Document) { d.openUIShell = true }} {
		invalid := d
		mutate(&invalid)
		if err := validateDocument(invalid); err == nil {
			t.Fatalf("accepted incompatible image mode: %#v", invalid)
		}
	}
}
