//go:build windows

package d3d11

import (
	"math"
	"runtime"
	"testing"

	"github.com/egoist/mygo/internal/gpu/gputest"
	"github.com/egoist/mygo/internal/scene"
)

func TestRawMaskGPUMatchesCPU(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	mask := scene.NewAtlas(1, 64, 64)
	// Uniform 25% coverage patch (64/255) plus an antialiased ring to exercise
	// every coverage value.
	patch25 := make([]byte, 12*12)
	for i := range patch25 {
		patch25[i] = 64
	}
	p25x, p25y, ok := mask.Alloc(12, 12)
	if !ok {
		t.Fatal("failed to allocate patch25 in mask atlas")
	}
	mask.Put(p25x, p25y, 12, 12, patch25, 12)

	ring := make([]byte, 16*16)
	for y := range 16 {
		for x := range 16 {
			dx, dy := float64(x)-7.5, float64(y)-7.5
			d := math.Sqrt(dx*dx + dy*dy)
			cov := max(0, min(1, 1.0-math.Abs(d-5.5)))
			ring[y*16+x] = byte(cov*255 + 0.5)
		}
	}
	rx, ry, ok := mask.Alloc(16, 16)
	if !ok {
		t.Fatal("failed to allocate ring in mask atlas")
	}
	mask.Put(rx, ry, 16, 16, ring, 16)

	muted := scene.Color{R: 104, G: 104, B: 104, A: 255}
	accent := scene.Color{R: 65, G: 118, B: 230, A: 255}
	semi := scene.Color{R: 104, G: 104, B: 104, A: 128}

	s := &scene.Scene{
		Width:     160,
		Height:    48,
		Clear:     scene.Color{R: 255, G: 255, B: 255, A: 255},
		MaskAtlas: mask,
		// Strong DirectWrite text contrast & gamma; RawMask must ignore both.
		Text: scene.TextParams{
			GammaRatios:      scene.GammaRatios(1.8),
			Contrast:         1.0,
			SubpixelContrast: 0.5,
		},
		Glyphs: []scene.Glyph{
			// 0: normal text mask (eats textCoverage inflation)
			{X: 8, Y: 8, W: 12, H: 12, U: uint16(p25x), V: uint16(p25y), UW: 12, VH: 12, Color: muted},
			// 1: raw mask (linear cov * color.a, no inflation)
			{X: 28, Y: 8, W: 12, H: 12, U: uint16(p25x), V: uint16(p25y), UW: 12, VH: 12, Color: muted, RawMask: true},
			// 2: translucent raw mask
			{X: 48, Y: 8, W: 12, H: 12, U: uint16(p25x), V: uint16(p25y), UW: 12, VH: 12, Color: semi, RawMask: true},
			// 3: raw mask antialiased ring in accent
			{X: 68, Y: 8, W: 16, H: 16, U: uint16(rx), V: uint16(ry), UW: 16, VH: 16, Color: accent, RawMask: true},
			// 4: raw mask ring filled with linear gradient
			{X: 92, Y: 8, W: 16, H: 16, U: uint16(rx), V: uint16(ry), UW: 16, VH: 16, Color: accent, RawMask: true},
		},
		Ops: []scene.Op{
			{Kind: scene.OpGlyphs, Start: 0, End: 4},
			{Kind: scene.OpGlyphs, Start: 4, End: 5, Paint: scene.PaintLinear, Color: accent, Color2: muted, Gradient: [4]float32{92, 8, 108, 24}},
		},
	}

	r, err := New(hiddenWindow(t, s.Width, s.Height))
	if err != nil {
		t.Skip("no Direct3D 11:", err)
	}
	defer r.Release()

	if err := r.draw(s); err != nil {
		t.Fatal(err)
	}
	pix, stride := r.readBack(t)
	gputest.Compare(t, "d3d11 rawmask", pix, stride, s)

	// Verify pixel values directly: linear blend for RawMask vs inflated textCoverage.
	// BGRA layout in readBack: index 2 is R.
	textR := int(pix[14*stride+14*4+2])
	rawR := int(pix[14*stride+34*4+2])
	wantLinear := int(math.Round(255*(1-64.0/255.0) + 104*(64.0/255.0))) // ~217
	if math.Abs(float64(rawR-wantLinear)) > 2 {
		t.Fatalf("D3D11 RawMask pixel R=%d, want linear ~%d", rawR, wantLinear)
	}
	if textR >= rawR-10 {
		t.Fatalf("expected normal text mask to be noticeably darker via textCoverage (textR=%d, rawR=%d)", textR, rawR)
	}
}
