package chrome

import "math"

// Per-frame affine/radial coefficients: transforms and trig are evaluated once
// per bloom. This samples the same CSS equations as gradientBlob.at.
type preparedGradientBlob struct {
	b                      gradientBlob
	ax, ay, bx, by, cx, cy float64
	radialX, radialY       float64
}

func prepareGradientBlob(b gradientBlob) preparedGradientBlob {
	co, si := math.Cos(b.rotation)/b.scale, math.Sin(b.rotation)/b.scale
	x, y := b.left+b.tx+b.ox, b.top+b.ty+b.oy
	p := preparedGradientBlob{b: b, ax: co, ay: si, bx: -si, by: co}
	p.cx, p.cy = b.ox-co*x-si*y, b.oy+si*x-co*y
	if b.ellipse {
		p.radialX = 2 / (b.width * math.Sqrt2 * b.stop)
		p.radialY = 2 / (b.height * math.Sqrt2 * b.stop)
	} else {
		r := 1 / (math.Sqrt(b.width*b.width+b.height*b.height) * .5 * b.stop)
		p.radialX, p.radialY = r, r
	}
	return p
}
func (p preparedGradientBlob) sample(x, y float64) gradientColor {
	b := p.b
	lx, ly := p.ax*x+p.ay*y+p.cx, p.bx*x+p.by*y+p.cy
	if lx < 0 || ly < 0 || lx > b.width || ly > b.height {
		return gradientColor{}
	}
	dx, dy := lx-b.width*.5, ly-b.height*.5
	alpha := 0.0
	if b.conic {
		angle := math.Mod(math.Atan2(dx, -dy)*180/math.Pi-b.conicStart+720, 360)
		alpha = math.Max(0, 1-math.Abs(angle-b.conicSpan/2)/(b.conicSpan/2))
	} else {
		dx *= p.radialX
		dy *= p.radialY
		distance2 := dx*dx + dy*dy
		if distance2 >= 1 {
			return gradientColor{}
		}
		alpha = 1 - math.Sqrt(distance2)
	}
	color := b.color
	color.a *= alpha * b.opacity
	return color
}
