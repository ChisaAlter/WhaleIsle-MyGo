package avatarimage

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBotAvatarImageBoundedLocalSelectionAndCrop(t *testing.T) {
	im := image.NewRGBA(image.Rect(0, 0, 160, 80))
	for y := 0; y < 80; y++ {
		for x := 0; x < 160; x++ {
			v := color.RGBA{230, 10, 10, 255}
			if x >= 80 {
				v = color.RGBA{10, 10, 230, 255}
			}
			im.SetRGBA(x, y, v)
		}
	}
	var p, j bytes.Buffer
	png.Encode(&p, im)
	jpeg.Encode(&j, im, &jpeg.Options{Quality: 90})
	dir := t.TempDir()
	path := filepath.Join(dir, "chosen.png")
	os.WriteFile(path, p.Bytes(), 0600)
	source, err := ReadPickedFile(path)
	if err != nil || source.Name != "chosen.png" || source.Width != 160 || source.Height != 80 {
		t.Fatal("selected PNG", err)
	}
	jp := filepath.Join(dir, "chosen.jpg")
	os.WriteFile(jp, j.Bytes(), 0600)
	if _, err := ReadPickedFile(jp); err != nil {
		t.Fatal(err)
	}
	t.Run("bounded-file-and-format", func(t *testing.T) {
		mismatch := filepath.Join(dir, "mismatch.jpg")
		os.WriteFile(mismatch, p.Bytes(), 0600)
		link := filepath.Join(dir, "link.png")
		linkAvailable := os.Symlink(path, link) == nil
		if !linkAvailable {
			t.Log("symlink privilege unavailable; skipping the symlink-specific path assertion")
		}
		large := filepath.Join(dir, "large.png")
		f, _ := os.Create(large)
		f.Truncate(MaxFileBytes + 1)
		f.Close()
		badPaths := []string{"https://fixture.invalid/a.png", "relative.png", "//server/a.png", dir, mismatch, large}
		if linkAvailable {
			badPaths = append(badPaths, link)
		}
		for _, bad := range badPaths {
			if _, err := ReadPickedFile(bad); err == nil {
				t.Fatal("unsafe selected path", bad)
			}
		}
		// Mutate PNG IHDR dimensions and CRC only. DecodeConfig rejects before IDAT
		// can be decoded or any oversized pixel allocation is attempted.
		for _, wh := range [][2]uint32{{4097, 1}, {3000, 3000}, {0, 1}} {
			b := append([]byte(nil), p.Bytes()...)
			binary.BigEndian.PutUint32(b[16:20], wh[0])
			binary.BigEndian.PutUint32(b[20:24], wh[1])
			binary.BigEndian.PutUint32(b[29:33], crc32.ChecksumIEEE(b[12:29]))
			if _, err := Decode(b, "bounded.png", "png"); err == nil {
				t.Fatal("config bound", wh)
			}
		}
		if _, err := Decode([]byte("GIF89a"), "x.png", "png"); err == nil {
			t.Fatal("unsupported format")
		}
		if _, err := Decode(p.Bytes(), "x.jpg", "jpeg"); err == nil {
			t.Fatal("MIME mismatch")
		}
	})
	t.Run("192-jpeg-pan-zoom-shape-and-metadata", func(t *testing.T) {
		crop := DefaultCrop("circle")
		crop.X = 0
		left, err := source.Bake(crop)
		if err != nil {
			t.Fatal(err)
		}
		crop.X = 1
		right, _ := source.Bake(crop)
		read := func(url string) image.Image {
			b, err := base64.StdEncoding.DecodeString(strings.SplitN(url, ",", 2)[1])
			if err != nil {
				t.Fatal(err)
			}
			im, err := jpeg.Decode(bytes.NewReader(b))
			if err != nil {
				t.Fatal(err)
			}
			return im
		}
		a, b := read(left), read(right)
		ar, _, ab, _ := a.At(96, 96).RGBA()
		br, _, bb, _ := b.At(96, 96).RGBA()
		if a.Bounds().Dx() != 192 || a.Bounds().Dy() != 192 || ar < ab*5 || bb < br*5 || ValidateBaked(left) != nil {
			t.Fatal("pan/crop/output")
		}
		crop.Shape = "square"
		same, _ := source.Bake(crop)
		if same != right {
			t.Fatal("display shape changed baked square")
		}
		crop.Zoom = 3
		zoom, _ := source.Bake(crop)
		if ValidateBaked(zoom) != nil {
			t.Fatal("zoom")
		}
		for _, bad := range []Crop{{Zoom: math.NaN(), X: .5, Y: .5, Shape: "circle"}, {Zoom: 4, X: .5, Y: .5, Shape: "circle"}, {Zoom: 1, X: -1, Y: .5, Shape: "circle"}, {Zoom: 1, X: .5, Y: 2, Shape: "circle"}, {Zoom: 1, X: .5, Y: .5, Shape: "unknown"}} {
			if _, err := source.Bake(bad); err == nil {
				t.Fatal("bad crop")
			}
		}
		// Synthetic EXIF orientation 6 (90° clockwise), no user image/metadata.
		exif := []byte{'E', 'x', 'i', 'f', 0, 0, 'I', 'I', 42, 0, 8, 0, 0, 0, 1, 0, 18, 1, 3, 0, 1, 0, 0, 0, 6, 0, 0, 0, 0, 0, 0, 0}
		data := append([]byte{255, 216, 255, 225, 0, byte(len(exif) + 2)}, exif...)
		data = append(data, j.Bytes()[2:]...)
		rotated, err := Decode(data, "rotated.jpg", "jpeg")
		if err != nil || rotated.Width != 80 || rotated.Height != 160 {
			t.Fatal("orientation", err)
		}
		output, err := rotated.Bake(DefaultCrop("square"))
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := base64.StdEncoding.DecodeString(strings.SplitN(output, ",", 2)[1])
		if bytes.Contains(raw, []byte("Exif")) {
			t.Fatal("original metadata retained")
		}
		for _, bad := range []string{"data:image/jpeg;base64,!", "data:image/png;base64," + base64.StdEncoding.EncodeToString(p.Bytes()), "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(j.Bytes()), left + strings.Repeat("A", MaxDataURLChars)} {
			if ValidateBaked(bad) == nil {
				t.Fatal("bad transport")
			}
		}
	})
	t.Run("stale-dialog-never-reads", func(t *testing.T) {
		live := true
		reads := 0
		read := func(string) (*Source, error) { reads++; return source, nil }
		got, err := Select(func() ([]string, error) { live = false; return []string{path}, nil }, func() bool { return live }, read)
		if got != nil || err != nil || reads != 0 {
			t.Fatal("stale selection read")
		}
		live = true
		got, err = Select(func() ([]string, error) { return nil, nil }, func() bool { return live }, read)
		if got != nil || err != nil || reads != 0 {
			t.Fatal("cancel read")
		}
		_, err = Select(func() ([]string, error) { return nil, errors.New("private path") }, func() bool { return live }, read)
		if err == nil || strings.Contains(err.Error(), "private") {
			t.Fatal("chooser error")
		}
		got, err = Select(func() ([]string, error) { return []string{path}, nil }, func() bool { return live }, func(string) (*Source, error) { live = false; reads++; return source, nil })
		if got != nil || err != nil || reads != 1 {
			t.Fatal("stale decode result retained")
		}
	})
}
