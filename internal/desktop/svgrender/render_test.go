package svgrender

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"image/png"
	"testing"
)

func TestSVGRendererArtifactAndText(t *testing.T) {
	if got := fmt.Sprintf("%x", sha256.Sum256(wasmGz)); got != "c9dc354506393777df9bce99fc6143786f5cc6e990031cfd371dc4bb3f6eb78d" {
		t.Fatal("renderer artifact changed", got)
	}
	data, err := Render(context.Background(), []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="160" height="40"><text x="2" y="30" font-family="Go" font-size="28">Hello</text></svg>`), 160, 40)
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	painted := 0
	for y := 0; y < 40; y++ {
		for x := 0; x < 160; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a != 0 {
				painted++
			}
		}
	}
	if painted < 50 {
		t.Fatal("SVG text was omitted", painted)
	}
}
