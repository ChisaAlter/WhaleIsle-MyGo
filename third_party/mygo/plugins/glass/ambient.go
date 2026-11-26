package glass

import (
	"fmt"
	"math"

	"github.com/egoist/mygo/internal/scene"
	"github.com/egoist/mygo/ui"
)

// AmbientVariant selects the motion and geometry pattern of AmbientBackdrop.
type AmbientVariant uint8

const (
	// AmbientOrbs renders large soft-focus circular orbs drifting vertically,
	// horizontally, and in eccentric circles.
	AmbientOrbs AmbientVariant = iota
	// AmbientAurora renders wide diagonal elliptical aurora bands drifting gently.
	AmbientAurora
	// AmbientChaos renders compact circular blooms pulsing in scale along
	// multi-segment paths.
	AmbientChaos
	// AmbientRays renders slow-rotating soft conic light wedges.
	AmbientRays
)

// AmbientBackdrop is a single-pass GPU animated multi-blob gradient backdrop
// implementing ui.Material with zero backdrop readback or texture uploads.
type AmbientBackdrop struct {
	Variant  AmbientVariant
	BaseFrom ui.Color
	BaseTo   ui.Color
	Colors   [5]ui.Color
	Count    int     // 1..5
	Time     float32 // 驱动动画的等效秒数（已乘 speed）
	Dark     bool
	Opacity  float32 // 默认 <=0 视为 1
}

// PaintMaterial paints the ambient backdrop over box, rounded by radii.
func (b AmbientBackdrop) PaintMaterial(p *ui.Painter, box ui.Rect, radii [4]float32) {
	if box.W <= 0 || box.H <= 0 {
		return
	}
	variant := b.Variant
	if variant > AmbientRays {
		variant = AmbientOrbs
	}
	count := min(max(b.Count, 1), 5)
	opacity := b.Opacity
	if opacity <= 0 {
		opacity = 1
	} else if opacity > 1 {
		opacity = 1
	}
	var dark float32
	if b.Dark {
		dark = 1
	}
	scale := p.Scale()
	if scale <= 0 {
		scale = 1
	}
	baseFrom, baseTo, colors := resolveAmbientColors(b.BaseFrom, b.BaseTo, b.Colors, b.Dark)
	params := [5][4]float32{
		{packRGB(baseFrom), packRGB(baseTo), packRGB(colors[0]), packRGB(colors[1])},
		{packRGB(colors[2]), packRGB(colors[3]), packRGB(colors[4]), float32(variant)},
		{float32(count), b.Time, dark, opacity},
		{scale, 0, 0, 0},
		{},
	}
	p.Effect(ambientEffect, box, radii, 0, params)
}

// String describes the ambient backdrop, as the inspector shows it.
func (b AmbientBackdrop) String() string {
	name := "orbs"
	switch b.Variant {
	case AmbientAurora:
		name = "aurora"
	case AmbientChaos:
		name = "chaos"
	case AmbientRays:
		name = "rays"
	}
	return fmt.Sprintf("ambient(%s, count=%d)", name, min(max(b.Count, 1), 5))
}

var (
	ambientLightDefaults = [7]ui.Color{
		ui.Hex("#edf3fe"), ui.Hex("#dbeafe"),
		ui.Hex("#7aaaff"), ui.Hex("#60a5fa"), ui.Hex("#b7c8fe"), ui.Hex("#4ed17e"), ui.Hex("#f7ad31"),
	}
	ambientDarkDefaults = [7]ui.Color{
		ui.Hex("#283142"), ui.Hex("#172554"),
		ui.Hex("#4176e6"), ui.Hex("#2563eb"), ui.Hex("#5686fe"), ui.Hex("#22c55e"), ui.Hex("#f59e0b"),
	}
)

func resolveAmbientColors(baseFrom, baseTo ui.Color, colors [5]ui.Color, dark bool) (ui.Color, ui.Color, [5]ui.Color) {
	defs := ambientLightDefaults
	if dark {
		defs = ambientDarkDefaults
	}
	if baseFrom.A == 0 {
		baseFrom = defs[0]
	}
	if baseTo.A == 0 {
		baseTo = defs[1]
	}
	for i := range colors {
		if colors[i].A == 0 {
			colors[i] = defs[i+2]
		}
	}
	return baseFrom, baseTo, colors
}

func packRGB(c ui.Color) float32 {
	return float32(uint32(c.R) + uint32(c.G)<<8 + uint32(c.B)<<16)
}

func unpackRGB(v float32) [3]float32 {
	b := floor(v / 65536)
	rem := v - b*65536
	g := floor(rem / 256)
	r := rem - g*256
	return [3]float32{r / 255, g / 255, b / 255}
}

// ignTriangular returns triangular-distributed Interleaved Gradient Noise in
// [-1, 1] for integer pixel coordinates (fx, fy).
func ignTriangular(fx, fy float32) float32 {
	dot := fx*0.06711056 + fy*0.00583715
	inner := dot - floor(dot)
	prod := 52.9829189 * inner
	n := prod - floor(prod)
	r := n*2 - 1
	signR := float32(1)
	if r < 0 {
		signR = -1
	}
	return signR * (1 - sqrt(max(0, 1-abs(r))))
}

const (
	ambientEaseLinear = 0
	ambientEaseCSS    = 1
	ambientEaseInOut  = 2

	ambientMotionCircle = 0
	ambientMotionVert   = 1
	ambientMotionHoriz  = 2
	ambientMotionDrift  = 3
	ambientMotionChaos  = 4
)

func ambientEase(x float32, mode int) float32 {
	if mode == ambientEaseLinear || x <= 0 || x >= 1 {
		return x
	}
	x1, y1, x2, y2 := float32(0.25), float32(0.1), float32(0.25), float32(1.0)
	if mode == ambientEaseInOut {
		x1, y1, x2, y2 = 0.42, 0, 0.58, 1
	}
	bez := func(t, a, b float32) float32 {
		u := 1 - t
		return 3*u*u*t*a + 3*u*t*t*b + t*t*t
	}
	lo, hi := float32(0), float32(1)
	for range 18 {
		mid := 0.5 * (lo + hi)
		if bez(mid, x1, x2) < x {
			lo = mid
		} else {
			hi = mid
		}
	}
	return bez(0.5*(lo+hi), y1, y2)
}

type ambientBlobState struct {
	left, top, bw, bh     float32
	ox, oy                float32
	tx, ty, co, si, scale float32
	opacity, stop, blurPx float32
	conicStart, conicSpan float32
	ellipse, conic        bool
	color                 [3]float32
}

type ambientPixels struct {
	rect     scene.Rect
	baseFrom [3]float32
	baseTo   [3]float32
	blobs    [5]ambientBlobState
	count    int
	keep     float32
	opacity  float32
}

func (px *ambientPixels) Begin(op *scene.EffectOp, rect scene.Rect, _ [4]float32) {
	px.rect = rect
	p := &op.Params
	px.baseFrom = unpackRGB(p[0][0])
	px.baseTo = unpackRGB(p[0][1])
	colors := [5][3]float32{
		unpackRGB(p[0][2]),
		unpackRGB(p[0][3]),
		unpackRGB(p[1][0]),
		unpackRGB(p[1][1]),
		unpackRGB(p[1][2]),
	}
	variant := AmbientVariant(int(floor(p[1][3] + 0.5)))
	px.count = min(max(int(floor(p[2][0]+0.5)), 1), 5)
	seconds := max(float32(0), p[2][1])
	px.keep = 1 - float32(61)/255
	if p[2][2] > 0.5 {
		px.keep = 1 - float32(128)/255
	}
	px.opacity = p[2][3]
	if px.opacity <= 0 {
		px.opacity = 1
	} else if px.opacity > 1 {
		px.opacity = 1
	}
	scale := p[3][0]
	if scale <= 0 {
		scale = 1
	}

	w, h := rect.W, rect.H
	orbsLeft := [5]float32{0.05, -0.02, 0.22, 0.15, 0.18}
	orbsTop := [5]float32{0, 0.15, 0.08, 0.22, -0.05}
	orbsOx := [5]float32{0, -400, 400, -200, -800}
	orbsOy := [5]float32{0, 0, 0, 0, -800}
	orbsDur := [5]float32{30, 20, 40, 40, 20}
	orbsRev := [5]bool{false, true, false, false, false}
	orbsEase := [5]int{ambientEaseCSS, ambientEaseCSS, ambientEaseLinear, ambientEaseCSS, ambientEaseCSS}
	orbsMotion := [5]int{ambientMotionVert, ambientMotionCircle, ambientMotionCircle, ambientMotionHoriz, ambientMotionCircle}

	auroraTop := [5]float32{0.2, 0.37, 0.12, 0.48, 0.3}
	auroraDur := [5]float32{36, 44, 30, 48, 40}
	auroraRev := [5]bool{false, true, false, true, false}

	chaosLeft := [5]float32{0.05, 0.3, 0.12, 0.25, 0.04}
	chaosTop := [5]float32{0, 0.08, 0.32, 0, 0.26}
	chaosDur := [5]float32{14, 18, 11, 16, 13}
	chaosRev := [5]bool{false, true, false, true, false}

	raysDur := [5]float32{30, 40, 24, 36, 28}
	raysRev := [5]bool{false, true, false, true, false}
	raysStart := [5]float32{0, 90, 180, 270, 45}
	raysSpan := [5]float32{80, 70, 100, 60, 90}

	chaosPoints := [5][3]float32{
		{-0.3, -0.35, 1},
		{0.35, 0.05, 1.3},
		{-0.15, 0.4, 0.75},
		{0.25, -0.3, 1.15},
		{-0.3, -0.35, 1},
	}

	for i := range px.count {
		var b ambientBlobState
		b.color = colors[i]
		b.scale = 1
		var dur float32
		var rev bool
		var easeMode, motion int
		switch variant {
		case AmbientAurora:
			b.bw, b.bh = 1.4*w, 0.36*h
			b.left, b.top = -0.2*w, auroraTop[i]*h
			b.ox, b.oy = 0.5*b.bw, 0.5*b.bh
			b.opacity, b.stop, b.ellipse, b.blurPx = 0.55, 0.55, true, 44*scale
			dur, rev, easeMode, motion = auroraDur[i], auroraRev[i], ambientEaseInOut, ambientMotionDrift
		case AmbientChaos:
			b.bw, b.bh = 0.6*w, 0.6*h
			b.left, b.top = chaosLeft[i]*w, chaosTop[i]*h
			b.ox, b.oy = 0.5*b.bw, 0.5*b.bh
			b.opacity, b.stop, b.blurPx = 0.75, 0.55, 30*scale
			dur, rev, easeMode, motion = chaosDur[i], chaosRev[i], ambientEaseInOut, ambientMotionChaos
		case AmbientRays:
			b.bw, b.bh = 1.5*w, 1.5*h
			b.left, b.top = -0.25*w, -0.25*h
			if i == 0 {
				b.left, b.top = 0.05*w, 0
			}
			b.ox, b.oy = 0.5*b.bw, 0.5*b.bh
			b.opacity, b.conic, b.blurPx = 0.45, true, 18*scale
			b.conicStart, b.conicSpan = raysStart[i], raysSpan[i]
			dur, rev, easeMode, motion = raysDur[i], raysRev[i], ambientEaseLinear, ambientMotionCircle
		default:
			b.bw, b.bh = 0.8*w, 0.8*h
			b.left, b.top = orbsLeft[i]*w, orbsTop[i]*h
			b.ox, b.oy = 0.5*b.bw+orbsOx[i]*scale, 0.5*b.bh+orbsOy[i]*scale
			b.opacity = 0.8
			if i == 3 {
				b.opacity = 0.7
			}
			b.stop, b.blurPx = 0.5, 64*scale
			dur, rev, easeMode, motion = orbsDur[i], orbsRev[i], orbsEase[i], orbsMotion[i]
		}
		q := seconds / dur
		progress := q - floor(q)
		if rev {
			progress = 1 - progress
		}
		segments := 2
		if motion == ambientMotionChaos {
			segments = 4
		}
		segFloat := progress * float32(segments)
		segment := min(segments-1, max(0, int(floor(segFloat))))
		local := ambientEase(segFloat-float32(segment), easeMode)
		var rot float32
		switch motion {
		case ambientMotionCircle:
			rot = (float32(segment) + local) * math.Pi
		case ambientMotionVert:
			b.ty = (-0.5 + local) * b.bh
			if segment == 1 {
				b.ty = (0.5 - local) * b.bh
			}
		case ambientMotionHoriz:
			k := -1 + 2*local
			if segment == 1 {
				k = -k
			}
			b.tx, b.ty = k*0.5*b.bw, k*0.1*b.bh
		case ambientMotionDrift:
			k := -1 + 2*local
			if segment == 1 {
				k = -k
			}
			b.tx, b.ty, rot = k*0.45*b.bw, k*0.06*b.bh, k*3*math.Pi/180
		case ambientMotionChaos:
			a, z := chaosPoints[segment], chaosPoints[segment+1]
			b.tx = (a[0] + (z[0]-a[0])*local) * b.bw
			b.ty = (a[1] + (z[1]-a[1])*local) * b.bh
			b.scale = a[2] + (z[2]-a[2])*local
		}
		b.co = float32(math.Cos(float64(rot)))
		b.si = float32(math.Sin(float64(rot)))
		px.blobs[i] = b
	}
}

func smoothstep01(edge0, edge1, x float32) float32 {
	t := min(max((x-edge0)/(edge1-edge0), 0), 1)
	return t * t * (3 - 2*t)
}

func (px *ambientPixels) Color(x, y float32, _ *scene.BackdropImage) [4]float32 {
	w, h := px.rect.W, px.rect.H
	if w <= 0 || h <= 0 {
		return [4]float32{}
	}
	qx, qy := x-px.rect.X, y-px.rect.Y
	const (
		sin40 = float32(0.6427876096865393)
		cos40 = float32(0.766044443118978)
	)
	gradLen := max(abs(w*sin40)+abs(h*(-cos40)), float32(1))
	gradT := min(max(0.5+((qx-0.5*w)*sin40+(qy-0.5*h)*(-cos40))/gradLen, 0), 1)
	col := [3]float32{
		px.baseFrom[0] + (px.baseTo[0]-px.baseFrom[0])*gradT,
		px.baseFrom[1] + (px.baseTo[1]-px.baseFrom[1])*gradT,
		px.baseFrom[2] + (px.baseTo[2]-px.baseFrom[2])*gradT,
	}

	for i := range px.count {
		b := &px.blobs[i]
		dx := qx - b.left - b.tx - b.ox
		dy := qy - b.top - b.ty - b.oy
		lx := (b.co*dx+b.si*dy)/b.scale + b.ox
		ly := (-b.si*dx+b.co*dy)/b.scale + b.oy
		bx, by := lx-0.5*b.bw, ly-0.5*b.bh
		var alpha float32
		if b.conic {
			halfSpan := 0.5 * b.conicSpan
			var deg float32
			if abs(bx)+abs(by) > 1e-5 {
				deg = float32(math.Atan2(float64(bx), float64(-by))) * (180 / math.Pi)
			}
			ang := deg - b.conicStart - halfSpan
			ang -= 360 * floor((ang+180)/360)
			rPx := sqrt(bx*bx + by*by)
			blurDeg := float32(math.Atan2(float64(b.blurPx), float64(max(rPx, 1)))) * (180 / math.Pi)
			angAlpha := 1 - smoothstep01(0, 1, abs(ang)/(halfSpan+1.5*blurDeg))
			rx := bx / max(0.5*b.bw, 1)
			ry := by / max(0.5*b.bh, 1)
			rNorm := sqrt(rx*rx + ry*ry)
			radFade := 1 - smoothstep01(0.55, 1, rNorm)
			alpha = angAlpha * radFade
		} else if b.ellipse {
			rx := bx / max(0.5*b.bw, 1)
			ry := by / max(0.5*b.bh, 1)
			d := sqrt(rx*rx+ry*ry) / 1.41421356
			blurNorm := min(1.5*b.blurPx/max(0.5*min(b.bw, b.bh), 1), 0.45)
			u := d / (b.stop + blurNorm)
			alpha = 1 - smoothstep01(0, 1, u)
		} else {
			halfDiag := max(sqrt(0.25*b.bw*b.bw+0.25*b.bh*b.bh), 1)
			d := sqrt(bx*bx+by*by) / halfDiag
			blurNorm := min(1.5*b.blurPx/halfDiag, 0.45)
			u := d / (b.stop + blurNorm)
			alpha = 1 - smoothstep01(0, 1, u)
		}
		alpha *= b.opacity
		if alpha > 0 {
			for ch := range 3 {
				a := col[ch]
				s := b.color[ch]
				var hl float32
				if s < 0.5 {
					hl = 2 * a * s
				} else {
					hl = 1 - 2*(1-a)*(1-s)
				}
				col[ch] = a + (hl-a)*alpha
			}
		}
	}

	dither := ignTriangular(floor(x), floor(y)) * (0.5 / 255)
	op := px.opacity
	for ch := range 3 {
		v := min(max(col[ch]*px.keep+dither, 0), 1)
		col[ch] = v * op
	}
	return [4]float32{col[0], col[1], col[2], op}
}
