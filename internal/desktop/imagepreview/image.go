// Package imagepreview validates bounded image containers for an inert browser
// image element. These are metadata/resource gates, not a codec implementation
// or a proof of the browser decoder's internal memory usage.
package imagepreview

import (
	"bytes"
	"encoding/base64"
	"errors"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"
	"html"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"path/filepath"
	"strings"
)

const MaxBytes int64 = 32 << 20
const MaxPixels int64 = 32_000_000
const MaxFrames = 256
const MaxSVGBytes = 32 << 20

func Supports(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".gif", ".ico", ".avif", ".svg":
		return true
	}
	return false
}

// SupportsSnapshot includes native bitmap formats without changing the normal
// Files editor's choice of native versus browser image rendering.
func SupportsSnapshot(path string) bool {
	if Supports(path) {
		return true
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".webp", ".bmp":
		return true
	}
	return false
}

// Build receives one already-authorized immutable snapshot. It returns fixed
// markup containing only a data image, never raw supplied document markup.
// The caller must still wrap this in the static sandbox and deny network,
// scripts, bindings, popups, downloads, permissions and non-capability URLs.
func Build(path string, data []byte) (string, error) {
	source, err := imageDataURL(path, data)
	if err != nil {
		return "", err
	}
	return imageViewer(html.EscapeString(filepath.Base(path)), source), nil
}

// imageDataURL is shared by the local fixed viewer and the public-image adapter.
// Both paths keep exactly the same byte, dimension and container validation.
func imageDataURL(path string, data []byte) (string, error) {
	if len(data) == 0 || int64(len(data)) > MaxBytes {
		return "", errors.New("image snapshot exceeds the 32 MiB byte budget")
	}
	var mime string
	var err error
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".webp", ".bmp":
		cfg, format, decodeErr := image.DecodeConfig(bytes.NewReader(data))
		if decodeErr != nil {
			return "", decodeErr
		}
		if cfg.Width < 1 || cfg.Height < 1 || int64(cfg.Width) > MaxPixels/int64(cfg.Height) {
			return "", errors.New("image exceeds pixel budget")
		}
		switch format {
		case "png", "jpeg", "webp", "bmp":
			mime = "image/" + format
		default:
			return "", errors.New("unsupported bitmap format")
		}
	case ".gif":
		mime, err = "image/gif", validateGIF(data)
	case ".ico":
		mime, err = "image/x-icon", validateICO(data)
	case ".avif":
		mime, err = "image/avif", validateAVIF(data)
	case ".svg":
		mime, err = "image/svg+xml", validateSVG(data)
	default:
		return "", errors.New("this format already uses the native image preview")
	}
	if err != nil {
		return "", err
	}
	encoded := base64.StdEncoding.EncodeToString(data)
	return "data:" + mime + ";base64," + encoded, nil
}
