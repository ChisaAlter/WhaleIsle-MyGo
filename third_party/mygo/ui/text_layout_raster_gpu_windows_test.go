//go:build windows && (amd64 || arm64)

package ui

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/egoist/mygo/internal/gpu/d3d11"
	"github.com/egoist/mygo/internal/gpu/gputest"
	"github.com/egoist/mygo/internal/scene"
)

// Exercises the actual native glyph -> owned RGBA -> DecorativeRaster.Paint
// -> Painter.Image -> D3D11 texture upload/draw/readback path. RenderOffscreen
// owns an existing never-shown fixture window. The device may be WARP; this
// proves the D3D11 path, not presentation on physical GPU hardware.
func TestTextLayoutRasterD3D11ImageUploadReadback(t *testing.T) {
	layout := ShapeRichText([]Span{
		{Text: "Native ", Color: RGB(255, 220, 60), Underline: true},
		{Text: "鲸屿 image upload", Color: RGB(90, 210, 255)},
	}, Font{Size: 26}, 0)
	for _, scale := range []float32{1.25, 1.5} {
		t.Run(fmt.Sprint(scale), func(t *testing.T) {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			pixels, offset := layout.Rasterize(RGB(255, 255, 255), scale)
			if pixels == nil || textRasterInk(pixels).Empty() {
				t.Fatal("native glyph raster produced no pixels for the D3D11 upload")
			}
			var retained DecorativeRaster
			tester := coreNewTester(func(c *context) {
				c.Root().Background(RGB(26, 31, 42))
				coreColumn(c).Padding(20).Children(func() {
					coreBox(c).Width(140).Height(52).Radius(8).Clip().Children(func() {
						coreBox(c).Fill().Opacity(0.65).Draw(func(p *Painter, box Rect) {
							// A non-integral horizontal compression and an image
							// crossing its parent's clip exercise image sampling,
							// inherited opacity and clipping on the real GPU path.
							retained.Paint(p, pixels, Rect{box.X - 8 + offset.X, box.Y + 6 + offset.Y, float32(pixels.Rect.Dx()) / scale * 0.83, float32(pixels.Rect.Dy()) / scale})
						})
					})
				})
			}, 220, 110)
			tester.SetScale(scale)
			display := tester.h.last
			images, clips := 0, 0
			for _, op := range display.Ops {
				switch op.Kind {
				case scene.OpImage:
					images++
					if op.Image == nil || op.Opacity != 0.65 || op.Image.W != pixels.Rect.Dx() || op.Image.H != pixels.Rect.Dy() {
						t.Fatal("native bitmap, inherited opacity or unmodified source did not reach Painter.Image")
					}
				case scene.OpPushClip:
					clips++
				}
			}
			if images != 1 || clips == 0 {
				t.Fatalf("fixture lost its native image/clip path: images=%d clips=%d", images, clips)
			}
			bgra, stride, err := d3d11.RenderOffscreen(display)
			if err != nil {
				t.Fatalf("actual D3D11 image upload/render/readback failed: %v", err)
			}
			gputest.Compare(t, fmt.Sprintf("native-text-image-%g", scale), bgra, stride, display)
		})
	}
}
