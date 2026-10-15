package wallpapers

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
	"image"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"os"
	"path/filepath"
)

type Crop struct {
	Aspect float64 `json:"aspect"`
	Zoom   float64 `json:"zoom"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
}

func DefaultCrop() Crop { return Crop{Aspect: 16.0 / 9, Zoom: 1, X: .5, Y: .5} }
func Decode(data []byte) (image.Image, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	return img, err
}

// Import retains a private copy, so moving the selected external file cannot break the backdrop.
func Import(dir string, data []byte) (string, error) {
	img, err := Decode(data)
	if err != nil {
		return "", err
	}
	b := img.Bounds()
	scale := min(1.0, 1920.0/float64(max(b.Dx(), b.Dy())))
	dst := image.NewRGBA(image.Rect(0, 0, max(1, int(float64(b.Dx())*scale)), max(1, int(float64(b.Dy())*scale))))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Src, nil)
	return writeImage(dir, dst)
}
func writeImage(dir string, img image.Image) (string, error) {
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		return "", err
	}
	sum := sha256.Sum256(out.Bytes())
	path := filepath.Join(dir, fmt.Sprintf("%x.png", sum[:16]))
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, out.Bytes(), 0600); err != nil {
		return "", err
	}
	return path, nil
}
func Cropped(img image.Image, c Crop) image.Image {
	b := img.Bounds()
	w, h := float64(b.Dx()), float64(b.Dy())
	aspect := max(.25, c.Aspect)
	cw, ch := w, w/aspect
	if w/h >= aspect {
		cw, ch = h*aspect, h
	}
	cw = max(1, cw/max(1, c.Zoom))
	ch = max(1, ch/max(1, c.Zoom))
	x := b.Min.X + int((w-cw)*max(0, min(1, c.X)))
	y := b.Min.Y + int((h-ch)*max(0, min(1, c.Y)))
	rect := image.Rect(x, y, x+int(cw), y+int(ch)).Intersect(b)
	dst := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	draw.Draw(dst, dst.Bounds(), img, rect.Min, draw.Src)
	return dst
}
func SaveCrop(dir, original string, c Crop) (string, error) {
	data, err := os.ReadFile(original)
	if err != nil {
		return "", err
	}
	img, err := Decode(data)
	if err != nil {
		return "", err
	}
	return writeImage(dir, Cropped(img, c))
}
