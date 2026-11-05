package frame

import (
	"github.com/egoist/mygo/ui"
)

// viewportDIP is the window's client area in DIPs, the unit every box in this
// frame is laid out in.
//
// ui.Context.Size documents DIPs but can report the device-pixel client instead.
// Measured at 150% (GetDpiForWindow=144) with the Wave 3 build: a window the
// desktop reports as a 1440x919 device-pixel client came back as 1440x919 from
// Context.Size, i.e. 1.5x the 960x613 DIPs the surface actually has. Every width
// threshold and the column resolution then get more room than the window and
// content is painted past the right edge.
//
// windowPixels reports the client rect this process sees: a size that matches it
// is divided by the DPI scale, and a size that already is DIP is left alone. The
// headless tester, other platforms and 100% displays have nothing to correct.
func viewportDIP(c *ui.Context) (width, height float32) {
	width, height = c.Size()
	pixels, pixelHeight, dpi := windowPixels()
	if pixels <= 0 || dpi <= 96 {
		return width, height
	}
	scale := float32(dpi) / 96
	if scale <= 1 {
		return width, height
	}
	// Device pixels: both dimensions match the client rect. A DIP size on the
	// same display is the client rect over the scale, so it cannot match.
	if within1(pixels, width) && within1(pixelHeight, height) {
		return width / scale, height / scale
	}
	return width, height
}

func within1(a, b float32) bool {
	diff := a - b
	return diff >= -1.5 && diff <= 1.5
}
