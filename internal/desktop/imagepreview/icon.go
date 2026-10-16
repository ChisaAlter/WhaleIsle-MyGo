package imagepreview

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/png"
)

// DecodeIcon validates every directory entry before choosing the largest image.
// Equal-size entries prefer greater bit depth. Both PNG and the bounded Windows
// DIB subset accepted by the isolated preview are decoded without external code.
func DecodeIcon(data []byte) (image.Image, error) {
	if err := validateICO(data); err != nil {
		return nil, err
	}
	count := int(binary.LittleEndian.Uint16(data[4:6]))
	var selected []byte
	best, depth := 0, 0
	for i := 0; i < count; i++ {
		e := data[6+i*16 : 6+(i+1)*16]
		w, h := int(e[0]), int(e[1])
		if w == 0 {
			w = 256
		}
		if h == 0 {
			h = 256
		}
		bits := int(binary.LittleEndian.Uint16(e[6:8]))
		if w*h > best || (w*h == best && bits > depth) {
			start := int(binary.LittleEndian.Uint32(e[12:16]))
			size := int(binary.LittleEndian.Uint32(e[8:12]))
			selected = data[start : start+size]
			best, depth = w*h, bits
		}
	}
	if bytes.HasPrefix(selected, []byte("\x89PNG\r\n\x1a\n")) {
		return png.Decode(bytes.NewReader(selected))
	}
	return decodeIconDIB(selected)
}

func decodeIconDIB(data []byte) (*image.NRGBA, error) {
	width, height, err := icoDIBDimensions(data, 0)
	if err != nil {
		return nil, err
	}
	w, h := int(width), int(height)
	bpp := int(binary.LittleEndian.Uint16(data[14:16]))
	colors := int(binary.LittleEndian.Uint32(data[32:36]))
	if bpp <= 8 && colors == 0 {
		colors = 1 << bpp
	}
	start := 40 + colors*4
	stride := ((w*bpp + 31) / 32) * 4
	maskStride := ((w + 31) / 32) * 4
	maskStart := start + stride*h
	alpha := false
	if bpp == 32 {
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				if data[start+y*stride+x*4+3] != 0 {
					alpha = true
				}
			}
		}
	}
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		row := data[start+(h-1-y)*stride:]
		mask := data[maskStart+(h-1-y)*maskStride:]
		for x := 0; x < w; x++ {
			c := color.NRGBA{A: 255}
			if bpp <= 8 {
				index := int(row[x*bpp/8]>>uint(8-bpp-(x*bpp)%8)) & ((1 << bpp) - 1)
				if index >= colors {
					return nil, errors.New("ICO pixel references an invalid palette entry")
				}
				at := 40 + index*4
				c.B, c.G, c.R = data[at], data[at+1], data[at+2]
			} else {
				at := x * (bpp / 8)
				c.B, c.G, c.R = row[at], row[at+1], row[at+2]
				if alpha {
					c.A = row[at+3]
				}
			}
			// Legacy all-zero alpha uses the AND bitmap; real 32-bit alpha wins.
			if !alpha && mask[x/8]&(0x80>>uint(x%8)) != 0 {
				c.A = 0
			}
			out.SetNRGBA(x, y, c)
		}
	}
	return out, nil
}
