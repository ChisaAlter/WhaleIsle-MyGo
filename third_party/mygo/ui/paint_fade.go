package ui

import "math"

// The mask uses disjoint physical-pixel strips only at the edges. All ordinary
// UI/scene primitives, including color glyphs and images, already multiply the
// painter's opacity. Reusing that path needs no renderer- or backend-specific
// mask operation, and leaves transparent material behind the drawing intact.
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
	a, z := math.Max(left, clipLeft), math.Min(right, clipRight)
	if z <= a || p.clip.H <= 0 {
		return
	}
	leftEnd := math.Min(right, left+math.Ceil(fadeLeft))
	rightStart := math.Max(leftEnd, right-math.Ceil(fadeRight))
	paint := func(x0, x1, alpha float64) {
		x0, x1 = math.Max(a, x0), math.Min(z, x1)
		if x1 <= x0 || alpha <= 0 {
			return
		}
		saved := p.opacity
		p.opacity *= float32(math.Min(alpha, 1))
		p.Clip(Rect{X: float32(x0) / p.scale, Y: p.clip.Y, W: float32(x1-x0) / p.scale, H: p.clip.H}, 0, func() {
			p.elementBody(e)
		})
		p.opacity = saved
	}
	alphaAt := func(x float64) float64 {
		alpha := float64(1)
		if fadeLeft > 0 {
			alpha = math.Min(alpha, (x-left)/fadeLeft)
		}
		if fadeRight > 0 {
			alpha = math.Min(alpha, (right-x)/fadeRight)
		}
		return math.Max(0, alpha)
	}
	for x := a; x < math.Min(z, leftEnd); x++ {
		paint(x, x+1, alphaAt(x+.5))
	}
	paint(leftEnd, rightStart, 1)
	for x := math.Max(a, rightStart); x < z; x++ {
		paint(x, x+1, alphaAt(x+.5))
	}
}
