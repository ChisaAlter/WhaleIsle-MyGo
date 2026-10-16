// Package imagepreview validates bounded image containers for an inert browser
// image element. These are metadata/resource gates, not a codec implementation
// or a proof of the browser decoder's internal memory usage.
package imagepreview

import (
	"encoding/base64"
	"errors"
	"html"
	"path/filepath"
	"strings"
)

const MaxBytes int64 = 16 << 20
const MaxPixels int64 = 32_000_000
const MaxFrames = 256
const MaxSVGBytes = 1 << 20

func Supports(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".gif", ".ico", ".avif", ".svg":
		return true
	}
	return false
}

// Build receives one already-authorized immutable snapshot. It returns fixed
// markup containing only a data image, never raw supplied document markup.
// The caller must still wrap this in the static sandbox and deny network,
// scripts, bindings, popups, downloads, permissions and non-capability URLs.
func Build(path string, data []byte) (string, error) {
	if len(data) == 0 || int64(len(data)) > MaxBytes {
		return "", errors.New("image snapshot exceeds the 16 MiB byte budget")
	}
	var mime string
	var err error
	switch strings.ToLower(filepath.Ext(path)) {
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
	name := html.EscapeString(filepath.Base(path))
	return imageViewer(name, "data:"+mime+";base64,"+encoded), nil
}
