package avatarimage

import (
	"encoding/binary"
	"image"
	"image/color"
	"testing"
)

func TestApplyJPEGOrientationReusesCheckedMapping(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 3, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 3; x++ {
			src.Set(x, y, color.RGBA{R: uint8(10 + y*3 + x), A: 255})
		}
	}
	for o, want := range map[uint16]uint8{1: 10, 2: 12, 3: 15, 4: 13, 5: 10, 6: 13, 7: 15, 8: 12} {
		for _, order := range []binary.ByteOrder{binary.LittleEndian, binary.BigEndian} {
			tiff := make([]byte, 26)
			if order == binary.LittleEndian {
				copy(tiff, "II")
			} else {
				copy(tiff, "MM")
			}
			order.PutUint16(tiff[2:], 42)
			order.PutUint32(tiff[4:], 8)
			order.PutUint16(tiff[8:], 1)
			order.PutUint16(tiff[10:], 274)
			order.PutUint16(tiff[12:], 3)
			order.PutUint32(tiff[14:], 1)
			order.PutUint16(tiff[18:], o)
			exif := append([]byte("Exif\x00\x00"), tiff...)
			header := []byte{255, 216, 255, 225, 0, 0}
			binary.BigEndian.PutUint16(header[4:], uint16(len(exif)+2))
			got := ApplyJPEGOrientation(src, append(header, exif...))
			if c := color.RGBAModel.Convert(got.At(0, 0)).(color.RGBA); c.R != want {
				t.Fatalf("orientation %d: red=%d want=%d", o, c.R, want)
			}
			w, h := got.Bounds().Dx(), got.Bounds().Dy()
			if o < 5 {
				if w != 3 || h != 2 {
					t.Fatal("nonrotated bounds changed")
				}
			} else if w != 2 || h != 3 {
				t.Fatal("rotated bounds unchanged")
			}
		}
	}
	if ApplyJPEGOrientation(src, []byte("bad metadata")) != src || ApplyJPEGOrientation(nil, nil) != nil {
		t.Fatal("metadata/nil fallback changed")
	}
}
