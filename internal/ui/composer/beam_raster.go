package composer

import (
	"image"
	"math"

	"github.com/egoist/mygo/ui"
)

type beamRGBA [4]float32 // premultiplied, sRGB
type beamPixel struct {
	angle, outer, inner, ring, bloom float32
	wash                             beamRGBA
}
type beamRaster struct {
	beamRasterData
	texture      ui.DecorativeRaster
	frames       chan *beamRasterData
	pending      bool
	front, spare *image.RGBA
}
type beamRasterData struct {
	width, height      int
	w, h, scale, track float32
	palette            []ui.Color
	legacy             bool
	geometry           []beamPixel
	pixels             *image.RGBA
	bloom, scratch     []beamRGBA
	blurSteps          []beamRGBA
	gaussian           beamGaussian
}

func beamOver(front, back beamRGBA) (out beamRGBA) {
	for i := range out {
		out[i] = front[i] + back[i]*(1-front[3])
	}
	return
}
func beamInk(c ui.Color, alpha float32) beamRGBA {
	return beamRGBA{float32(c.R) / 255 * alpha, float32(c.G) / 255 * alpha, float32(c.B) / 255 * alpha, alpha}
}
func beamMultiply(c beamRGBA, alpha float32) beamRGBA {
	for i := range c {
		c[i] *= alpha
	}
	return c
}

func beamDistance(x, y, w, h, r float32) float32 {
	r = min(r, w/2, h/2)
	a, b := float32(math.Abs(float64(x-w/2)))-w/2+r, float32(math.Abs(float64(y-h/2)))-h/2+r
	return float32(math.Hypot(float64(max(a, 0)), float64(max(b, 0)))) + min(max(a, b), 0) - r
}
func beamRoundedCoverage(x, y, w, h, radius, scale float32) float32 {
	if w <= 0 || h <= 0 {
		return 0
	}
	d := beamDistance(x, y, w, h, radius)
	// Only boundary pixels need area sampling. Every point in a pixel lies
	// within sqrt(.5) device pixels of its center.
	if d*scale < -.708 {
		return 1
	}
	if d*scale > .708 {
		return 0
	}
	inside := 0
	for sy := range 4 {
		for sx := range 4 {
			dx, dy := (float32(sx)+.5)/4-.5, (float32(sy)+.5)/4-.5
			if beamDistance(x+dx/scale, y+dy/scale, w, h, radius) <= 0 {
				inside++
			}
		}
	}
	return float32(inside) / 16
}

func (b *beamRasterData) matches(w, h, scale, track float32, colors []ui.Color, legacy bool) bool {
	if b.w != w || b.h != h || b.scale != scale || b.track != track || b.legacy != legacy || len(b.palette) != len(colors) {
		return false
	}
	for i, c := range colors {
		if b.palette[i] != c {
			return false
		}
	}
	return true
}

func (b *beamRasterData) prepare(w, h, scale, track float32, colors []ui.Color, legacy bool) {
	if b.matches(w, h, scale, track, colors, legacy) {
		return
	}
	b.w, b.h, b.scale, b.track, b.legacy = w, h, scale, track, legacy
	b.palette = append(b.palette[:0], colors...)
	b.width, b.height = max(1, int(math.Ceil(float64((w+8)*scale)))), max(1, int(math.Ceil(float64((h+8)*scale))))
	n := b.width * b.height
	b.geometry = make([]beamPixel, n)
	b.bloom = make([]beamRGBA, n)
	b.scratch = make([]beamRGBA, n)
	b.blurSteps = make([]beamRGBA, max(b.width, b.height))
	b.pixels = image.NewRGBA(image.Rect(0, 0, b.width, b.height))
	radials := [4][5]float32{{.74, 1, 180, 32, 107.0 / 255}, {.94, 0, 74, 32, 97.0 / 255}, {.06, 0, 80, 40, 102.0 / 255}, {.20, 0, 90, 45, 92.0 / 255}}
	inks := [4]ui.Color{colors[0], colors[1%len(colors)], colors[2%len(colors)], colors[3%len(colors)]}
	shadow := [2]ui.Color{colors[0], colors[len(colors)-1]}
	shadowAlpha := [2]float32{102.0 / 255, 51.0 / 255}
	if legacy {
		inks = [4]ui.Color{ui.RGB(100, 70, 255), ui.RGB(255, 120, 40), ui.RGB(40, 140, 255), ui.RGB(255, 50, 100)}
		for i, a := range []float32{.42, .38, .4, .36} {
			radials[i][4] = a
		}
		shadow = [2]ui.Color{inks[0], inks[3]}
		shadowAlpha = [2]float32{.28, .1}
	}
	for y := 0; y < b.height; y++ {
		for x := 0; x < b.width; x++ {
			px, py := (float32(x)+.5)/scale-4, (float32(y)+.5)/scale-4
			g := &b.geometry[y*b.width+x]
			g.angle = float32(math.Atan2(float64(px-w/2), -float64(py-h/2)) / (2 * math.Pi))
			g.inner = beamRoundedCoverage(px, py, w, h, 22, scale)
			g.outer = beamRoundedCoverage(px+4, py+4, w+8, h+8, 26, scale)
			g.ring = max(0, g.inner-beamRoundedCoverage(px-track, py-track, w-2*track, h-2*track, max(0, 22-track), scale))
			g.bloom = max(0, g.inner-beamRoundedCoverage(px-1.5, py-1.5, w-3, h-3, 20.5, scale))
			var wash beamRGBA
			for i := 3; i >= 0; i-- {
				rad := radials[i]
				distance := float32(math.Hypot(float64((px-rad[0]*w)/rad[2]), float64((py-rad[1]*h)/rad[3])))
				wash = beamOver(beamInk(inks[i], rad[4]*max(0, 1-distance)), wash)
			}
			g.wash = wash
		}
	}
	// CSS paints the first inset shadow above the second, both above the
	// radial backgrounds. Blur the actual rounded hole in two dimensions;
	// a one-dimensional distance integral loses the other edges and corners.
	b.insetShadow(shadow[1], shadowAlpha[1], 10, 21)
	b.insetShadow(shadow[0], shadowAlpha[0], 3, 9)
}

func (b *beamRasterData) insetShadow(ink ui.Color, opacity, spread, sigma float32) {
	for y := 0; y < b.height; y++ {
		for x := 0; x < b.width; x++ {
			px, py := (float32(x)+.5)/b.scale-4-spread, (float32(y)+.5)/b.scale-4-spread
			coverage := beamRoundedCoverage(px, py, b.w-2*spread, b.h-2*spread, max(0, 22-spread), b.scale)
			b.bloom[y*b.width+x] = beamRGBA{0, 0, 0, coverage}
		}
	}
	b.blur(float64(sigma * b.scale))
	for i := range b.geometry {
		alpha := (1 - min(1, b.bloom[i][3])) * opacity
		b.geometry[i].wash = beamOver(beamInk(ink, alpha), b.geometry[i].wash)
	}
}

var beamInnerMaskA = [][2]float64{{0, 0}, {.1, 0}, {.18, .08}, {.3, .25}, {.44, .7}, {.56, 1}, {.72, 1}, {.8, .65}, {.84, .2}, {.88, 0}, {1, 0}}
var beamInnerMaskB = [][2]float64{{0, 0}, {.4, 0}, {.5, .15}, {.62, .5}, {.72, 1}, {.82, 1}, {.88, .3}, {.94, 0}, {1, 0}}
var beamHighlight = [][2]float64{{0, 0}, {.54, 0}, {.58, .06}, {.64, .5}, {.7, .15}, {.76, 0}, {1, 0}}

func beamStops(t float64, stops [][2]float64) float32 {
	for i := 1; i < len(stops); i++ {
		if t <= stops[i][0] {
			a, z := stops[i-1], stops[i]
			return float32(a[1] + (z[1]-a[1])*(t-a[0])/(z[0]-a[0]))
		}
	}
	return 0
}

func beamFiltered(c beamRGBA, matrix [9]float64, enabled bool, static bool) beamRGBA {
	if c[3] <= 0 {
		return c
	}
	if !enabled && !static {
		return c
	}
	a := c[3]
	r, g, blue := float64(c[0]/a), float64(c[1]/a), float64(c[2]/a)
	if enabled {
		r, g, blue = matrix[0]*r+matrix[1]*g+matrix[2]*blue, matrix[3]*r+matrix[4]*g+matrix[5]*blue, matrix[6]*r+matrix[7]*g+matrix[8]*blue
	}
	// The static filters still brighten/saturate when the hue animation is off.
	r, g, blue = min(1, max(0, r)*1.3), min(1, max(0, g)*1.3), min(1, max(0, blue)*1.3)
	luma := .213*r + .715*g + .072*blue
	return beamRGBA{float32(min(1, max(0, luma+(r-luma)*1.2))) * a, float32(min(1, max(0, luma+(g-luma)*1.2))) * a, float32(min(1, max(0, luma+(blue-luma)*1.2))) * a, a}
}

func (b *beamRaster) paint(p *ui.Painter, r ui.Rect, style beamStyle, colors []ui.Color, intervals int, phase, hue float64, hueEnabled bool, opacity beamOpacities) {
	if r.W <= 0 || r.H <= 0 {
		return
	}
	scale := p.Scale()
	legacy := style.Palette.Kind != "custom" && style.Palette.ID == "legacy"
	if b.pending {
		select {
		case frame := <-b.frames:
			b.pending = false
			b.beamRasterData = *frame
			if frame.matches(r.W, r.H, scale, style.TrackWidth, colors, legacy) {
				b.spare, b.front = b.front, frame.pixels
			} else {
				// An obsolete size/palette may be reused as worker storage, but
				// cannot replace the native seat's last complete image.
				b.spare = frame.pixels
			}
		default:
		}
	}
	if !b.pending {
		if b.frames == nil {
			b.frames = make(chan *beamRasterData, 1)
		}
		result := b.frames
		palette := append([]ui.Color(nil), colors...)
		state := b.beamRasterData
		// The worker never writes the RGBA currently consumed by native paint.
		// Two buffers are exchanged after completion, without a per-frame copy.
		state.pixels = b.spare
		b.spare = nil
		b.pending = true
		go func() {
			state.render(r.W, r.H, scale, style, palette, intervals, phase, hue, hueEnabled, opacity)
			result <- &state
		}()
	}
	if b.front != nil {
		b.texture.Paint(p, b.front, ui.Rect{X: r.X - 4, Y: r.Y - 4, W: r.W + 8, H: r.H + 8})
	}
	// One finite off-UI image job, one buffered result, no persistent worker or
	// queued historical frames. A detached beam leaves no waiting goroutine.
	p.AnimationFrame()
}

func (b *beamRasterData) render(w, h, scale float32, style beamStyle, colors []ui.Color, intervals int, phase, hue float64, hueEnabled bool, opacity beamOpacities) {
	b.prepare(w, h, scale, style.TrackWidth, colors, style.Palette.Kind != "custom" && style.Palette.ID == "legacy")
	if b.pixels == nil || b.pixels.Rect.Dx() != b.width || b.pixels.Rect.Dy() != b.height {
		b.pixels = image.NewRGBA(image.Rect(0, 0, b.width, b.height))
	}
	matrix := beamHueMatrix(hue)
	// Each conic gradient has its own box: bloom's background is on the
	// inset child, so all three centers coincide at the actual card center.
	const samples = 2048
	var strokes, inners, blooms [samples]beamRGBA
	first, second := colors[0], colors[1%len(colors)]
	if b.legacy {
		first, second = ui.RGB(255, 80, 180), ui.RGB(80, 180, 255)
	}
	for i := 0; i < samples; i++ {
		t := float64(i) / samples
		at := t * float64(intervals)
		index := int(at)
		a, z := colors[index], colors[(index+1)%len(colors)]
		mix := float32(at - float64(index))
		color := beamRGBA{(float32(a.R)*(1-mix) + float32(z.R)*mix) / 255, (float32(a.G)*(1-mix) + float32(z.G)*mix) / 255, (float32(a.B)*(1-mix) + float32(z.B)*mix) / 255, 1}
		color = beamOver(beamRGBA{beamStops(t, beamHighlight), beamStops(t, beamHighlight), beamStops(t, beamHighlight), beamStops(t, beamHighlight)}, color)
		strokes[i] = beamMultiply(beamFiltered(color, matrix, hueEnabled, false), float32(beamMask(t))*float32(opacity.stroke))
		aMask, bMask := beamStops(t, beamInnerMaskA), beamStops(t, beamInnerMaskB)
		inners[i][3] = (aMask + bMask - aMask*bMask) * float32(opacity.inner)
		stops := []struct {
			t float64
			c beamRGBA
		}{{0, beamRGBA{}}, {.58, beamRGBA{}}, {.64, beamRGBA{.08, .08, .08, .08}}, {.69, beamInk(first, 140.0/255)}, {.705, beamInk(second, 179.0/255)}, {.73, beamInk(first, 115.0/255)}, {.82, beamRGBA{}}, {1, beamRGBA{}}}
		if b.legacy {
			stops[3].c = beamInk(first, .55)
			stops[4].c = beamInk(second, .7)
			stops[5].c = beamInk(first, .45)
		}
		for j := 1; j < len(stops); j++ {
			if t <= stops[j].t {
				a, z := stops[j-1], stops[j]
				f := float32((t - a.t) / (z.t - a.t))
				for k := range color {
					color[k] = a.c[k] + (z.c[k]-a.c[k])*f
				}
				break
			}
		}
		// CSS bloom filter order is blur, hue-rotate, brightness, saturation.
		blooms[i] = color
	}
	for i, g := range b.geometry {
		at := int(math.Mod(float64(g.angle)-phase+2, 1) * samples)
		b.bloom[i] = beamMultiply(blooms[at], g.bloom)
	}
	if style.GlowBlur > 0 {
		b.blur(float64(style.GlowBlur * b.scale))
	}
	for i, g := range b.geometry {
		at := int(math.Mod(float64(g.angle)-phase+2, 1) * samples)
		inner := beamMultiply(beamFiltered(g.wash, matrix, hueEnabled, false), g.inner*inners[at][3])
		bloom := beamMultiply(beamFiltered(b.bloom[i], matrix, hueEnabled, true), float32(opacity.bloom))
		color := beamMultiply(beamOver(beamMultiply(strokes[at], g.ring), beamOver(bloom, inner)), g.outer)
		for channel := range color {
			b.pixels.Pix[i*4+channel] = uint8(min(1, max(0, color[channel]))*255 + .5)
		}
	}
}
