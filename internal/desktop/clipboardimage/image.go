// Package clipboardimage validates bytes explicitly supplied by a paste action.
// It never reads the OS clipboard or captures a screen.
package clipboardimage

import (
	"bytes"
	"fmt"
	"image/png"
	"io"
	"os"
)

const MaxBytes = 16 * 1024 * 1024
const MaxPixels = 24 * 1024 * 1024

func StorePNG(dir string, raw []byte) (path string, err error) {
	if len(raw) == 0 {
		return "", fmt.Errorf("The clipboard contains no image")
	}
	if len(raw) > MaxBytes {
		return "", fmt.Errorf("Clipboard image exceeds 16 MiB")
	}
	cfg, e := png.DecodeConfig(bytes.NewReader(raw))
	if e != nil || cfg.Width < 1 || cfg.Height < 1 || int64(cfg.Width)*int64(cfg.Height) > MaxPixels {
		return "", fmt.Errorf("Clipboard PNG is invalid or exceeds 24 megapixels")
	}
	// Decode the complete payload, not merely its header, before admission.
	if _, e = png.Decode(bytes.NewReader(raw)); e != nil {
		return "", fmt.Errorf("Clipboard PNG is incomplete: %w", e)
	}
	if err = os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	f, e := os.CreateTemp(dir, "pasted-*.png")
	if e != nil {
		return "", e
	}
	path = f.Name()
	defer func() {
		if err != nil {
			_ = f.Close()
			_ = os.Remove(path)
			path = ""
		}
	}()
	if _, err = io.Copy(f, bytes.NewReader(raw)); err != nil {
		return path, err
	}
	if err = f.Sync(); err != nil {
		return path, err
	}
	err = f.Close()
	return path, err
}
