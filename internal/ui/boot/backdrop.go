package boot

import (
	"image"
	"math"
	"sync"

	"github.com/egoist/mygo/ui"
)

const sceneHorizon = .62

type sceneStop struct {
	at    float32
	color ui.Color
}
type scenePalette struct {
	ink, muted, brand, alert, hairline, log, panel ui.Color
	underwater, falloff, abyss                     ui.Color
	// underAlert is --boot-under-alert, the ticker's colour while the Host is
	// down, darker than the alert it sits over sea (boot-tokens.css:13,52).
	underAlert ui.Color
	clouds     [3]ui.Color
	stops      []sceneStop
}

// The installed Whale Isle 0.3.15 boot-tokens.css has its own light and dark
// sea palettes. Only the scheme comes from the workbench theme.
var day = scenePalette{
	ink: ui.RGB(24, 42, 54), muted: ui.RGB(96, 116, 128), brand: ui.RGB(24, 42, 54), alert: ui.RGB(198, 46, 36),
	hairline: ui.RGBA(30, 60, 76, .38), log: ui.RGB(230, 244, 249), panel: ui.RGBA(248, 251, 253, .92),
	underAlert: ui.RGB(140, 30, 22),
	underwater: ui.RGBA(96, 150, 176, .30), falloff: ui.RGBA(255, 255, 255, .5), abyss: ui.RGBA(40, 80, 105, .18),
	clouds: [3]ui.Color{ui.RGBA(255, 240, 214, .7), ui.RGBA(182, 212, 228, .62), ui.RGBA(208, 228, 238, .6)},
	stops:  []sceneStop{{0, ui.RGB(241, 246, 248)}, {.55, ui.RGB(229, 239, 243)}, {.62, ui.RGB(204, 227, 235)}, {.70, ui.RGB(160, 192, 210)}, {.85, ui.RGB(118, 160, 182)}, {1, ui.RGB(84, 128, 152)}},
}
var night = scenePalette{
	ink: ui.RGB(234, 240, 244), muted: ui.RGB(118, 132, 144), brand: ui.RGBA(214, 226, 236, .68), alert: ui.RGB(255, 92, 80),
	hairline: ui.RGBA(190, 224, 240, .4), log: ui.RGB(132, 194, 214), panel: ui.RGBA(8, 16, 26, .88),
	underAlert: ui.RGB(255, 140, 128),
	underwater: ui.RGBA(14, 44, 66, .52), falloff: ui.RGBA(96, 170, 200, .16), abyss: ui.RGBA(1, 6, 12, .55),
	clouds: [3]ui.Color{ui.RGBA(72, 112, 168, .22), ui.RGBA(116, 88, 154, .17), ui.RGBA(60, 120, 150, .16)},
	stops:  []sceneStop{{0, ui.RGB(2, 4, 9)}, {.40, ui.RGB(4, 9, 17)}, {.55, ui.RGB(8, 18, 30)}, {.62, ui.RGB(12, 34, 48)}, {.78, ui.RGB(8, 24, 38)}, {1, ui.RGB(4, 13, 23)}},
}

func bootPalette(dark bool) *scenePalette {
	if dark {
		return &night
	}
	return &day
}

var dayScene = sync.OnceValue(func() *ui.Bitmap { return ui.NewBitmap(scenePixels(false)) })
var nightScene = sync.OnceValue(func() *ui.Bitmap { return ui.NewBitmap(scenePixels(true)) })

func ramp(stops []sceneStop, pos float32) ui.Color {
	for i := 1; i < len(stops); i++ {
		if pos <= stops[i].at {
			return stops[i-1].color.Mix(stops[i].color, (pos-stops[i-1].at)/(stops[i].at-stops[i-1].at))
		}
	}
	return stops[len(stops)-1].color
}

func ellipse(x, y, cx, cy, rx, ry float32) float32 {
	dx, dy := (x-cx)/rx, (y-cy)/ry
	return max(0, 1-float32(math.Sqrt(float64(dx*dx+dy*dy)))/.72)
}

// The atmosphere is low frequency. Rasterize only once per used scheme, then
// stretch the immutable texture under the DIP-sized horizon and particles.
func scenePixels(dark bool) *image.RGBA {
	const w, h = 384, 384
	p := bootPalette(dark)
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	for row := range h {
		y := (float32(row) + .5) / h
		base := ramp(p.stops, y)
		for col := range w {
			x := (float32(col) + .5) / w
			c := base
			if y < sceneHorizon {
				ySky := y / sceneHorizon
				if dark {
					band := max(0, 1-float32(math.Abs(float64(.84*(x-.5)+.16*(ySky-.5))))/.18)
					c = ui.RGBA(160, 200, 228, .12*band).Over(c)
				}
				c = p.clouds[2].Alpha(ellipse(x, ySky, .52, .46, .34, .22)).Over(c)
				c = p.clouds[1].Alpha(ellipse(x, ySky, .70, .14, .40, .30)).Over(c)
				c = p.clouds[0].Alpha(ellipse(x, ySky, .26, .26, .48, .34)).Over(c)
			} else {
				ySea := (y - sceneHorizon) / (1 - sceneHorizon)
				c = p.underwater.Alpha(max(0, 1-ySea/.62)).Over(c)
				c = p.falloff.Alpha(ellipse(x, ySea, .5, 0, 1.2, .58)).Over(c)
				c = p.abyss.Alpha(max(0, (y-.78)/.22)).Over(c)
			}
			o := row*out.Stride + col*4
			out.Pix[o], out.Pix[o+1], out.Pix[o+2], out.Pix[o+3] = c.R, c.G, c.B, 255
		}
	}
	return out
}

var starDust = [][2]float32{
	{.15, .42}, {.22, .30}, {.33, .46}, {.40, .22}, {.47, .38}, {.52, .08}, {.66, .30}, {.74, .44}, {.88, .22}, {.05, .48},
	{.08, .26}, {.36, .14}, {.61, .12}, {.79, .36}, {.96, .14}, {.13, .60}, {.31, .66}, {.44, .56}, {.60, .62}, {.77, .58},
	{.95, .54}, {.19, .74}, {.37, .80}, {.55, .72}, {.70, .78}, {.85, .68}, {.03, .30}, {.97, .46}, {.11, .38}, {.25, .52},
	{.42, .68}, {.49, .24}, {.57, .48}, {.64, .70}, {.72, .16}, {.83, .76}, {.91, .62}, {.17, .16}, {.29, .06}, {.45, .04}, {.69, .04},
}
var brightStars = [][2]float32{{.09, .20}, {.26, .10}, {.58, .16}, {.82, .12}, {.93, .34}, {.44, .08}}

func bootScene(dark, failed, motion bool) func(*ui.Painter, ui.Rect) {
	return func(p *ui.Painter, r ui.Rect) {
		palette := bootPalette(dark)
		if dark {
			p.Image(nightScene(), r, ui.FillBox)
		} else {
			p.Image(dayScene(), r, ui.FillBox)
		}
		now := motionNow(p.Now())
		if motion {
			p.After(motionWait(p.Now()))
		}
		horizon := r.Y + r.H*sceneHorizon
		line := palette.hairline
		if failed {
			line = palette.alert.Alpha(.45)
		}
		p.Fill(ui.Rect{X: r.X, Y: horizon, W: r.W, H: 1}, line, 0)
		point := func(x, y, d float32, c ui.Color) { p.Fill(ui.Rect{X: x - d/2, Y: y - d/2, W: d, H: d}, c, d/2) }
		if dark {
			for i, s := range starDust {
				d, a := float32(1), float32(.68)
				if i < 15 {
					d, a = 1.4, .85
				}
				point(r.X+r.W*s[0], r.Y+r.H*sceneHorizon*s[1], d, ui.RGBA(214, 232, 246, a))
			}
			alpha := float32(.75)
			if motion {
				alpha = .55 + .45*float32((1-math.Cos(float64(now.UnixMilli()%11000)/11000*2*math.Pi))/2)
			}
			for i, s := range brightStars {
				ink := ui.RGB(240, 248, 254)
				if i == 5 {
					ink = ui.RGB(255, 232, 200)
				}
				x, y := r.X+r.W*s[0], r.Y+r.H*sceneHorizon*s[1]
				point(x, y, 3.2, ink.Alpha(alpha*.30))
				point(x, y, 1.4, ink.Alpha(alpha))
			}
		}
		sea := r.H * (1 - sceneHorizon)
		ink := ui.RGBA(60, 110, 135, .28)
		if dark {
			ink = ui.RGBA(175, 212, 230, .5)
		}
		delays := [5]int64{0, 6000, 3000, 11000, 8000}
		periods := [5]int64{14000, 19000, 16000, 21000, 17000}
		for i, x := range [5]float32{.32, .41, .53, .62, .71} {
			phase := float32(i+1) * .15
			if motion {
				phase = float32((now.UnixMilli()+delays[i])%periods[i]) / float32(periods[i])
			}
			xDrift := (phase - .5) * 12
			d := float32(2)
			if i == 1 || i == 3 {
				d = 1.5
			}
			point(r.X+r.W*x+xDrift, horizon+sea*(.08+.83*phase), d, ink.Alpha(min(1, phase/.12, (1-phase)/.12)))
		}
	}
}
