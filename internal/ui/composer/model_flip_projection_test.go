package composer

import (
	"image"
	"image/color"
	"math"
	"testing"
)

func TestModelFlipProjectionIdentityNonzeroRasterOrigin(t *testing.T) {
	source := image.NewRGBA(image.Rect(-7, 9, 25, 25))
	for y := source.Rect.Min.Y; y < source.Rect.Max.Y; y++ {
		for x := source.Rect.Min.X; x < source.Rect.Max.X; x++ {
			source.SetRGBA(x, y, color.RGBA{R: uint8(x + 7), G: uint8(y - 9), B: 80, A: 160})
		}
	}
	projected, offset := projectModelFlip(source, 16, 8, 2, 0, 1)
	if projected == nil || projected.Bounds() != image.Rect(0, 0, 32, 16) || offset.X != 0 || offset.Y != 0 {
		t.Fatalf("identity changed raster geometry: image=%v offset=%v", projected, offset)
	}
	for y := range 16 {
		for x := range 32 {
			if got, want := projected.RGBAAt(x, y), source.RGBAAt(x-7, y+9); got != want {
				t.Fatalf("identity pixel (%d,%d)=%v want %v", x, y, got, want)
			}
		}
	}
}

func TestModelFlipProjectionPerspectiveNearAndFarEdges(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 800, 80))
	for y := range 80 {
		for x := range 800 {
			source.SetRGBA(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	// Independently calculated 140-DIP perspective corner bounds for a
	// 200x20-DIP seat at 4x scale. A flat vertical compression stays 800px wide.
	for _, test := range []struct {
		angle                    float64
		left, top, right, bottom int
	}{
		{80, -31, 33, 831, 48}, {-80, -31, 32, 831, 47},
	} {
		projected, offset := projectModelFlip(source, 200, 20, 4, test.angle, 1)
		if projected == nil {
			t.Fatalf("angle %v has no projection", test.angle)
		}
		if got := projected.Bounds(); got != image.Rect(0, 0, test.right-test.left, test.bottom-test.top) || offset.X != float32(test.left)/4 || offset.Y != float32(test.top)/4 {
			t.Fatalf("angle %v: bounds=%v offset=%v, want corner box [%d,%d,%d,%d]", test.angle, got, offset, test.left, test.top, test.right, test.bottom)
		}
		span := func(globalPixelY int) int {
			first, last := -1, -1
			for x := range projected.Bounds().Dx() {
				if projected.RGBAAt(x, globalPixelY-test.top).A < 200 {
					continue
				}
				if first < 0 {
					first = x
				}
				last = x
			}
			if first < 0 {
				t.Fatalf("angle %v has no ink at global y=%d", test.angle, globalPixelY)
			}
			// Both layers rotate around the common seat's x=100 DIP, rather than
			// anchoring a near edge or centering an individual ink bounding box.
			if math.Abs(float64(first+last-(2*(400-test.left)-1))) > 1 {
				t.Fatalf("angle %v row moved off the shared seat center: %d..%d", test.angle, first, last)
			}
			return last - first + 1
		}
		upper, lower := span(36), span(44)
		if test.angle > 0 && lower <= upper+40 || test.angle < 0 && upper <= lower+40 {
			t.Fatalf("angle %v lost the near/far trapezoid: upper=%d lower=%d", test.angle, upper, lower)
		}
	}
}

func TestModelFlipProjectionFractionalDPIUsesExactLogicalSeat(t *testing.T) {
	for _, test := range []struct {
		scale                    float32
		left, top, right, bottom int
	}{
		{1.25, -2, 8, 62, 13}, {1.5, -3, 10, 74, 16},
	} {
		const width, height float32 = 47.4, 17.1
		source := image.NewRGBA(image.Rect(0, 0, int(math.Ceil(float64(width)*float64(test.scale))), int(math.Ceil(float64(height)*float64(test.scale)))))
		projected, offset := projectModelFlip(source, width, height, test.scale, 80, 1)
		// The ceil-padded raster is 48 DIP wide at both scales. Treating that as
		// the seat changes the 1.25x top/bottom and the 1.5x right edge.
		if projected == nil || projected.Bounds() != image.Rect(0, 0, test.right-test.left, test.bottom-test.top) || offset.X != float32(test.left)/test.scale || offset.Y != float32(test.top)/test.scale {
			t.Fatalf("scale %v used ceil-padded center: image=%v offset=%v want pixel box [%d,%d,%d,%d]", test.scale, projected, offset, test.left, test.top, test.right, test.bottom)
		}
	}
}

func TestModelFlipProjectionBilinearPremultipliedOpacity(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 200, 4))
	source.SetRGBA(49, 0, color.RGBA{R: 100, A: 128})
	source.SetRGBA(50, 0, color.RGBA{G: 120, A: 160})
	source.SetRGBA(49, 1, color.RGBA{B: 180, A: 200})
	source.SetRGBA(50, 1, color.RGBA{R: 100, G: 100, B: 100, A: 240})
	projected, offset := projectModelFlip(source, 200, 4, 1, 60, .5)
	if projected == nil {
		t.Fatal("no premultiplied projection")
	}
	// Output center (50.5,1.5) DIP maps to source pixel coordinate
	// (49.6918922383,.4937756008). All four differently colored texels
	// contribute; nearest sampling or alpha-unpremultiplication changes this.
	got := projected.RGBAAt(50-int(offset.X), 1-int(offset.Y))
	want := color.RGBA{R: 25, G: 38, B: 31, A: 94}
	if got != want {
		t.Fatalf("bilinear premultiplied half opacity=%v want %v", got, want)
	}
	for y := range projected.Bounds().Dy() {
		for x := range projected.Bounds().Dx() {
			pixel := projected.RGBAAt(x, y)
			if pixel.R > pixel.A || pixel.G > pixel.A || pixel.B > pixel.A || pixel.A > 120 {
				t.Fatalf("invalid premultiplied/opacity edge at (%d,%d): %v", x, y, pixel)
			}
		}
	}
}
