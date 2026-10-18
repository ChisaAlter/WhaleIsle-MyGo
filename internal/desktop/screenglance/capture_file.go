//go:build darwin || (linux && (amd64 || arm64))

package screenglance

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"os"

	"golang.org/x/image/draw"
)

// Native screenshot providers return PNG files. Only the bounded JPEG reaches
// the existing model route; no full-resolution frame is sent to the Host.
func captureFile(ctx context.Context, path string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("screen glance: temporary screenshot unavailable")
	}
	defer f.Close()
	shape, _, err := image.DecodeConfig(f)
	if err != nil {
		return nil, fmt.Errorf("screen glance: unreadable screenshot")
	}
	w, h, err := thumbnailSize(shape.Width, shape.Height)
	if err != nil {
		return nil, err
	}
	if _, err := f.Seek(0, 0); err != nil {
		return nil, err
	}
	source, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("screen glance: unreadable screenshot")
	}
	switch pixels := source.(type) {
	case *image.RGBA:
		defer clear(pixels.Pix)
	case *image.NRGBA:
		defer clear(pixels.Pix)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	frame := image.NewRGBA(image.Rect(0, 0, w, h))
	defer clear(frame.Pix)
	draw.ApproxBiLinear.Scale(frame, frame.Bounds(), source, source.Bounds(), draw.Src, nil)
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, frame, &jpeg.Options{Quality: 70}); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		clear(encoded.Bytes())
		return nil, err
	}
	return encoded.Bytes(), nil
}
