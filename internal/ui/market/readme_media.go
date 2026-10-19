package market

import (
	"bytes"
	"context"
	"image"
	"io"
	"os"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/imagepreview"
	"whaleisle/internal/ui/svgimage"
)

var readmeDecodeSlots = make(chan struct{}, 2)

func (d *readmeDecodedImage) poll(path string) {
	if d.pending != nil {
		select {
		case result := <-d.pending:
			result.path = path
			*d = result
		default:
		}
	}
}

func (d *readmeDecodedImage) advance(now time.Time) time.Duration {
	if d.animation == nil || len(d.frames) == 0 {
		return 0
	}
	if d.started.IsZero() {
		d.started = now
	}
	index, wait := d.animation.FrameAt(now.Sub(d.started))
	d.bitmap = d.frames[index]
	return wait
}

func (d *readmeDecodedImage) load(path string, notify func()) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > imagepreview.MaxBytes {
		return
	}
	data, err := io.ReadAll(io.LimitReader(file, imagepreview.MaxBytes+1))
	if err != nil || int64(len(data)) > imagepreview.MaxBytes {
		return
	}
	if imagepreview.NeedsBrowserSVG(data) {
		d.svgSeat, _ = svgimage.NewSeat(data)
		if d.svgSeat != nil {
			d.width, d.height = d.svgSeat.Size()
		}
		return
	}
	cfg, format, bitmapErr := image.DecodeConfig(bytes.NewReader(data))
	if bitmapErr == nil && (cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width) > imagepreview.MaxPixels/int64(cfg.Height)) {
		return
	}
	animated := format == "gif" || imagepreview.IsAnimatedWebP(data) || imagepreview.IsAnimatedAVIF(data)
	if !animated && bitmapErr == nil {
		d.bitmap, _ = ui.DecodeBitmap(data)
		d.size()
		return
	}
	// Keep the existing immediate path for small supported vector artwork.
	// Full static SVG and animation decoding are bounded background work, owned
	// by this mounted image seat; closing it leaves no recurring timer.
	if bitmapErr != nil && supportedReadmeSVG(data) && fastReadmeSVG(data) {
		d.svg, _ = ui.ParseSVG(data)
		if d.svg != nil {
			d.width, d.height = d.svg.Size()
			return
		}
	}
	isSVG := supportedReadmeSVG(data)
	isICO := len(data) >= 4 && bytes.Equal(data[:4], []byte{0, 0, 1, 0})
	isAVIF := len(data) >= 12 && string(data[4:8]) == "ftyp"
	if !animated && !isSVG && !isICO && !isAVIF {
		return
	}
	d.pending = make(chan readmeDecodedImage, 1)
	result := d.pending
	go func() {
		readmeDecodeSlots <- struct{}{}
		loaded := decodeReadmeMedia(data, animated, isSVG, isICO)
		<-readmeDecodeSlots
		result <- loaded
		notify()
	}()
}

func decodeReadmeMedia(data []byte, animated, svg, ico bool) readmeDecodedImage {
	d := readmeDecodedImage{}
	if animated {
		var motion *imagepreview.Animation
		var err error
		if imagepreview.IsAnimatedAVIF(data) {
			motion, err = imagepreview.DecodeAVIFAnimationContext(context.Background(), data)
		} else {
			motion, err = imagepreview.DecodeAnimation(data)
		}
		if err != nil {
			return d
		}
		for i, frame := range motion.Frames {
			d.frames = append(d.frames, ui.NewBitmap(frame))
			motion.Frames[i] = nil
		}
		motion.Frames = nil
		d.animation, d.bitmap = motion, d.frames[0]
	} else {
		var decoded image.Image
		var err error
		switch {
		case svg:
			decoded, err = imagepreview.DecodeSVGContext(context.Background(), data)
		case ico:
			decoded, err = imagepreview.DecodeIcon(data)
		default:
			decoded, err = imagepreview.DecodeAVIFContext(context.Background(), data)
		}
		if err != nil || decoded == nil {
			return d
		}
		d.bitmap = ui.NewBitmap(decoded)
	}
	d.size()
	return d
}

func (d *readmeDecodedImage) size() {
	if d.bitmap != nil {
		width, height := d.bitmap.Size()
		d.width, d.height = float32(width), float32(height)
	}
}
