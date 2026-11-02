package chrome

// This oracle interprets the extracted stylesheet, instead of copying the native
// anchor arrays. It evaluates the CSS cascade/keyframes/transforms independently
// and compares five real time samples for every variant and each bloom.
import (
	"context"
	"image"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/wallpapers"
)

type gradientCSSRule struct{ selector, body string }

func gradientCSSRules(text string) []gradientCSSRule {
	var rules []gradientCSSRule
	for {
		start := strings.Index(text, "{")
		if start < 0 {
			return rules
		}
		depth, end := 1, start+1
		for ; end < len(text) && depth > 0; end++ {
			if text[end] == '{' {
				depth++
			}
			if text[end] == '}' {
				depth--
			}
		}
		if depth != 0 {
			return rules
		}
		rules = append(rules, gradientCSSRule{strings.TrimSpace(text[:start]), text[start+1 : end-1]})
		text = text[end:]
	}
}
func gradientCSSDeclarations(text string) map[string]string {
	values := map[string]string{}
	for _, pair := range strings.Split(text, ";") {
		key, val, ok := strings.Cut(pair, ":")
		if ok {
			values[strings.TrimSpace(key)] = strings.TrimSpace(val)
		}
	}
	return values
}
func gradientCSSNumber(text string) float64 {
	v, _ := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSpace(text), "%"), "deg"), 64)
	return v
}
func gradientCSSLength(text string, size float64) float64 {
	if strings.HasSuffix(text, "%") {
		return gradientCSSNumber(text) / 100 * size
	}
	return gradientCSSNumber(strings.TrimSuffix(text, "px"))
}
func gradientCSSBezier(x float64, mode string) float64 {
	if mode == "linear" {
		return x
	}
	xy := [4]float64{.25, .1, .25, 1}
	if mode == "ease-in-out" {
		xy = [4]float64{.42, 0, .58, 1}
	}
	polynomial := func(t, p, q float64) float64 { return t * (3*p + t*(3*q-6*p+t*(1+3*p-3*q))) }
	// Deliberately different numerical solver from production's interval bisection.
	u := x
	for range 12 {
		v := polynomial(u, xy[0], xy[2]) - x
		derivative := 3*xy[0] + u*(6*xy[2]-12*xy[0]+u*(3+9*xy[0]-9*xy[2]))
		if math.Abs(derivative) < 1e-9 {
			break
		}
		u = math.Min(1, math.Max(0, u-v/derivative))
	}
	return polynomial(u, xy[1], xy[3])
}

type gradientCSSTransform struct{ x, y, angle, scale float64 }

var gradientTransformRE = regexp.MustCompile(`(translateY|translate|rotate|scale)\(([^)]+)\)`)

func gradientCSSTransformValue(text string, w, h float64) gradientCSSTransform {
	out := gradientCSSTransform{scale: 1}
	for _, m := range gradientTransformRE.FindAllStringSubmatch(text, -1) {
		switch m[1] {
		case "translateY":
			out.y = gradientCSSLength(m[2], h)
		case "translate":
			parts := strings.Split(m[2], ",")
			out.x = gradientCSSLength(parts[0], w)
			if len(parts) > 1 {
				out.y = gradientCSSLength(parts[1], h)
			}
		case "rotate":
			out.angle = gradientCSSNumber(m[2]) * math.Pi / 180
		case "scale":
			out.scale = gradientCSSNumber(m[2])
		}
	}
	return out
}
func gradientCSSStyles(rules []gradientCSSRule, variant string, index int) map[string]string {
	out := map[string]string{}
	for _, rule := range rules {
		if !strings.Contains(rule.selector, ">[data-blob") || strings.HasPrefix(rule.selector, "@") {
			continue
		}
		if strings.Contains(rule.selector, "data-variant=") && !strings.Contains(rule.selector, "data-variant="+variant+"]") {
			continue
		}
		if strings.Contains(rule.selector, `data-blob="`) && !strings.Contains(rule.selector, `data-blob="`+strconv.Itoa(index)+`"`) {
			continue
		}
		for k, v := range gradientCSSDeclarations(rule.body) {
			out[k] = v
		}
	}
	return out
}
func gradientCSSSample(rules []gradientCSSRule, variant string, index int, w, h, seconds, speed, x, y float64, reduced bool) float64 {
	style := gradientCSSStyles(rules, variant, index)
	bw, bh := gradientCSSLength(style["width"], w), gradientCSSLength(style["height"], h)
	left, top := gradientCSSLength(style["left"], w), gradientCSSLength(style["top"], h)
	ox, oy := bw/2, bh/2
	origin := style["transform-origin"]
	if strings.HasPrefix(origin, "calc(") {
		offsets := regexp.MustCompile(`calc\(50% ([+-]) ([0-9]+)px\)`).FindAllStringSubmatch(origin, -1)
		for axis, item := range offsets {
			sign := 1.0
			if item[1] == "-" {
				sign = -1
			}
			if axis == 0 {
				ox += sign * gradientCSSNumber(item[2])
			} else {
				oy += sign * gradientCSSNumber(item[2])
			}
		}
	}
	transform := gradientCSSTransform{scale: 1}
	if !reduced {
		animation := style["animation"]
		words := strings.Fields(animation)
		name := words[0]
		duration := gradientCSSNumber(regexp.MustCompile(`calc\(([0-9]+)s`).FindStringSubmatch(animation)[1])
		phase := math.Mod(seconds*speed/100/duration, 1)
		if strings.Contains(animation, "reverse") {
			phase = 1 - phase
		}
		mode := "ease"
		if strings.Contains(animation, "linear") {
			mode = "linear"
		}
		if strings.Contains(animation, "ease-in-out") {
			mode = "ease-in-out"
		}
		var stages []gradientCSSRule
		for _, rule := range rules {
			if rule.selector == "@keyframes "+name {
				stages = gradientCSSRules(rule.body)
			}
		}
		for n := 1; n < len(stages); n++ {
			key := func(s string) float64 {
				if s == "to" {
					return 1
				}
				return gradientCSSNumber(s) / 100
			}
			a, b := key(stages[n-1].selector), key(stages[n].selector)
			if phase < a || phase > b {
				continue
			}
			t := gradientCSSBezier((phase-a)/(b-a), mode)
			from := gradientCSSTransformValue(gradientCSSDeclarations(stages[n-1].body)["transform"], bw, bh)
			to := gradientCSSTransformValue(gradientCSSDeclarations(stages[n].body)["transform"], bw, bh)
			transform = gradientCSSTransform{from.x + (to.x-from.x)*t, from.y + (to.y-from.y)*t, from.angle + (to.angle-from.angle)*t, from.scale + (to.scale-from.scale)*t}
			break
		}
	}
	// Solve an affine transform by its inverse determinant, independently of the
	// native rotation inversion and origin code.
	a, b := math.Cos(transform.angle)*transform.scale, math.Sin(transform.angle)*transform.scale
	px, py := x-left-transform.x-ox, y-top-transform.y-oy
	determinant := a*a + b*b
	lx, ly := (a*px+b*py)/determinant+ox, (-b*px+a*py)/determinant+oy
	if lx < 0 || ly < 0 || lx > bw || ly > bh {
		return 0
	}
	dx, dy := lx-bw/2, ly-bh/2
	radial := style["background"]
	alpha := 0.0
	if strings.HasPrefix(radial, "conic-gradient") {
		values := regexp.MustCompile(`([0-9]+)deg`).FindAllStringSubmatch(radial, -1)
		start, peak, last := gradientCSSNumber(values[0][1]), gradientCSSNumber(values[2][1]), gradientCSSNumber(values[3][1])
		angle := math.Mod(math.Atan2(dx, -dy)*180/math.Pi-start+720, 360)
		if angle <= peak {
			alpha = angle / peak
		} else if angle <= last {
			alpha = (last - angle) / (last - peak)
		}
	} else {
		stop := .5
		if strings.Contains(radial, "55%") {
			stop = .55
		}
		distance := math.Sqrt(dx*dx+dy*dy) / math.Sqrt((bw*bw+bh*bh)/4)
		if strings.Contains(radial, "ellipse") {
			distance = math.Sqrt((dx*dx/(bw*bw/4) + dy*dy/(bh*bh/4)) / 2)
		}
		alpha = math.Max(0, 1-distance/stop)
	}
	return alpha * gradientCSSNumber(style["opacity"])
}
func TestAmbientGradientExtractedCSSOracle(t *testing.T) {
	css, err := os.ReadFile("testdata/installed-gradient-wallpaper.css")
	if err != nil {
		t.Fatal(err)
	}
	rules := gradientCSSRules(string(css))
	for _, variant := range []string{"orbs", "aurora", "chaos", "rays"} {
		for _, reduced := range []bool{false, true} {
			p := wallpapers.Defaults()
			p.Variant = variant
			c := gradientConfiguration(p, 1440, 864, false, reduced)
			for _, seconds := range []float64{0, 1.25, 7.5, 19.75, 37.125} {
				blobs := c.blobs(seconds)
				for i, b := range blobs {
					for y := float64(-120); y <= 1100; y += 97 {
						for x := float64(-130); x <= 1650; x += 103 {
							want := gradientCSSSample(rules, variant, i+1, 1440, 864, seconds, c.speed, x, y, reduced)
							got := b.at(x, y).a
							if math.Abs(got-want) > 2e-5 {
								t.Fatalf("%s reduced=%v t=%.3f blob%d @%.0f,%.0f alpha %.8f source %.8f", variant, reduced, seconds, i+1, x, y, got, want)
							}
						}
					}
				}
			}
		}
	}
}
func TestAmbientGradientBlendAndBlurOrder(t *testing.T) {
	for _, test := range []struct{ b, s, want float64 }{{.2, .25, .1}, {.2, .5, .2}, {.2, .75, .6}, {.8, .25, .4}, {0, 1, 1}, {1, 0, 0}} {
		if got := gradientHardLight(test.b, test.s); math.Abs(got-test.want) > 1e-9 {
			t.Fatalf("hardlight(%v,%v)=%v", test.b, test.s, got)
		}
	}
	transparent := gradientBlend([4]float32{}, gradientColor{.2, .8, .4, .5})
	if transparent != [4]float32{.1, .4, .2, .5} {
		t.Fatalf("transparent backdrop must preserve source premultiplication: %v", transparent)
	}
	dst := [4]float32{.1, .3, .2, .5}
	src := gradientColor{.9, .2, .7, .4}
	got := gradientBlend(dst, src)
	// W3C source-over equation expanded separately for partial overlap.
	for i, s := range [3]float64{src.r, src.g, src.b} {
		b := float64(dst[i]) / .5
		blend := 2 * b * s
		if s > .5 {
			blend = 1 - 2*(1-b)*(1-s)
		}
		want := (1-.4)*float64(dst[i]) + (1-.5)*.4*s + .5*.4*blend
		if math.Abs(float64(got[i])-want) > 1e-7 {
			t.Fatalf("partial alpha %v want %v", got[i], want)
		}
	}
	k := gradientKernel(4)
	sum := float32(0)
	for _, v := range k {
		sum += v
	}
	if math.Abs(float64(sum)-1) > 1e-6 {
		t.Fatal(sum)
	}
	values := make([]float32, 21*21*4)
	temp := make([]float32, len(values))
	values[(10*21+10)*4+3] = 1
	if !gradientGaussian(context.Background(), values, temp, 21, 21, 2, 2) {
		t.Fatal("blur cancelled")
	}
	if values[(10*21+10)*4+3] >= 1 || values[(10*21+9)*4+3] <= 0 {
		t.Fatal("group blur did not spread premultiplied alpha")
	}
}
func TestAmbientGradientBudgetCacheCancellation(t *testing.T) {
	p := wallpapers.Defaults()
	p.Preset = "aurora"
	for _, size := range [][2]float64{{1440, 864}, {2560, 1440}, {3840, 2160}, {1e9, 1e8}, {20, 20}} {
		c := gradientConfiguration(p, size[0], size[1], false, false)
		plan, ok := planGradient(c)
		if !ok || plan.width > gradientRasterEdge || plan.height > gradientRasterEdge || plan.fullWidth > gradientPaddedEdge || plan.fullHeight > gradientPaddedEdge {
			t.Fatalf("unbounded plan %+v for %v", plan, size)
		}
		if plan.fullWidth*plan.fullHeight*32+plan.width*plan.height*16 > gradientCacheByteLimit {
			t.Fatal("byte bound")
		}
	}
	for _, size := range [][2]float64{{0, 10}, {10, -1}, {math.NaN(), 10}, {math.Inf(1), 10}} {
		if _, ok := planGradient(gradientConfiguration(p, size[0], size[1], false, false)); ok {
			t.Fatalf("accepted invalid %v", size)
		}
	}
	cache := gradientFrameCache{}
	now := time.Unix(100, 0)
	c := gradientConfiguration(p, 1440, 864, false, false)
	a := cache.get(c, now)
	if a == nil {
		t.Fatal("first result missing")
	}
	b := cache.get(c, now.Add(time.Millisecond))
	if a != b || cache.renders != 1 {
		t.Fatal("unchanged phase rendered")
	}
	if next := cache.get(c, now.Add(2*gradientFrameInterval)); next == nil || next == a || cache.renders != 2 {
		t.Fatal("phase did not advance")
	}
	cache.reset()
	if cache.frame != nil || cache.configured {
		t.Fatal("reset retained source")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if renderGradient(ctx, c, 0) != nil {
		t.Fatal("cancelled source published")
	}
	c.reduced = true
	a = cache.get(c, now)
	if cache.get(c, now.Add(time.Hour)) != a {
		t.Fatal("reduced motion did not cache static no-transform frame")
	}
	p.Speed = 0
	if got := gradientConfiguration(p, 1440, 864, false, false).speed; got != 20 {
		t.Fatalf("source clamp speed0=%v", got)
	}
}
func TestAmbientGradientPresetAndPanelSamples(t *testing.T) {
	for _, preset := range []string{"default", "dune", "aurora", "sunset", "ocean", "sakura", "custom"} {
		p := wallpapers.Defaults()
		p.Preset = preset
		p.Colors = [7]string{}
		c := gradientConfiguration(p, 1440, 864, false, true)
		pixels := renderGradient(context.Background(), c, 0)
		if pixels == nil || pixels.Bounds().Empty() {
			t.Fatal(preset)
		}
		frame := &gradientFrame{pixels: pixels, config: c}
		panel := gradientPanelPixels(frame, ui.RGB(255, 255, 255), p, false)
		if panel == nil || panel.Bounds() != pixels.Bounds() {
			t.Fatal("panel/source sample dimensions differ")
		}
	}
	p := wallpapers.Defaults()
	p.Colors = [7]string{"#010203", "bad", "", "", "", "", ""}
	c := gradientConfiguration(p, 1440, 864, false, true)
	if c.colors[0].r != 1.0/255 || c.colors[1].r != 219.0/255 {
		t.Fatal("slot override/token fallback")
	}
	// Uniform input checks composed backdrop order: gradient-mask is already in
	// pixels, then canvas .45, dialog mask, blur, and finally saturation.
	src := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for i := 0; i < len(src.Pix); i += 4 {
		src.Pix[i], src.Pix[i+1], src.Pix[i+2], src.Pix[i+3] = 100, 100, 100, 255
	}
	frame := &gradientFrame{pixels: src, config: c}
	panel := gradientPanelPixels(frame, ui.RGB(255, 255, 255), p, false)
	want := uint8(math.Round((255*.45 + 100*.55) * (1 - 61.0/255)))
	if math.Abs(float64(panel.Pix[0])-float64(want)) > 1 {
		t.Fatalf("panel backdrop=%d want%d", panel.Pix[0], want)
	}
}
func BenchmarkAmbientGradient1440(b *testing.B) {
	p := wallpapers.Defaults()
	p.Preset = "aurora"
	c := gradientConfiguration(p, 1440, 864, false, false)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if renderGradient(context.Background(), c, float64(i)/30) == nil {
			b.Fatal("render")
		}
	}
}
