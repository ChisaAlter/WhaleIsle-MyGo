package media

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"testing"
)

// rasterize decodes svg through a driven service and returns its pixels.
func rasterize(t *testing.T, svg string) image.Image {
	t.Helper()
	var out image.Image
	rasterHook = func(data []byte) { out, _ = png.Decode(bytes.NewReader(data)) }
	defer func() { rasterHook = nil }()
	s := New(Config{Workers: -1})
	key, src := Bytes([]byte(svg), ".svg")
	h := s.Request(nil, key, src)
	s.RunPending()
	if h.State() != Ready || h.Bitmap() == nil {
		t.Fatalf("not rasterised: %v", h.Failure())
	}
	if out == nil {
		t.Fatal("no raster output")
	}
	return out
}

func TestRasterSwitchKeepsOnlyFirstChild(t *testing.T) {
	img := rasterize(t, `<svg width="40" height="20"><switch><rect width="40" height="20" fill="red"/><rect width="40" height="20" fill="blue"/></switch><filter id="f"/></svg>`)
	r, g, b, a := img.At(20, 10).RGBA()
	if r != 65535 || g != 0 || b != 0 || a != 65535 {
		t.Fatalf("switch was dropped or flattened: %v %v %v %v", r, g, b, a)
	}
}

func TestRasterEmbeddedImageAndFilter(t *testing.T) {
	raster := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			raster.SetNRGBA(x, y, color.NRGBA{R: 255, A: 255})
		}
	}
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, raster); err != nil {
		t.Fatal(err)
	}
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="8"><defs><filter id="shift" x="-100%%" y="-100%%" width="400%%" height="400%%"><feOffset dx="4" dy="0"/></filter></defs><image width="2" height="2" href="data:image/png;base64,%s"/><rect x="10" width="2" height="2" fill="blue" filter="url(#shift)"/></svg>`, base64.StdEncoding.EncodeToString(buffer.Bytes()))
	img := rasterize(t, svg)
	if r, _, b, a := img.At(0, 0).RGBA(); r == 0 || b != 0 || a == 0 {
		t.Fatal("embedded raster omitted")
	}
	if _, _, b, a := img.At(14, 0).RGBA(); b == 0 || a == 0 {
		t.Fatal("static filter omitted")
	}
	if _, _, _, a := img.At(10, 0).RGBA(); a != 0 {
		t.Fatal("filter transform was ignored")
	}
	for _, href := range []string{"file:///secret.png", "https://example.com/image.png", "data:image/svg+xml;base64,PHN2Zy8+"} {
		if info := ClassifySVG([]byte(fmt.Sprintf(`<svg width="2" height="2"><image href="%s"/></svg>`, href))); info.Class != SVGUnsupported {
			t.Fatal("accepted external/nested SVG resource", href)
		}
	}
}

func TestRasterTextAndGradient(t *testing.T) {
	img := rasterize(t, `<svg xmlns="http://www.w3.org/2000/svg" width="200" height="60"><defs><linearGradient id="g"><stop offset="0" stop-color="red"/><stop offset="1" stop-color="blue"/></linearGradient></defs><rect width="20" height="20" fill="url(#g)"/><text x="30" y="40" font-size="28">Text</text></svg>`)
	if img.Bounds().Dx() != 200 || img.Bounds().Dy() != 60 {
		t.Fatal(img.Bounds())
	}
	if left, _, right, _ := img.At(2, 10).RGBA(); left <= right {
		t.Fatal("gradient content omitted")
	}
	painted := 0
	for y := 0; y < 60; y++ {
		for x := 30; x < 200; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a != 0 {
				painted++
			}
		}
	}
	if painted < 50 {
		t.Fatal("text content omitted", painted)
	}
	for _, bad := range []string{`<svg><image href="file:///secret"/></svg>`, `<svg width="999999" height="999999"/>`} {
		if ClassifySVG([]byte(bad)).Class != SVGUnsupported {
			t.Fatal("accepted unsafe SVG", bad)
		}
	}
}
