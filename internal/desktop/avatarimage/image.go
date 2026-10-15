// Package avatarimage imports only explicitly selected local JPEG/PNG files.
// It never fetches URLs, discovers paths, or writes an image file.
package avatarimage

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
)

const MaxFileBytes = 8 << 20
const MaxSourcePixels = 8_000_000
const OutputSize = 192
const MaxDataURLChars = 120000

type Crop struct {
	Zoom, X, Y float64
	Shape      string
}

func DefaultCrop(shape string) Crop {
	if shape != "square" {
		shape = "circle"
	}
	return Crop{Zoom: 1, X: .5, Y: .5, Shape: shape}
}
func ValidCrop(c Crop) bool {
	return !math.IsNaN(c.Zoom) && !math.IsNaN(c.X) && !math.IsNaN(c.Y) && c.Zoom >= 1 && c.Zoom <= 3 && c.X >= 0 && c.X <= 1 && c.Y >= 0 && c.Y <= 1 && (c.Shape == "circle" || c.Shape == "square")
}

type Source struct {
	pixels               image.Image
	Name, Format         string
	Bytes, Width, Height int
}

func ReadPickedFile(path string) (*Source, error) {
	if !filepath.IsAbs(path) || strings.HasPrefix(path, `\\`) || strings.HasPrefix(path, "//") || strings.Contains(path, "://") {
		return nil, fmt.Errorf("select one ordinary local JPEG/PNG file")
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		return nil, fmt.Errorf("select JPEG or PNG")
	}
	before, err := os.Lstat(path)
	if err != nil || !before.Mode().IsRegular() || before.Mode()&os.ModeSymlink != 0 || before.Size() < 1 || before.Size() > MaxFileBytes {
		return nil, fmt.Errorf("selected image must be a regular local file of at most 8 MiB")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("selected image could not be opened")
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(before, opened) || opened.Size() != before.Size() {
		return nil, fmt.Errorf("selected file changed before reading")
	}
	data, err := io.ReadAll(io.LimitReader(f, MaxFileBytes+1))
	if err != nil || len(data) > MaxFileBytes || int64(len(data)) != opened.Size() {
		return nil, fmt.Errorf("selected file is too large or changed while reading")
	}
	after, err := f.Stat()
	if err != nil || !os.SameFile(opened, after) || after.Size() != opened.Size() || !after.ModTime().Equal(opened.ModTime()) {
		return nil, fmt.Errorf("selected file changed while reading")
	}
	expected := "jpeg"
	if ext == ".png" {
		expected = "png"
	}
	return Decode(data, filepath.Base(path), expected)
}
func Decode(data []byte, name, expected string) (*Source, error) {
	if len(data) < 1 || len(data) > MaxFileBytes {
		return nil, fmt.Errorf("image exceeds 8 MiB")
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "jpeg" && format != "png") || format != expected || cfg.Width < 1 || cfg.Height < 1 || cfg.Width > 4096 || cfg.Height > 4096 || int64(cfg.Width)*int64(cfg.Height) > MaxSourcePixels {
		return nil, fmt.Errorf("JPEG/PNG required: dimensions ≤4096 and pixels ≤8000000")
	}
	im, actual, err := image.Decode(bytes.NewReader(data))
	if err != nil || actual != format {
		return nil, fmt.Errorf("selected JPEG/PNG could not be decoded")
	}
	if format == "jpeg" {
		if o := orientation(data); o > 1 {
			im = oriented{Image: im, orientation: o}
		}
	}
	b := im.Bounds()
	return &Source{pixels: im, Name: name, Format: format, Bytes: len(data), Width: b.Dx(), Height: b.Dy()}, nil
}

// Bake samples the selected square, flattens transparency against black like
// canvas JPEG export, and removes original EXIF/metadata by re-encoding.
func (s *Source) Bake(c Crop) (string, error) {
	if s == nil || s.pixels == nil || !ValidCrop(c) {
		return "", fmt.Errorf("choose a valid crop before applying")
	}
	side := float64(min(s.Width, s.Height)) / c.Zoom
	x0 := float64(s.Width)*c.X - side*c.X
	y0 := float64(s.Height)*c.Y - side*c.Y
	out := image.NewRGBA(image.Rect(0, 0, OutputSize, OutputSize))
	for y := 0; y < OutputSize; y++ {
		for x := 0; x < OutputSize; x++ {
			sx := math.Max(0, math.Min(float64(s.Width-1), x0+(float64(x)+.5)*side/OutputSize-.5))
			sy := math.Max(0, math.Min(float64(s.Height-1), y0+(float64(y)+.5)*side/OutputSize-.5))
			ix, iy := int(sx), int(sy)
			fx, fy := sx-float64(ix), sy-float64(iy)
			var red, green, blue float64
			for dy := 0; dy < 2; dy++ {
				for dx := 0; dx < 2; dx++ {
					weight := 1.0
					if dx == 0 {
						weight *= 1 - fx
					} else {
						weight *= fx
					}
					if dy == 0 {
						weight *= 1 - fy
					} else {
						weight *= fy
					}
					r, g, b, _ := s.pixels.At(min(ix+dx, s.Width-1), min(iy+dy, s.Height-1)).RGBA()
					red += float64(r) * weight
					green += float64(g) * weight
					blue += float64(b) * weight
				}
			}
			out.SetRGBA(x, y, color.RGBA{uint8(red/257 + .5), uint8(green/257 + .5), uint8(blue/257 + .5), 255})
		}
	}
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, out, &jpeg.Options{Quality: 82}); err != nil {
		return "", fmt.Errorf("avatar JPEG encoding failed")
	}
	value := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(encoded.Bytes())
	if len(value) > MaxDataURLChars {
		return "", fmt.Errorf("cropped avatar exceeds Host size limit; choose a simpler crop")
	}
	return value, nil
}

// ValidateBaked restricts newly imported transport bytes to the actual native
// output format, independent of the UI/controller call path.
func ValidateBaked(value string) error {
	const prefix = "data:image/jpeg;base64,"
	if !strings.HasPrefix(value, prefix) || len(value) > MaxDataURLChars {
		return fmt.Errorf("new image must be the reviewed 192px JPEG crop")
	}
	data, err := base64.StdEncoding.Strict().DecodeString(value[len(prefix):])
	if err != nil {
		return fmt.Errorf("invalid JPEG encoding")
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || format != "jpeg" || cfg.Width != OutputSize || cfg.Height != OutputSize {
		return fmt.Errorf("new avatar must be a 192×192 JPEG")
	}
	if _, err := jpeg.Decode(bytes.NewReader(data)); err != nil {
		return fmt.Errorf("incomplete JPEG crop")
	}
	return nil
}

// Same checked EXIF orientation mapping as the existing native bitmap renderer,
// applied lazily to source pixels so no second full-size image is retained.
type oriented struct {
	image.Image
	orientation int
}

func (o oriented) Bounds() image.Rectangle {
	b := o.Image.Bounds()
	if o.orientation >= 5 {
		return image.Rect(0, 0, b.Dy(), b.Dx())
	}
	return image.Rect(0, 0, b.Dx(), b.Dy())
}
func (o oriented) At(x, y int) color.Color {
	b := o.Image.Bounds()
	w, h := b.Dx(), b.Dy()
	sx, sy := x, y
	switch o.orientation {
	case 2:
		sx = w - 1 - x
	case 3:
		sx, sy = w-1-x, h-1-y
	case 4:
		sy = h - 1 - y
	case 5:
		sx, sy = y, x
	case 6:
		sx, sy = y, h-1-x
	case 7:
		sx, sy = w-1-y, h-1-x
	case 8:
		sx, sy = w-1-y, x
	}
	return o.Image.At(b.Min.X+sx, b.Min.Y+sy)
}
func orientation(data []byte) int {
	if len(data) < 4 || data[0] != 255 || data[1] != 216 {
		return 0
	}
	for i := 2; i+4 <= len(data); {
		if data[i] != 255 {
			return 0
		}
		marker, size := data[i+1], int(binary.BigEndian.Uint16(data[i+2:]))
		if marker == 218 || size < 2 || i+2+size > len(data) {
			return 0
		}
		seg := data[i+4 : i+2+size]
		if marker == 225 && len(seg) > 14 && string(seg[:6]) == "Exif\x00\x00" {
			t := seg[6:]
			var bo binary.ByteOrder
			switch string(t[:2]) {
			case "II":
				bo = binary.LittleEndian
			case "MM":
				bo = binary.BigEndian
			default:
				return 0
			}
			offset := uint64(bo.Uint32(t[4:]))
			if offset < 8 || offset+2 > uint64(len(t)) {
				return 0
			}
			at := int(offset)
			n := int(bo.Uint16(t[at:]))
			for p := at + 2; p+12 <= len(t) && n > 0; p, n = p+12, n-1 {
				if bo.Uint16(t[p:]) == 274 && bo.Uint16(t[p+2:]) == 3 {
					v := int(bo.Uint16(t[p+8:]))
					if v >= 1 && v <= 8 {
						return v
					}
					return 0
				}
			}
			return 0
		}
		i += 2 + size
	}
	return 0
}

// Select runs on a worker. A dialog result is not authority to read after its
// owning view/Host/Bot ticket expires. Readers are injected only for fake tests.
func Select(choose func() ([]string, error), current func() bool, read func(string) (*Source, error)) (*Source, error) {
	if !current() {
		return nil, nil
	}
	paths, err := choose()
	if !current() {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("image chooser could not open")
	}
	if len(paths) == 0 {
		return nil, nil
	}
	if len(paths) != 1 || paths[0] == "" {
		return nil, fmt.Errorf("select one local JPEG/PNG file")
	}
	source, err := read(paths[0])
	if !current() {
		return nil, nil
	}
	return source, err
}
