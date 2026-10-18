package imagepreview

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestNativeSVGEmbeddedRasterAndFilter(t *testing.T) {
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
	svg := []byte(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="8"><defs><filter id="shift" x="-100%%" y="-100%%" width="400%%" height="400%%"><feOffset dx="4" dy="0"/></filter></defs><image width="2" height="2" href="data:image/png;base64,%s"/><rect x="10" width="2" height="2" fill="blue" filter="url(#shift)"/></svg>`, base64.StdEncoding.EncodeToString(buffer.Bytes())))
	if err := validateSVG(svg); err == nil {
		t.Fatal("isolated document validator was relaxed")
	}
	decoded, err := DecodeSVG(svg)
	if err != nil {
		t.Fatal(err)
	}
	r, _, b, a := decoded.At(0, 0).RGBA()
	if r == 0 || b != 0 || a == 0 {
		t.Fatal("embedded raster omitted")
	}
	_, _, b, a = decoded.At(14, 0).RGBA()
	if b == 0 || a == 0 {
		t.Fatal("static filter omitted")
	}
	if _, _, _, a = decoded.At(10, 0).RGBA(); a != 0 {
		t.Fatal("filter transform was ignored")
	}
	for _, href := range []string{"file:///secret.png", "https://example.com/image.png", "data:image/svg+xml;base64,PHN2Zy8+"} {
		if err := validateNativeSVG([]byte(fmt.Sprintf(`<svg width="2" height="2"><image href="%s"/></svg>`, href))); err == nil {
			t.Fatal("accepted external/nested SVG resource", href)
		}
	}
}
