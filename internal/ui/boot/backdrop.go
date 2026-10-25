package boot

import (
	"github.com/egoist/mygo/ui"
)

// The reference boot page is one .scene split at a 62% waterline (boot.css:110-207):
// a sky of cloud and stars above it, the underwater band with the light entering
// it and the drifting marine snow below it, and the abyss at the foot. One
// painter draws every layer, so the page and the reveal curtain's frozen scene
// cannot drift apart.
//
// The scene carries no palette of its own: each color is one of the page's theme
// colors mixed with another (or with black or white), so the boot page follows the
// theme's tokens instead of adding a second color system next to them.
type sceneSpec struct {
	background ui.Color
	accent     ui.Color
	ink        ui.Color
	alert      ui.Color
	dark       bool
	// failed tints the waterline the way body[data-state="error"] does
	// (boot.css:145-147).
	failed bool
}

// The scene's fixed points: .horizon sits at 62% and .abyss is the last 22%.
const (
	sceneHorizon = .62
	sceneAbyss   = .22
)

// sceneTints are the strengths the scene mixes with, per scheme: a sunlit sea in
// light and a deep one in dark (boot-tokens.css --boot-scene, --boot-falloff,
// --boot-snow, --boot-abyss).
type sceneTints struct {
	sky, sea, depth, cloud, star, snow, falloff, abyss float32
}

func (s sceneSpec) tints() sceneTints {
	if s.dark {
		return sceneTints{sky: .16, sea: .12, depth: .82, cloud: .10, star: .55, snow: .5, falloff: .16, abyss: .55}
	}
	// The light scheme has no star fields at all: --boot-star-dust and
	// --boot-star-bright are "none" there (boot-tokens.css:44-45); its clouds are
	// pale tints over a near-white sky, and the painter's halo is a shape where the
	// reference has a radial gradient, so they stay faint enough to read as tone.
	return sceneTints{sky: .10, sea: .34, depth: .56, cloud: .05, star: 0, snow: .22, falloff: .34, abyss: .18}
}

// sceneInk is the contrast color the frozen scene mixes with when its caller does
// not pass the theme's text color.
func sceneInk(dark bool) ui.Color {
	if dark {
		return ui.RGBA(255, 255, 255, 1)
	}
	return ui.RGBA(0, 0, 0, 1)
}

// bootBackdrop paints the scene without motion, for the reveal curtain's frozen
// windows (boot-reveal.css:7-8). It takes the palette the page paints with, so
// the slice the curtain shows is the page's own scene, not a look-alike.
func bootBackdrop(bg, accent ui.Color, dark bool) func(*ui.Painter, ui.Rect) {
	spec := sceneSpec{background: bg, accent: accent, ink: sceneInk(dark), dark: dark}
	return func(p *ui.Painter, r ui.Rect) { paintScene(p, r, spec, false) }
}

// bootScene paints the page's scene: the same layers, moving. The snow and the
// twinkling stars ask for the next paint themselves, so the page keeps drawing
// until something covers it; without motion it is the frozen scene.
func bootScene(spec sceneSpec, motion bool) func(*ui.Painter, ui.Rect) {
	return func(p *ui.Painter, r ui.Rect) {
		if motion {
			p.AnimationFrame()
		}
		paintScene(p, r, spec, motion)
	}
}

// paintScene paints the layers in the reference's order: sky, stars, waterline,
// sea, surface light, snow, abyss.
func paintScene(p *ui.Painter, r ui.Rect, s sceneSpec, motion bool) {
	if r.W <= 0 || r.H <= 0 {
		return
	}
	t := s.tints()
	horizon := r.Y + r.H*sceneHorizon
	sky := ui.Rect{X: r.X, Y: r.Y, W: r.W, H: max(0, horizon-r.Y)}
	sea := ui.Rect{X: r.X, Y: horizon, W: r.W, H: max(0, r.H-(horizon-r.Y))}
	// --boot-scene: the sky runs from the page's own background to a tinted
	// waterline, and the sea keeps deepening to the foot of the page.
	skyTop := s.background.Mix(s.accent, t.sky*.5)
	skyHorizon := s.background.Mix(s.accent, t.sky)
	seaBottom := s.background.Mix(ui.RGBA(0, 0, 0, 1), t.depth).Mix(s.accent, t.sea)
	p.FillGradient(sky, ui.LinearGradient{From: skyTop, To: skyHorizon, Angle: 180}, 0)
	p.FillGradient(sea, ui.LinearGradient{From: skyHorizon, To: seaBottom, Angle: 180}, 0)

	paintSky(p, sky, s, t, motion)

	// .horizon: one clean hairline between sky and sea, tinted with the alert
	// color while the page reports a failure.
	hairline := s.accent.Mix(s.ink, .55).Alpha(.34)
	if s.failed {
		hairline = s.alert.Alpha(.48)
	}
	p.Fill(ui.Rect{X: r.X, Y: horizon, W: r.W, H: 1}, hairline, 0)

	if sea.H > 0 {
		// .underwater::before: the light entering the water at the surface, a
		// soft bloom clipped to the sea.
		under := ui.RGBA(255, 255, 255, 1)
		if s.dark {
			under = s.accent.Mix(ui.RGBA(255, 255, 255, 1), .35)
		}
		p.Clip(sea, 0, func() {
			cloud(p, ui.Rect{X: r.X + r.W*.08, Y: horizon, W: r.W * .84, H: max(sea.H*.12, 8)},
				under.Alpha(t.falloff*.32), sea.H*.12)
		})
		paintSnow(p, sea, s, t, motion)
	}

	// .abyss: the foot of the scene goes dark in both schemes.
	p.FillGradient(
		ui.Rect{X: r.X, Y: r.Y + r.H*(1-sceneAbyss), W: r.W, H: r.H * sceneAbyss},
		ui.LinearGradient{From: ui.RGBA(0, 0, 0, 0), To: ui.RGBA(0, 0, 0, 1).Alpha(t.abyss), Angle: 180},
		0,
	)
}

// paintSky paints .stars: the three cloud fields and, in the dark scheme, the two
// star fields (boot-tokens.css --boot-nebula-*, --boot-star-dust, --boot-star-bright).
func paintSky(p *ui.Painter, sky ui.Rect, s sceneSpec, t sceneTints, motion bool) {
	if sky.W <= 0 || sky.H <= 0 {
		return
	}
	// A warm cloud on the left, a cool one high on the right, a pale one between
	// them (boot.css:116-124).
	cloud(p, skyBox(sky, .26, .26, .62, .46), s.ink.Mix(s.background, .90).Alpha(t.cloud*.8), sky.H*.10)
	cloud(p, skyBox(sky, .70, .14, .52, .40), s.accent.Mix(s.background, .88).Alpha(t.cloud), sky.H*.10)
	cloud(p, skyBox(sky, .52, .46, .46, .30), s.accent.Mix(s.ink, .92).Alpha(t.cloud*.8), sky.H*.07)
	if t.star <= 0 {
		return
	}
	// Dust: a fixed lattice, so every paint of the page is the same sky.
	dust := s.ink.Alpha(t.star * .5)
	for i := 0; i < 46; i++ {
		fx := float32((i*37)%100) / 100
		fy := float32((i*53)%100) / 100
		d := float32(1)
		if i%4 == 0 {
			d = 1.5
		}
		p.Fill(ui.Rect{X: sky.X + sky.W*(.03+.94*fx), Y: sky.Y + sky.H*(.04+.90*fy), W: d, H: d},
			dust.Alpha(.5+float32(i%5)*.1), d/2)
	}
	// Bright stars: the reference's six, twinkling on its 5.5s alternate
	// (boot.css:127-132, twinkle).
	bright := s.ink.Alpha(t.star * .85)
	for _, star := range [][3]float32{{.09, .20, 1.0}, {.26, .10, 1.1}, {.58, .16, 1.0}, {.82, .12, 1.2}, {.93, .34, 1.0}, {.44, .08, .9}} {
		alpha := float32(1)
		if motion {
			alpha = twinkle(p.Now().UnixMilli())
		}
		d := star[2] * 1.8
		p.Fill(ui.Rect{X: sky.X + sky.W*star[0] - d/2, Y: sky.Y + sky.H*star[1] - d/2, W: d, H: d},
			bright.Alpha(alpha), d/2)
	}
}

// skyBox is the box a cloud covers in r: fw and fh of its size, centred on (fx, fy).
func skyBox(r ui.Rect, fx, fy, fw, fh float32) ui.Rect {
	w, h := r.W*fw, r.H*fh
	return ui.Rect{X: r.X + r.W*fx - w/2, Y: r.Y + r.H*fy - h/2, W: w, H: h}
}

// cloud paints a soft ellipse. The painter fills no radial gradient, and a flat
// fill would end on a rounded rectangle's rim, so a cloud is a blurred silhouette:
// a box smaller than the shape, blurred well past its edge, which paints as a soft
// halo. That is what the reference's radial-gradient clouds and its surface light
// read as (boot.css:116-124,171-177).
func cloud(p *ui.Painter, box ui.Rect, color ui.Color, blur float32) {
	if box.W <= 0 || box.H <= 0 || color.A == 0 {
		return
	}
	core := ui.Rect{X: box.X + box.W*.22, Y: box.Y + box.H*.22, W: box.W * .56, H: box.H * .56}
	p.Shadow(core, core.H*.5, 0, 0, max(blur, 1), 0, color)
}

// twinkle is the stars' alternate: 0.55 to 1 and back over 5.5s.
func twinkle(ms int64) float32 {
	const period = 5500
	t := float32(ms%period) / period
	if t > .5 {
		t = 1 - t
	}
	return .55 + .9*t
}

// paintSnow is the marine snow: five particles drifting through the sea on the
// reference's 14-21s cycles (boot.css:198-205, snow-fall).
func paintSnow(p *ui.Painter, sea ui.Rect, s sceneSpec, t sceneTints, motion bool) {
	if sea.H <= 0 {
		return
	}
	color := s.ink.Alpha(t.snow)
	lefts := [5]float32{.32, .41, .53, .62, .71}
	periods := [5]int64{14000, 19000, 16000, 21000, 17000}
	for i := range lefts {
		size := float32(2)
		if i == 1 || i == 3 {
			size = 1.5
		}
		// Without motion the particles stand where their fall would have them.
		progress := float32(i) * .21
		if motion {
			progress = float32(p.Now().UnixMilli()%periods[i]) / float32(periods[i])
		}
		y := sea.Y + sea.H*(.02+.92*progress)
		p.Fill(ui.Rect{X: sea.X + sea.W*lefts[i], Y: y, W: size, H: size},
			color.Alpha(snowFade(progress)), size/2)
	}
}

// snowFade is snow-fall's opacity: in over the first 12% of the fall, out over
// the last 12% (boot.css:536-542).
func snowFade(t float32) float32 {
	if t < .12 {
		return t / .12
	}
	if t > .88 {
		return max(0, (1-t)/.12)
	}
	return 1
}
