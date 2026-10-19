package imagepreview

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"image"
	"image/png"
	"runtime"
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

// Supply the browser's initial serif family and 16px size without rewriting
// child styles or family choices. Root attributes/inline CSS stay authoritative.
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
		family, size := false, false
		for _, a := range root.Attr {
			family = family || a.Name.Local == "font-family" || a.Name.Local == "style" && strings.Contains(strings.ToLower(a.Value), "font-family")
			size = size || a.Name.Local == "font-size" || a.Name.Local == "style" && strings.Contains(strings.ToLower(a.Value), "font-size")
		}
		if family && size {
			return data
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
		if !family {
			// Original <img> SVG inherits the browser's initial serif family,
			// rather than the application's sans font or the bundled Go font.
			name := "DejaVu Serif"
			switch runtime.GOOS {
			case "windows":
				name = "Times New Roman"
			case "darwin":
				name = "Times"
			}
			result = append(result, []byte(` font-family="`+name+`"`)...)
		}
		if !size {
			result = append(result, []byte(` font-size="16"`)...)
		}
		result = append(result, data[end:]...)
		return result
	}
}
