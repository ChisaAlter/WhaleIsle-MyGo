package main

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/egoist/mygo"
	"golang.org/x/image/draw"
)

// Icon rasters, matching the native shell:
//   - tray.js:19-21 resizes the tray image to 24x24 on win32 only;
//   - window.js:177-188 resizes the window/app icon to 48x48 on win32 (it
//     prefers icon.ico when one ships, else the PNG at 48).
const (
	trayIconSize   = 24
	windowIconSize = 48
)

// scalePNG downscales or upscales PNG bytes to a square size×size RGBA PNG.
// Uses Catmull-Rom interpolation for smoother icons than nearest-neighbor. The
// source bytes are never modified: an already square image is returned as is.
func scalePNG(data []byte, size int) ([]byte, error) {
	src, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	b := src.Bounds()
	if b.Dx() == size && b.Dy() == size {
		return data, nil
	}
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)
	var out bytes.Buffer
	if err := png.Encode(&out, dst); err != nil {
		return nil, fmt.Errorf("encode tray icon: %w", err)
	}
	return out.Bytes(), nil
}

// resizeTrayIcon returns the native tray raster: Windows gets 24x24
// (tray.js:20), every other platform keeps the source image, as tray.js:19 only
// resizes on win32.
func resizeTrayIcon(platform string, data []byte) ([]byte, error) {
	if platform != "windows" {
		return data, nil
	}
	return scalePNG(data, trayIconSize)
}

// windowIconPNG returns the native window icon raster: Windows gets 48x48
// (window.js:185), other platforms draw the application icon themselves and
// return nil. An already-right-sized source is passed through untouched.
func windowIconPNG(platform string, data []byte) ([]byte, error) {
	if platform != "windows" {
		return nil, nil
	}
	return scalePNG(data, windowIconSize)
}

// applyWindowIcon gives a window the native 48 DIP Windows icon. It is best
// effort: the window keeps the application icon when the asset or the native
// call is unavailable, exactly as the shell runs without a tray icon.
func applyWindowIcon(win *mygo.Window) {
	if win == nil || win.IsDestroyed() {
		return
	}
	icon := windowIcon()
	if len(icon) == 0 {
		return
	}
	if err := win.SetIcon(icon); err != nil {
		log.Println("window icon:", err)
	}
}

// windowIcon loads resources/icon.png and returns the native window raster.
func windowIcon() []byte {
	data, ok := readResourceIcon()
	if !ok {
		return nil
	}
	icon, err := windowIconPNG(runtime.GOOS, data)
	if err != nil {
		log.Println("window icon:", err)
		return nil
	}
	return icon
}

// readResourceIcon reads resources/icon.png from the working directory or the
// executable's directory, as the tray icon always has.
func readResourceIcon() ([]byte, bool) {
	for _, base := range []string{".", filepath.Dir(os.Args[0])} {
		data, err := os.ReadFile(filepath.Join(base, "resources", "icon.png"))
		if err != nil {
			continue
		}
		return data, true
	}
	return nil, false
}
