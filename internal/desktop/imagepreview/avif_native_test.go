package imagepreview

import (
	"bytes"
	"image"
	"os"
	"strings"
	"testing"
)

func TestNativeAVIFPixelsDepthAndOrientation(t *testing.T) {
	for _, tc := range []struct {
		name          string
		width, height int
		deep          bool
	}{
		// Upstream test_rot pixi is 00000000 03 0c0c0c: three 12-bit channels.
		{"test_rot.avif", 480, 640, true}, {"test10.avif", 512, 512, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := os.ReadFile("testdata/" + tc.name)
			if err != nil {
				t.Fatal(err)
			}
			img, err := DecodeAVIF(data)
			if err != nil {
				t.Fatal(err)
			}
			if img.Bounds().Dx() != tc.width || img.Bounds().Dy() != tc.height {
				t.Fatal(img.Bounds())
			}
			_, deep := img.(*image.RGBA64)
			if deep != tc.deep {
				t.Fatalf("unexpected pixel depth %T", img)
			}
			_, _, _, a := img.At(0, 0).RGBA()
			if a != 65535 {
				t.Fatal("lost opaque pixels", a)
			}
			// A second module must not overwrite or free the returned frame's pixels.
			old := img.At(0, 0)
			if _, err := DecodeAVIF(data); err != nil {
				t.Fatal(err)
			}
			if img.At(0, 0) != old {
				t.Fatal("pixels alias decoder memory")
			}
		})
	}
}
func TestNativeAVIFRejectsAnimationRatherThanFlattening(t *testing.T) {
	data, err := os.ReadFile("testdata/test.avifs")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeAVIF(data); err == nil || !strings.Contains(err.Error(), "animated AVIF") {
		t.Fatal(err)
	}
	for _, bad := range [][]byte{nil, []byte("not an AVIF"), make([]byte, MaxBytes+1)} {
		if _, err := DecodeAVIF(bad); err == nil {
			t.Fatal("invalid AVIF accepted")
		}
	}
}

func TestNativeAVIFRejectsCorruptBitstream(t *testing.T) {
	data, err := os.ReadFile("testdata/test_rot.avif")
	if err != nil {
		t.Fatal(err)
	}
	offset := bytes.Index(data, []byte("mdat"))
	if offset < 0 {
		t.Fatal("fixture missing media")
	}
	clear(data[offset+4:])
	if _, err := DecodeAVIF(data); err == nil {
		t.Fatal("corrupt AV1 payload produced pixels")
	}
}
