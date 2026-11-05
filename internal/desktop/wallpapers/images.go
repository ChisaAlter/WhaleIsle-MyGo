package wallpapers

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
	"image"
	"image/draw"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
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

const MaxImportBytes = 12 * 1024 * 1024
const MaxDataURLChars = 1800000

func ReadImportFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, MaxImportBytes+1))
	if err == nil && len(data) > MaxImportBytes {
		err = fmt.Errorf("wallpaper must be at most 12 MiB")
	}
	return data, err
}

// PrepareImport is the shared Host/local boundary. The crop editor retains a
// bounded private source, never the full decoded camera image.
func PrepareImport(data []byte) (image.Image, []byte, error) {
	if len(data) > MaxImportBytes {
		return nil, nil, fmt.Errorf("wallpaper must be at most 12 MiB")
	}
	img, err := Decode(data)
	if err != nil {
		return nil, nil, err
	}
	img = boundedImage(img)
	encoded, err := EncodeJPEG(img)
	return img, encoded, err
}
func boundedImage(img image.Image) image.Image {
	b := img.Bounds()
	scale := min(1.0, 1920.0/float64(max(b.Dx(), b.Dy())))
	dst := image.NewRGBA(image.Rect(0, 0, max(1, int(float64(b.Dx())*scale)), max(1, int(float64(b.Dy())*scale))))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Src, nil)
	return dst
}
func EncodeJPEG(img image.Image) ([]byte, error) {
	var out bytes.Buffer
	if err := jpeg.Encode(&out, boundedImage(img), &jpeg.Options{Quality: 82}); err != nil {
		return nil, err
	}
	if len("data:image/jpeg;base64,")+base64.StdEncoding.EncodedLen(out.Len()) > MaxDataURLChars {
		return nil, fmt.Errorf("wallpaper exceeds the 1,800,000 character image limit")
	}
	return out.Bytes(), nil
}
func ImageDataURL(data []byte) string {
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(data)
}

// Import retains a private copy; both persistence paths use the same codec.
func Import(dir string, data []byte) (string, error) {
	_, encoded, err := PrepareImport(data)
	if err != nil {
		return "", err
	}
	return writeEncodedImage(dir, encoded)
}
func writeImage(dir string, img image.Image) (string, error) {
	encoded, err := EncodeJPEG(img)
	if err != nil {
		return "", err
	}
	return writeEncodedImage(dir, encoded)
}
func writeEncodedImage(dir string, encoded []byte) (string, error) {
	sum := sha256.Sum256(encoded)
	path := filepath.Join(dir, fmt.Sprintf("%x.jpg", sum[:16]))
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, encoded, 0600); err != nil {
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
	data, err := ReadImportFile(original)
	if err != nil {
		return "", err
	}
	img, err := Decode(data)
	if err != nil {
		return "", err
	}
	return writeImage(dir, Cropped(img, c))
}
