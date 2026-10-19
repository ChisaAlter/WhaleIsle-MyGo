package composer

import (
	"image"
	"math"

	"github.com/egoist/mygo/ui"
)

type beamRGBA [4]float32 // premultiplied, sRGB
type beamPixel struct {
	angle, distance, outer, inner, ring, bloom float32
	wash                                       beamRGBA
}
type beamRaster struct {
	width, height      int
	w, h, scale, track float32
	palette            []ui.Color
	legacy             bool
	geometry           []beamPixel
	pixels             *image.RGBA
	bloom, scratch     []beamRGBA
	texture            ui.DecorativeRaster
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
func beamCoverage(distance, scale float32) float32 { return min(1, max(0, .5-distance*scale)) }

func (b *beamRaster) prepare(w, h, scale, track float32, colors []ui.Color, legacy bool) {
	changed := b.w != w || b.h != h || b.scale != scale || b.track != track || b.legacy != legacy || len(b.palette) != len(colors)
	if !changed {
		for i, c := range colors {
			if b.palette[i] != c {
				changed = true
				break
			}
		}
	}
	if !changed {
		return
	}
	b.w, b.h, b.scale, b.track, b.legacy = w, h, scale, track, legacy
	b.palette = append(b.palette[:0], colors...)
	b.width, b.height = max(1, int(math.Ceil(float64((w+8)*scale)))), max(1, int(math.Ceil(float64((h+8)*scale))))
	n := b.width * b.height
	b.geometry = make([]beamPixel, n)
	b.bloom = make([]beamRGBA, n)
	b.scratch = make([]beamRGBA, n)
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
			d := beamDistance(px, py, w, h, 22)
			g := &b.geometry[y*b.width+x]
			g.angle = float32(math.Atan2(float64(px-w/2), -float64(py-h/2)) / (2 * math.Pi))
			g.distance = d
			g.inner = beamCoverage(d, scale)
			g.outer = beamCoverage(beamDistance(px+4, py+4, w+8, h+8, 26), scale)
			g.ring = max(0, g.inner-beamCoverage(beamDistance(px-track, py-track, w-2*track, h-2*track, max(0, 22-track)), scale))
			g.bloom = max(0, g.inner-beamCoverage(beamDistance(px-1.5, py-1.5, w-3, h-3, 20.5), scale))
			var wash beamRGBA
			for i := 3; i >= 0; i-- {
				rad := radials[i]
				distance := float32(math.Hypot(float64((px-rad[0]*w)/rad[2]), float64((py-rad[1]*h)/rad[3])))
				wash = beamOver(beamInk(inks[i], rad[4]*max(0, 1-distance)), wash)
			}
			// CSS inset shadows lie above the radial backgrounds. The Gaussian
			// edge integral reproduces their spread/blur falloff along each edge.
			for i := 1; i >= 0; i-- {
				spread, sigma := float32(3), float32(9)
				if i == 1 {
					spread, sigma = 10, 21
				}
				alpha := float32(.5*math.Erfc(float64((-d-spread)/(sigma*math.Sqrt2)))) * shadowAlpha[i]
				wash = beamOver(beamInk(shadow[i], alpha), wash)
			}
			g.wash = wash
		}
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
	b.prepare(r.W, r.H, p.Scale(), style.TrackWidth, colors, style.Palette.Kind != "custom" && style.Palette.ID == "legacy")
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
	box := ui.Rect{X: r.X - 4, Y: r.Y - 4, W: r.W + 8, H: r.H + 8}
	b.texture.Paint(p, b.pixels, box)
}

// Three running-sum box passes approximate the CSS Gaussian without drawing
// extra wide strokes. The independent bloom remains visible at blur zero.
func (b *beamRaster) blur(sigma float64) {
	ideal := math.Sqrt(4*sigma*sigma + 1)
	low := int(math.Floor(ideal))
	if low%2 == 0 {
		low--
	}
	low = max(1, low)
	high := low + 2
	m := int(math.Round((12*sigma*sigma - 3*float64(low*low) - 12*float64(low) - 9) / (-4*float64(low) - 4)))
	for pass := 0; pass < 3; pass++ {
		size := high
		if pass < m {
			size = low
		}
		radius := (size - 1) / 2
		for axis := 0; axis < 2; axis++ {
			lines, length := b.height, b.width
			if axis == 1 {
				lines, length = b.width, b.height
			}
			index := func(line, pos int) int {
				if axis == 0 {
					return line*b.width + pos
				}
				return pos*b.width + line
			}
			for line := 0; line < lines; line++ {
				var sum beamRGBA
				for k := 0; k <= radius && k < length; k++ {
					v := b.bloom[index(line, k)]
					for c := range sum {
						sum[c] += v[c]
					}
				}
				for pos := 0; pos < length; pos++ {
					out := index(line, pos)
					for c := range sum {
						b.scratch[out][c] = sum[c] / float32(size)
					}
					if remove := pos - radius; remove >= 0 {
						v := b.bloom[index(line, remove)]
						for c := range sum {
							sum[c] -= v[c]
						}
					}
					if add := pos + radius + 1; add < length {
						v := b.bloom[index(line, add)]
						for c := range sum {
							sum[c] += v[c]
						}
					}
				}
			}
			b.bloom, b.scratch = b.scratch, b.bloom
		}
	}
}
