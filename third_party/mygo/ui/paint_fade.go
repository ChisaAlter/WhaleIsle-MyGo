package ui

import (
	"math"

	"github.com/egoist/mygo/internal/scene"
)

// fadedElement paints e once under a single horizontal-fade clip operation
// instead of slicing the edges into ~25 1px clip strips. Both the GPU shaders
// and the CPU software rasterizer apply the linear edge alpha mask per pixel
// across fills, glyphs (including ClearType and color emoji), images and
// effects, preserving transparent/checkerboard/glass backgrounds with a single
// draw call and GPU batch.
func (p *Painter) fadedElement(e *node) {
	if e.flags&flagInvisible != 0 || p.scale <= 0 || math.IsNaN(float64(p.scale)) || math.IsInf(float64(p.scale), 0) {
		return
	}
	if e.w <= 0 || math.IsNaN(float64(e.x)) || math.IsInf(float64(e.x), 0) || math.IsNaN(float64(e.w)) || math.IsInf(float64(e.w), 0) {
		return
	}
	// Match the renderer's physical-pixel clip rounding at fractional DPI.
	left := float64(round(e.x * p.scale))
	right := float64(round((e.x + e.w) * p.scale))
	width := right - left
	if width <= 0 {
		return
	}
	fadeLeft, fadeRight := float64(e.fade[0])*float64(p.scale), float64(e.fade[1])*float64(p.scale)
	if fadeLeft+fadeRight > width {
		ratio := width / (fadeLeft + fadeRight)
		fadeLeft, fadeRight = fadeLeft*ratio, fadeRight*ratio
	}
	clipLeft := float64(round(p.clip.X * p.scale))
	clipRight := float64(round((p.clip.X + p.clip.W) * p.scale))
	if math.Min(right, clipRight) <= math.Max(left, clipLeft) || p.clip.H <= 0 {
		return
	}
	y0 := round(p.clip.Y * p.scale)
	y1 := round((p.clip.Y + p.clip.H) * p.scale)
	if y1 <= y0 {
		return
	}
	savedClip := p.clip
	p.clip = intersect(p.clip, Rect{X: float32(left) / p.scale, Y: p.clip.Y, W: float32(width) / p.scale, H: p.clip.H})
	p.s.Ops = append(p.s.Ops, scene.Op{
		Kind: scene.OpPushClip,
		Rect: scene.Rect{X: float32(left), Y: y0, W: float32(width), H: y1 - y0},
		Fade: [2]float32{float32(fadeLeft), float32(fadeRight)},
	})
	p.elementBody(e)
	p.s.Ops = append(p.s.Ops, scene.Op{Kind: scene.OpPopClip})
	p.clip = savedClip
}
