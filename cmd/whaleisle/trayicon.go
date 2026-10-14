package main

import (
	"bytes"
	"fmt"
	"image"
	"image/png"

	"golang.org/x/image/draw"
)

// scalePNG downscales or upscales PNG bytes to a square size×size RGBA PNG.
// Uses Catmull-Rom interpolation for smoother tray icons than nearest-neighbor.
func scalePNG(data []byte, size int) ([]byte, error) {
	src, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	b := src.Bounds()
	if b.Dx() == size && b.Dy() == size {
		return data, nil
	}
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)
	var out bytes.Buffer
	if err := png.Encode(&out, dst); err != nil {
		return nil, fmt.Errorf("encode tray icon: %w", err)
	}
	return out.Bytes(), nil
}
