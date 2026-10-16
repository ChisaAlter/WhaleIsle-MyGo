package imagepreview

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"image"
	"image/png"
	"strings"
	"time"

	"whaleisle/internal/desktop/svgrender"
)

// DecodeSVG renders all accepted static SVG content, including text, using a
// trusted WASM rasterizer with no filesystem/network capabilities. This is image
// decoding, not page capture: it never creates a WebView or evaluates document JS.
func DecodeSVG(data []byte) (image.Image, error) {
	return DecodeSVGContext(context.Background(), data)
}

// DecodeSVGContext observes cancellation while running the isolated WASM decoder.
func DecodeSVGContext(ctx context.Context, data []byte) (image.Image, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := validateNativeSVG(data); err != nil {
		return nil, err
	}
	data = svgDefaultFont(data)
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	rendered, err := svgrender.Render(ctx, data, 0, 0)
	if err != nil {
		return nil, err
	}
	config, err := png.DecodeConfig(bytes.NewReader(rendered))
	if err != nil {
		return nil, err
	}
	if config.Width <= 0 || config.Height <= 0 || int64(config.Width) > MaxPixels/int64(config.Height) {
		return nil, errors.New("SVG decoded dimensions exceed the image budget")
	}
	return png.Decode(bytes.NewReader(rendered))
}

// Supply a bundled default family without rewriting child styles or family
// choices. An explicit root family or inline family remains authoritative.
func svgDefaultFont(data []byte) []byte {
	d := xml.NewDecoder(bytes.NewReader(data))
	for {
		token, err := d.Token()
		if err != nil {
			return data
		}
		root, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		for _, a := range root.Attr {
			if a.Name.Local == "font-family" || (a.Name.Local == "style" && strings.Contains(strings.ToLower(a.Value), "font-family")) {
				return data
			}
		}
		end := int(d.InputOffset()) - 1
		if end < 0 || end >= len(data) {
			return data
		}
		if end > 0 && data[end-1] == '/' {
			end--
		}
		result := make([]byte, 0, len(data)+18)
		result = append(result, data[:end]...)
		result = append(result, []byte(` font-family="Go"`)...)
		result = append(result, data[end:]...)
		return result
	}
}
