package imagepreview

import (
	"encoding/binary"
	"image/color"
	"testing"
)

func nativeIconFixture(bpp uint16, pixels, mask []byte) []byte {
	palette := 0
	if bpp <= 8 {
		palette = 1 << bpp
	}
	dib := make([]byte, 40+palette*4+len(pixels)+len(mask))
	binary.LittleEndian.PutUint32(dib, 40)
	binary.LittleEndian.PutUint32(dib[4:], 2)
	binary.LittleEndian.PutUint32(dib[8:], 2)
	binary.LittleEndian.PutUint16(dib[12:], 1)
	binary.LittleEndian.PutUint16(dib[14:], bpp)
	for i := 0; i < palette; i++ {
		dib[40+i*4+2] = byte(i * 255 / (palette - 1))
	}
	copy(dib[40+palette*4:], pixels)
	copy(dib[40+palette*4+len(pixels):], mask)
	data := make([]byte, 22)
	binary.LittleEndian.PutUint16(data[2:], 1)
	binary.LittleEndian.PutUint16(data[4:], 1)
	data[6], data[7] = 2, 1
	binary.LittleEndian.PutUint16(data[10:], 1)
	binary.LittleEndian.PutUint16(data[12:], bpp)
	binary.LittleEndian.PutUint32(data[14:], uint32(len(dib)))
	binary.LittleEndian.PutUint32(data[18:], 22)
	return append(data, dib...)
}
func TestNativeIconDIBPixelsAndMask(t *testing.T) {
	for _, bpp := range []uint16{1, 4, 8, 24, 32} {
		var pixels []byte
		switch bpp {
		case 1:
			pixels = []byte{0x80, 0, 0, 0}
		case 4:
			pixels = []byte{0xf0, 0, 0, 0}
		case 8:
			pixels = []byte{255, 0, 0, 0}
		case 24:
			pixels = []byte{0, 0, 255, 0, 0, 0, 0, 0}
		case 32:
			pixels = []byte{0, 0, 255, 0, 0, 0, 0, 0}
		}
		img, err := DecodeIcon(nativeIconFixture(bpp, pixels, []byte{0x40, 0, 0, 0}))
		if err != nil {
			t.Fatal(bpp, err)
		}
		if c := color.NRGBAModel.Convert(img.At(0, 0)).(color.NRGBA); c.R != 255 || c.A != 255 {
			t.Fatal(bpp, c)
		}
		if _, _, _, a := img.At(1, 0).RGBA(); a != 0 {
			t.Fatal("AND mask lost", bpp)
		}
	}
}
func TestNativeIconAlphaAndTruncation(t *testing.T) {
	data := nativeIconFixture(32, []byte{0, 0, 255, 128, 0, 255, 0, 255}, []byte{0, 0, 0, 0})
	img, err := DecodeIcon(data)
	if err != nil {
		t.Fatal(err)
	}
	if c := color.NRGBAModel.Convert(img.At(0, 0)).(color.NRGBA); c.A != 128 || c.R != 255 {
		t.Fatal(c)
	}
	if _, err := DecodeIcon(data[:len(data)-1]); err == nil {
		t.Fatal("accepted truncated mask")
	}
}
