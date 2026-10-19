package imagepreview

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"time"

	"golang.org/x/image/webp"
)

// IsAnimatedWebP identifies the animation feature in the required leading VP8X
// chunk. Full framing, dimensions and allocation limits are checked on decode.
func IsAnimatedWebP(data []byte) bool {
	return len(data) >= 30 && string(data[:4]) == "RIFF" && string(data[8:16]) == "WEBPVP8X" && data[20]&2 != 0
}

type webpFrame struct {
	rect             image.Rectangle
	data             []byte
	delay            time.Duration
	dispose, replace bool
}

func decodeWebPAnimation(data []byte) (*Animation, error) {
	if err := rasterInput(data); err != nil {
		return nil, err
	}
	if !IsAnimatedWebP(data) || uint64(binary.LittleEndian.Uint32(data[4:8]))+8 != uint64(len(data)) {
		return nil, errors.New("invalid animated WebP container")
	}
	var frames []webpFrame
	var canvas image.Rectangle
	var loops int
	var pixels int64
	header, animation := false, false
	for pos := 12; pos < len(data); {
		kind, payload, next, err := webpChunk(data, pos)
		if err != nil {
			return nil, err
		}
		pos = next
		switch kind {
		case "VP8X":
			if header || next != 30 || len(payload) != 10 || payload[0]&0xc1 != 0 || !avifZero(payload[1:4]) {
				return nil, errors.New("invalid animated WebP extended header")
			}
			w, h := webpUint24(payload[4:7])+1, webpUint24(payload[7:10])+1
			pixels, err = rasterPixels(uint64(w), uint64(h))
			if err != nil {
				return nil, err
			}
			canvas, header = image.Rect(0, 0, w, h), true
		case "ANIM":
			if !header || animation || len(frames) != 0 || len(payload) != 6 {
				return nil, errors.New("invalid WebP animation header")
			}
			// Match Chromium's image element: the ANIM background hint is
			// ignored, and the initial/disposed rectangle is transparent.
			loops = int(binary.LittleEndian.Uint16(payload[4:6]))
			animation = true
		case "ANMF":
			if !animation || len(payload) < 16 || payload[15]&0xfc != 0 {
				return nil, errors.New("invalid WebP animation frame")
			}
			x, y := 2*webpUint24(payload[:3]), 2*webpUint24(payload[3:6])
			w, h := webpUint24(payload[6:9])+1, webpUint24(payload[9:12])+1
			if x > canvas.Dx()-w || y > canvas.Dy()-h {
				return nil, errors.New("WebP frame exceeds canvas bounds")
			}
			if len(frames) >= MaxFrames || int64(len(frames)+1) > MaxPixels/pixels {
				return nil, errors.New("WebP animation exceeds the frame or pixel limit")
			}
			encoded, err := webpFrameImage(payload[16:], w, h)
			if err != nil {
				return nil, err
			}
			delay := time.Duration(webpUint24(payload[12:15])) * time.Millisecond
			if delay <= 10*time.Millisecond {
				delay = 100 * time.Millisecond
			}
			frames = append(frames, webpFrame{image.Rect(x, y, x+w, y+h), encoded, delay, payload[15]&1 != 0, payload[15]&2 != 0})
		case "ICCP", "EXIF", "XMP ":
			// Color/profile metadata does not carry resource authority. The shared
			// still WebP codec also ignores these chunks.
		case "ALPH", "VP8 ", "VP8L":
			return nil, errors.New("unsupported animated WebP chunk")
		default:
			// Unknown RIFF metadata has no rendering or resource authority.
		}
	}
	if !animation || len(frames) == 0 {
		return nil, errors.New("WebP animation has no frames")
	}
	// GIF uses repeat count after the first cycle; WebP uses total cycles.
	if loops == 1 {
		loops = -1
	} else if loops > 1 {
		loops--
	}
	a := &Animation{loops: loops}
	composite := image.NewNRGBA(canvas)
	for _, frame := range frames {
		decoded, err := webp.Decode(bytes.NewReader(frame.data))
		if err != nil {
			return nil, err
		}
		if decoded.Bounds().Dx() != frame.rect.Dx() || decoded.Bounds().Dy() != frame.rect.Dy() {
			return nil, errors.New("WebP frame dimensions differ from animation header")
		}
		op := draw.Over
		if frame.replace {
			op = draw.Src
		}
		draw.Draw(composite, frame.rect, decoded, decoded.Bounds().Min, op)
		snapshot := image.NewNRGBA(canvas)
		copy(snapshot.Pix, composite.Pix)
		a.Frames = append(a.Frames, snapshot)
		a.duration += frame.delay
		a.ends = append(a.ends, a.duration)
		if frame.dispose {
			draw.Draw(composite, frame.rect, image.NewUniform(color.Transparent), image.Point{}, draw.Src)
		}
	}
	return a, nil
}

func webpUint24(data []byte) int { return int(data[0]) | int(data[1])<<8 | int(data[2])<<16 }

func webpChunk(data []byte, pos int) (string, []byte, int, error) {
	if len(data)-pos < 8 {
		return "", nil, 0, errors.New("truncated WebP chunk")
	}
	size := uint64(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))
	padded := size + size%2
	if padded > uint64(len(data)-pos-8) {
		return "", nil, 0, errors.New("WebP chunk exceeds container bounds")
	}
	return string(data[pos : pos+4]), data[pos+8 : pos+8+int(size)], pos + 8 + int(padded), nil
}

// Rewrap one ANMF payload as a still WebP accepted by the existing codec. Check
// the actual VP8/VP8L dimensions before pixel decode rather than trusting VP8X.
func webpFrameImage(data []byte, width, height int) ([]byte, error) {
	alpha, bitstream := false, false
	for pos := 0; pos < len(data); {
		start := pos
		kind, _, next, err := webpChunk(data, pos)
		if err != nil {
			return nil, err
		}
		pos = next
		switch kind {
		case "ALPH":
			if alpha || bitstream {
				return nil, errors.New("invalid WebP frame alpha order")
			}
			alpha = true
		case "VP8 ", "VP8L":
			if bitstream || (kind == "VP8L" && alpha) {
				return nil, errors.New("invalid WebP frame bitstream")
			}
			cfg, err := webp.DecodeConfig(bytes.NewReader(webpStill(data[start:next], nil)))
			if err != nil {
				return nil, err
			}
			if cfg.Width != width || cfg.Height != height {
				return nil, errors.New("WebP bitstream dimensions differ from frame header")
			}
			bitstream = true
		case "VP8X", "ANIM", "ANMF":
			return nil, errors.New("nested WebP animation chunk")
		default:
			// Ancillary frame chunks are ignored by the still codec too.
		}
	}
	if !bitstream {
		return nil, errors.New("WebP frame has no bitstream")
	}
	var extended []byte
	if alpha {
		extended = make([]byte, 18)
		copy(extended, "VP8X")
		binary.LittleEndian.PutUint32(extended[4:8], 10)
		extended[8] = 16
		for i := 0; i < 3; i++ {
			extended[12+i] = byte((width - 1) >> (8 * i))
			extended[15+i] = byte((height - 1) >> (8 * i))
		}
	}
	return webpStill(data, extended), nil
}

func webpStill(data, extended []byte) []byte {
	wrapped := make([]byte, 12+len(extended)+len(data))
	copy(wrapped, "RIFF")
	binary.LittleEndian.PutUint32(wrapped[4:8], uint32(len(wrapped)-8))
	copy(wrapped[8:12], "WEBP")
	copy(wrapped[12:], extended)
	copy(wrapped[12+len(extended):], data)
	return wrapped
}
