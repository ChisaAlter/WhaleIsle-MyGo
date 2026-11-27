package glass

import (
	"runtime"
	"testing"

	"github.com/egoist/mygo/internal/gpu"
	"github.com/egoist/mygo/internal/gpu/d3d11"
	"github.com/egoist/mygo/internal/gpu/gputest"
	"github.com/egoist/mygo/internal/scene"
)

// TestDirect3D checks that Direct3D 11 draws the glass as the CPU does.
func TestDirect3D(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	s := testScene()
	pix, stride, err := d3d11.RenderOffscreen(s)
	if err != nil {
		t.Skip("no Direct3D 11:", err)
	}
	gputest.Compare(t, "glass-d3d11", pix, stride, s)
	s = blurScene()
	if pix, stride, err = d3d11.RenderOffscreen(s); err != nil {
		t.Fatal(err)
	}
	gputest.Compare(t, "blur-d3d11", pix, stride, s)
	s = ambientScene()
	if pix, stride, err = d3d11.RenderOffscreen(s); err != nil {
		t.Fatal(err)
	}
	gputest.Compare(t, "ambient-d3d11", pix, stride, s)
}

// TestBlurSinglePassTintDirect3D verifies that glass.Blur with Tint, TintMix,
// and LuminosityOffset renders identically on D3D11 and CPU rasterizer.
func TestBlurSinglePassTintDirect3D(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	s := blurScene()
	p2, p3 := blurTone{
		saturation: 0.8,
		offset:     0.03,
		mix:        0.44,
		color:      [3]float32{1, 1, 1},
		dither:     true,
	}.params()
	s.Ops = append(s.Ops, scene.Op{Kind: scene.OpEffect, Rect: scene.Rect{X: 16, Y: 16, W: 120, H: 60}, Radii: r4(12), Start: int32(len(s.Effects))})
	s.Effects = append(s.Effects, scene.EffectOp{Effect: blurEffect, Blur: 24, Params: [5][4]float32{{}, {24, 24, -1, 0}, p2, p3}})
	pix, stride, err := d3d11.RenderOffscreen(s)
	if err != nil {
		t.Skip("no Direct3D 11:", err)
	}
	gputest.Compare(t, "blur-tint-d3d11", pix, stride, s)
}

// TestBytecode checks that the bytecode compiled ahead of time comes from
// glass.hlsl, blur.hlsl and ambient.hlsl and the renderer's head and tail as
// they are.
func TestBytecode(t *testing.T) {
	if gpu.SourceSum(d3d11.EffectSource(hlslSource)) != hlslSum {
		t.Error("glass.hlsl or the Direct3D renderer's effect.hlsl changed since shaders_windows.go was generated: run go generate ./plugins/glass on Windows")
	}
	if gpu.SourceSum(d3d11.EffectSource(blurHLSLSource)) != blurHLSLSum {
		t.Error("blur.hlsl or the Direct3D renderer's effect.hlsl changed since shaders_windows.go was generated: run go generate ./plugins/glass on Windows")
	}
	if gpu.SourceSum(d3d11.EffectSource(ambientHLSLSource)) != ambientHLSLSum {
		t.Error("ambient.hlsl or the Direct3D renderer's effect.hlsl changed since shaders_windows.go was generated: run go generate ./plugins/glass on Windows")
	}
}
