// Package screenglance captures one explicitly requested monitor image.
// It has no timers, clipboard access, or network operations. Native macOS and
// Linux providers use temporary files as disclosed by StorageDetail.
package screenglance

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
)

const MaxWidth, MaxHeight = 768, 480

func thumbnailSize(width, height int) (int, int, error) {
	if width <= 0 || height <= 0 || width > 32768 || height > 32768 {
		return 0, 0, fmt.Errorf("screen glance: invalid monitor size")
	}
	w, h := width, height
	if w > MaxWidth {
		h = max(1, h*MaxWidth/w)
		w = MaxWidth
	}
	if h > MaxHeight {
		w = max(1, w*MaxHeight/h)
		h = MaxHeight
	}
	return w, h, nil
}

func encodeBGRA(pixels []byte, width, height int) ([]byte, error) {
	if width <= 0 || height <= 0 || width > MaxWidth || height > MaxHeight || len(pixels) != width*height*4 {
		return nil, fmt.Errorf("screen glance: invalid frame")
	}
	frame := image.NewRGBA(image.Rect(0, 0, width, height))
	defer clear(frame.Pix)
	for i := 0; i < len(pixels); i += 4 {
		frame.Pix[i], frame.Pix[i+1], frame.Pix[i+2], frame.Pix[i+3] = pixels[i+2], pixels[i+1], pixels[i], 255
	}
	var out bytes.Buffer
	if err := jpeg.Encode(&out, frame, &jpeg.Options{Quality: 70}); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
