package ui

import (
	"math"

	"github.com/egoist/mygo/internal/scene"
)

// ClipFade restricts what fn draws to r and linearly fades alpha from 0 at
// r's left and right edges to 1 inward over fadeLeft and fadeRight DIPs.
// Zero, negative or nonfinite fade widths disable that edge. Fades wider than
// r meet proportionally in its middle.
func (p *Painter) ClipFade(r Rect, fadeLeft, fadeRight float32, fn func()) {
	if fn == nil || p.scale <= 0 || math.IsNaN(float64(p.scale)) || math.IsInf(float64(p.scale), 0) {
		return
	}
	if r.W <= 0 || r.H <= 0 ||
		math.IsNaN(float64(r.X)) || math.IsInf(float64(r.X), 0) ||
		math.IsNaN(float64(r.Y)) || math.IsInf(float64(r.Y), 0) ||
		math.IsNaN(float64(r.W)) || math.IsInf(float64(r.W), 0) ||
		math.IsNaN(float64(r.H)) || math.IsInf(float64(r.H), 0) {
		return
	}
	d := p.snap(r)
	if d.W <= 0 || d.H <= 0 {
		return
	}
	fl := float64(fadeWidth(fadeLeft)) * float64(p.scale)
	fr := float64(fadeWidth(fadeRight)) * float64(p.scale)
	if fl+fr > float64(d.W) {
		ratio := float64(d.W) / (fl + fr)
		fl, fr = fl*ratio, fr*ratio
	}
	savedClip := p.clip
	p.clip = intersect(p.clip, Rect{X: d.X / p.scale, Y: d.Y / p.scale, W: d.W / p.scale, H: d.H / p.scale})
	if p.clip.W <= 0 || p.clip.H <= 0 {
		p.clip = savedClip
		return
	}
	p.s.Ops = append(p.s.Ops, scene.Op{
		Kind: scene.OpPushClip,
		Rect: d,
		Fade: [2]float32{float32(fl), float32(fr)},
	})
	fn()
	p.s.Ops = append(p.s.Ops, scene.Op{Kind: scene.OpPopClip})
	p.clip = savedClip
}

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
	insetL, insetR := e.fadeInset[0], e.fadeInset[1]
	// Match the renderer's physical-pixel clip rounding at fractional DPI.
	left := float64(round((e.x + insetL) * p.scale))
	right := float64(round((e.x + e.w - insetR) * p.scale))
	width := right - left
	if width <= 0 {
		return
	}
	fadeL, fadeR := e.fade[0], e.fade[1]
	if e.fadeOverflow && e.kind == kindText && e.tl != nil && !e.tl.Truncated {
		maxRight := float32(0)
		cx := e.contentX()
		for i := range e.tl.Lines {
			line := &e.tl.Lines[i]
			if r := cx + line.X + line.Width; r > maxRight {
				maxRight = r
			}
		}
		if maxRight <= e.w-insetR {
			fadeR = 0
		}
	}
	fadeLeft, fadeRight := float64(fadeL)*float64(p.scale), float64(fadeR)*float64(p.scale)
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
