package chrome

// Source: installed Whale Isle 0.3.10 ui-theme rc.2, lib/client.js:7318.
// The verbatim CSS and bundle hash are in testdata/installed-gradient-*.
// Coordinates, durations, easing and blend equations are CSS values, in DIPs.
import (
	"math"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/wallpapers"
)

type gradientColor struct{ r, g, b, a float64 }
type gradientConfig struct {
	width, height float64
	colors        [7]gradientColor
	variant       string
	speed         float64
	count         int
	dark, reduced bool
}

func gradientConfiguration(p wallpapers.Preferences, width, height float64, dark, reduced bool) gradientConfig {
	c := gradientConfig{width: width, height: height, variant: p.Variant, dark: dark, reduced: reduced}
	switch c.variant {
	case "aurora", "chaos", "rays":
	default:
		c.variant = "orbs"
	}
	c.speed = p.Speed
	if math.IsNaN(c.speed) || math.IsInf(c.speed, 0) {
		c.speed = 190
	}
	// CSS clampBackgroundEffectSpeed: zero is 20%, not a paused animation.
	c.speed = math.Min(300, math.Max(20, math.Round(c.speed)))
	count := p.Count
	if math.IsNaN(count) || math.IsInf(count, 0) {
		count = 5
	}
	c.count = int(math.Min(5, math.Max(1, math.Round(count))))
	defaults := [7]string{"#edf3fe", "#dbeafe", "#7aaaff", "#60a5fa", "#b7c8fe", "#4ed17e", "#f7ad31"}
	if dark {
		defaults = [7]string{"#283142", "#172554", "#4176e6", "#2563eb", "#5686fe", "#22c55e", "#f59e0b"}
	}
	colors := p.Colors
	if colors == [7]string{} && p.Preset != "" {
		colors = wallpapers.PresetColors(p.Preset)
	}
	// Presets and individual overrides retain the same seven-slot contract.
	for i := range c.colors {
		raw := defaults[i]
		if gradientHex(colors[i]) {
			raw = strings.TrimSpace(colors[i])
		}
		v := ui.Hex(raw)
		c.colors[i] = gradientColor{float64(v.R) / 255, float64(v.G) / 255, float64(v.B) / 255, 1}
	}
	return c
}
func gradientHex(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) != 7 || s[0] != '#' {
		return false
	}
	for _, r := range s[1:] {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F') {
			return false
		}
	}
	return true
}
func gradientFinite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

// CSS cubic-bezier timing. Solve x(t), then evaluate y(t); a keyframe's
// easing applies separately to each interval, including the 50% circle stop.
func gradientEase(x float64, mode string) float64 {
	if mode == "linear" || x <= 0 || x >= 1 {
		return x
	}
	x1, y1, x2, y2 := .25, .1, .25, 1.0
	if mode == "ease-in-out" {
		x1, y1, x2, y2 = .42, 0, .58, 1
	}
	bez := func(t, a, b float64) float64 { return 3*(1-t)*(1-t)*t*a + 3*(1-t)*t*t*b + t*t*t }
	lo, hi := 0.0, 1.0
	for range 22 {
		mid := (lo + hi) / 2
		if bez(mid, x1, x2) < x {
			lo = mid
		} else {
			hi = mid
		}
	}
	return bez((lo+hi)/2, y1, y2)
}

type gradientBlob struct {
	left, top, width, height, ox, oy     float64
	tx, ty, rotation, scale              float64
	opacity, stop, conicStart, conicSpan float64
	ellipse, conic                       bool
	color                                gradientColor
}

func (c gradientConfig) blobs(seconds float64) [5]gradientBlob {
	var out [5]gradientBlob
	left := [5]float64{.05, -.02, .22, .15, .18}
	top := [5]float64{0, .15, .08, .22, -.05}
	durations := [5]float64{30, 20, 40, 40, 20}
	reverse := [5]bool{false, true, false, false, false}
	ease := [5]string{"ease", "ease", "linear", "ease", "ease"}
	motion := [5]string{"vertical", "circle", "circle", "horizontal", "circle"}
	for i := range out {
		b := gradientBlob{left: left[i] * c.width, top: top[i] * c.height, width: .8 * c.width, height: .8 * c.height, scale: 1, opacity: .8, stop: .5, color: c.colors[i+2]}
		if i == 3 {
			b.opacity = .7
		}
		b.ox, b.oy = b.width/2, b.height/2
		if i == 1 {
			b.ox -= 400
		}
		if i == 2 {
			b.ox += 400
		}
		if i == 3 {
			b.ox -= 200
		}
		if i == 4 {
			b.ox -= 800
			b.oy -= 800
		}
		switch c.variant {
		case "aurora":
			b.width, b.height = 1.4*c.width, .36*c.height
			b.left = -.2 * c.width
			b.top = ([5]float64{.2, .37, .12, .48, .3})[i] * c.height
			b.ox, b.oy = b.width/2, b.height/2
			b.opacity, b.stop, b.ellipse = .55, .55, true
			durations = [5]float64{36, 44, 30, 48, 40}
			reverse = [5]bool{false, true, false, true, false}
			motion[i], ease[i] = "drift", "ease-in-out"
		case "chaos":
			b.width, b.height = .6*c.width, .6*c.height
			b.left = ([5]float64{.05, .3, .12, .25, .04})[i] * c.width
			b.top = ([5]float64{0, .08, .32, 0, .26})[i] * c.height
			b.ox, b.oy = b.width/2, b.height/2
			b.opacity, b.stop = .75, .55
			durations = [5]float64{14, 18, 11, 16, 13}
			reverse = [5]bool{false, true, false, true, false}
			motion[i], ease[i] = "chaos", "ease-in-out"
		case "rays":
			b.width, b.height = 1.5*c.width, 1.5*c.height
			b.left, b.top = -.25*c.width, -.25*c.height
			if i == 0 {
				b.left, b.top = .05*c.width, 0
			}
			b.ox, b.oy = b.width/2, b.height/2
			b.opacity, b.conic = .45, true
			b.conicStart = ([5]float64{0, 90, 180, 270, 45})[i]
			b.conicSpan = ([5]float64{80, 70, 100, 60, 90})[i]
			durations = [5]float64{30, 40, 24, 36, 28}
			reverse = [5]bool{false, true, false, true, false}
			motion[i], ease[i] = "circle", "linear"
		}
		// animation:none removes transform, rather than evaluating the first keyframe.
		if !c.reduced {
			progress := math.Mod(math.Max(0, seconds)*c.speed/100/durations[i], 1)
			if reverse[i] {
				progress = 1 - progress
			}
			segments := 2
			if motion[i] == "chaos" {
				segments = 4
			}
			segment := min(segments-1, int(progress*float64(segments)))
			local := gradientEase(progress*float64(segments)-float64(segment), ease[i])
			switch motion[i] {
			case "circle":
				b.rotation = (float64(segment) + local) * math.Pi
			case "vertical":
				b.ty = (-.5 + local) * b.height
				if segment == 1 {
					b.ty = (.5 - local) * b.height
				}
			case "horizontal":
				k := -1 + 2*local
				if segment == 1 {
					k = -k
				}
				b.tx, b.ty = k*.5*b.width, k*.1*b.height
			case "drift":
				k := -1 + 2*local
				if segment == 1 {
					k = -k
				}
				b.tx, b.ty, b.rotation = k*.45*b.width, k*.06*b.height, k*3*math.Pi/180
			case "chaos":
				points := [5][3]float64{{-.3, -.35, 1}, {.35, .05, 1.3}, {-.15, .4, .75}, {.25, -.3, 1.15}, {-.3, -.35, 1}}
				a, z := points[segment], points[segment+1]
				b.tx = (a[0] + (z[0]-a[0])*local) * b.width
				b.ty = (a[1] + (z[1]-a[1])*local) * b.height
				b.scale = a[2] + (z[2]-a[2])*local
			}
		}
		out[i] = b
	}
	return out
}
func (b gradientBlob) at(x, y float64) gradientColor {
	// Invert CSS translate() rotate()/scale() about transform-origin. Translation
	// is outside rotation/scale in the authored transform lists.
	dx, dy := x-b.left-b.tx-b.ox, y-b.top-b.ty-b.oy
	co, si := math.Cos(b.rotation), math.Sin(b.rotation)
	lx, ly := (co*dx+si*dy)/b.scale+b.ox, (-si*dx+co*dy)/b.scale+b.oy
	if lx < 0 || ly < 0 || lx > b.width || ly > b.height {
		return gradientColor{}
	}
	dx, dy = lx-b.width/2, ly-b.height/2
	alpha := 0.0
	if b.conic {
		angle := math.Mod(math.Atan2(dx, -dy)*180/math.Pi-b.conicStart+720, 360)
		alpha = math.Max(0, 1-math.Abs(angle-b.conicSpan/2)/(b.conicSpan/2))
	} else {
		distance := math.Hypot(dx, dy) / math.Hypot(b.width/2, b.height/2)
		if b.ellipse {
			distance = math.Hypot(dx/(b.width/2), dy/(b.height/2)) / math.Sqrt2
		}
		alpha = math.Max(0, 1-distance/b.stop)
	}
	color := b.color
	color.a *= alpha * b.opacity
	return color
}
func gradientHardLight(b, s float64) float64 {
	if s <= .5 {
		return 2 * b * s
	}
	return 1 - 2*(1-b)*(1-s)
}

// Both inputs/outputs are premultiplied sRGB. Blend only the intersection:
// Co=(1-as)Cb+(1-ab)Cs+as*ab*B(cb,cs); ao=as+ab-as*ab.
func gradientBlend(dst [4]float32, src gradientColor) [4]float32 {
	a, b := src.a, float64(dst[3])
	if a <= 0 {
		return dst
	}
	out := [4]float32{0, 0, 0, float32(a + b - a*b)}
	channels := [3]float64{src.r, src.g, src.b}
	for i, s := range channels {
		d := float64(dst[i])
		v := (1-a)*d + (1-b)*a*s
		if b > 0 {
			v += a * b * gradientHardLight(d/b, s)
		}
		out[i] = float32(v)
	}
	return out
}
func (c gradientConfig) baseline(x, y float64) gradientColor {
	// CSS 0deg points up; the line spans the box's projected corners.
	dx, dy := math.Sin(40*math.Pi/180), -math.Cos(40*math.Pi/180)
	length := math.Abs(c.width*dx) + math.Abs(c.height*dy)
	t := math.Max(0, math.Min(1, .5+((x-c.width/2)*dx+(y-c.height/2)*dy)/length))
	a, b := c.colors[0], c.colors[1]
	return gradientColor{a.r + (b.r-a.r)*t, a.g + (b.g-a.g)*t, a.b + (b.b-a.b)*t, 1}
}
func (c gradientConfig) blur() float64 {
	switch c.variant {
	case "aurora":
		return 44
	case "chaos":
		return 30
	case "rays":
		return 18
	}
	return 64
}
