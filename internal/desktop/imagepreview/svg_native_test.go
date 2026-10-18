package imagepreview

import (
	"testing"
)

func TestNativeSVGContentAndIsolation(t *testing.T) {
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="200" height="60"><defs><linearGradient id="g"><stop offset="0" stop-color="red"/><stop offset="1" stop-color="blue"/></linearGradient></defs><rect width="20" height="20" fill="url(#g)"/><text x="30" y="40" font-size="28">Text</text></svg>`)
	img, err := DecodeSVG(svg)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 200 || img.Bounds().Dy() != 60 {
		t.Fatal(img.Bounds())
	}
	left, _, right, _ := img.At(2, 10).RGBA()
	if left <= right {
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
	for _, bad := range []string{`<svg><script>alert(1)</script></svg>`, `<svg><image href="file:///secret"/></svg>`, `<svg width="999999" height="999999"/>`} {
		if _, err := DecodeSVG([]byte(bad)); err == nil {
			t.Fatal("accepted unsafe SVG", bad)
		}
	}
}
