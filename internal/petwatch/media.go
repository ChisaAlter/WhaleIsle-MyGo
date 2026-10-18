package petwatch

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var errSticker = errors.New("sticker unavailable")

// ReadSticker accepts only an existing direct member of Whale's own sticker
// directory or the actual running plugin's bundled pack. It cannot read an
// arbitrary outbox path, follow a file symlink, open a URL or render SVG/HTML.
func ReadSticker(home, bundled, file string) ([]byte, string, error) {
	if !filepath.IsAbs(home) || !filepath.IsAbs(file) || strings.HasPrefix(file, `\\`) || strings.HasPrefix(file, "//") {
		return nil, "", errSticker
	}
	realHome, err := filepath.EvalSymlinks(home)
	if err != nil {
		return nil, "", errSticker
	}
	roots := []string{filepath.Join(realHome, "data", "whale", "stickers")}
	if bundled != "" && filepath.IsAbs(bundled) {
		roots = append(roots, bundled)
	}
	allowed := false
	for i, root := range roots {
		if filepath.Dir(file) != root {
			continue
		}
		real, err := filepath.EvalSymlinks(root)
		if err != nil || real != root {
			continue
		}
		if i == 0 {
			for _, p := range []string{filepath.Join(realHome, "data"), filepath.Join(realHome, "data", "whale"), root} {
				st, err := os.Lstat(p)
				if err != nil || !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
					return nil, "", errSticker
				}
			}
		}
		allowed = true
		break
	}
	if !allowed {
		return nil, "", errSticker
	}
	ext := strings.ToLower(filepath.Ext(file))
	mime := map[string]string{".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".gif": "image/gif", ".webp": "image/webp"}[ext]
	if mime == "" {
		return nil, "", errSticker
	}
	stat, err := os.Lstat(file)
	if err != nil || !stat.Mode().IsRegular() || stat.Size() <= 0 || stat.Size() > 2*1024*1024 {
		return nil, "", errSticker
	}
	f, err := os.Open(file)
	if err != nil {
		return nil, "", errSticker
	}
	defer f.Close()
	actual, err := f.Stat()
	if err != nil || !os.SameFile(stat, actual) {
		return nil, "", errSticker
	}
	data, err := io.ReadAll(io.LimitReader(f, 2*1024*1024+1))
	if err != nil || len(data) > 2*1024*1024 {
		return nil, "", errSticker
	}
	width, height := 0, 0
	if ext == ".webp" {
		width, height = webpDimensions(data)
	} else {
		cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			return nil, "", errSticker
		}
		if "image/"+format != mime {
			return nil, "", errSticker
		}
		width, height = cfg.Width, cfg.Height
	}
	if width <= 0 || height <= 0 || width > 4096 || height > 4096 || int64(width)*int64(height) > 16*1024*1024 {
		return nil, "", errSticker
	}
	return data, mime, nil
}

// WebP's bounded outer canvas headers avoid decoding pixels or importing a
// second runtime. The browser still validates the raster on the self-only URL.
func webpDimensions(b []byte) (int, int) {
	if len(b) < 20 || string(b[:4]) != "RIFF" || string(b[8:12]) != "WEBP" || uint64(binary.LittleEndian.Uint32(b[4:8]))+8 != uint64(len(b)) {
		return 0, 0
	}
	size := int(binary.LittleEndian.Uint32(b[16:20]))
	if size < 0 || size > len(b)-20 {
		return 0, 0
	}
	d := b[20:]
	le24 := func(b []byte) int { return int(b[0]) | int(b[1])<<8 | int(b[2])<<16 }
	switch string(b[12:16]) {
	case "VP8X":
		if size >= 10 {
			return le24(d[4:7]) + 1, le24(d[7:10]) + 1
		}
	case "VP8 ":
		if size >= 10 && d[3] == 0x9d && d[4] == 0x01 && d[5] == 0x2a {
			return int(binary.LittleEndian.Uint16(d[6:8]) & 0x3fff), int(binary.LittleEndian.Uint16(d[8:10]) & 0x3fff)
		}
	case "VP8L":
		if size >= 5 && d[0] == 0x2f {
			bits := binary.LittleEndian.Uint32(d[1:5])
			return int(bits&0x3fff) + 1, int((bits>>14)&0x3fff) + 1
		}
	}
	return 0, 0
}
