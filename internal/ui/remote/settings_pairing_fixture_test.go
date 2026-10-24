package remote

import (
	"image"
	"image/color"
	"testing"
)

// pairingFixturePNG builds a QR-looking bitmap (corner eyes + pseudo-random
// modules) so the headless render shows a realistic pairing image.
func pairingFixturePNG(t *testing.T) image.Image {
	t.Helper()
	const size = 264
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	white := color.RGBA{255, 255, 255, 255}
	black := color.RGBA{17, 17, 17, 255}
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.Set(x, y, white)
		}
	}
	eye := func(ox, oy int) {
		for y := 0; y < 7; y++ {
			for x := 0; x < 7; x++ {
				if x == 0 || y == 0 || x == 6 || y == 6 || (x >= 2 && x <= 4 && y >= 2 && y <= 4) {
					for dy := 0; dy < 8; dy++ {
						for dx := 0; dx < 8; dx++ {
							img.Set(ox+x*8+dx, oy+y*8+dy, black)
						}
					}
				}
			}
		}
	}
	eye(8, 8)
	eye(size-64, 8)
	eye(8, size-64)
	seed := uint32(7)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			seed = seed*1664525 + 1013904223
			if seed%3 == 0 && (x+8)%8 < 7 && (y+8)%8 < 7 {
				img.Set(x, y, black)
			}
		}
	}
	return img
}
