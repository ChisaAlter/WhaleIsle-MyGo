package ui

import (
	"math"
	"math/rand/v2"
	"time"
)

// DecorativeEffects configures paint-only, window-local pointer and button
// decoration. It creates no hit target and does not consume input.
type DecorativeEffects struct {
	PointerEnabled  bool
	PointerEffect   string
	Colors          [6]Color
	ColorCount      int
	Speed, Size     float32
	MetallicButtons bool
}
type decorationCell struct {
	born  time.Time
	color Color
}
type decorationPoint struct{ x, y int }

type decorationState struct {
	config                       DecorativeEffects
	cells                        map[decorationPoint]decorationCell
	fluidWorker                  *splashWorker
	fluidError                   error
	fluidBitmap                  *Bitmap
	fluidInput                   splashRequest
	fluidBusy                    bool
	fluidLastInput, fluidNext    time.Time
	fluidSubmitted               time.Time
	fluidPointerX, fluidPointerY float32
	random                       *rand.Rand
	width, height                float32
	lastX, lastY                 float32
	hasPointer                   bool
	pendingPress                 bool
	pressX, pressY               float32
	lastTime                     time.Time
}

// SetDecorativeEffects updates the native renderer; desktop Reduce Motion
// disables pointer animation and makes metallic button highlights static.
func SetDecorativeEffects(c *Context, config DecorativeEffects) {
	raw := c.build()
	if raw == nil {
		return
	}
	s := &raw.rt.decorations
	if config.Speed <= 0 {
		config.Speed = 100
	}
	if config.Size <= 0 {
		config.Size = 100
	}

	if raw.rt.preferences().ReduceMotion {
		config.PointerEnabled = false
	}
	if s.config.PointerEnabled != config.PointerEnabled || s.config.PointerEffect != config.PointerEffect {
		s.clearPointer()
	}
	s.config = config
	if config.PointerEnabled {
		raw.root.flags |= flagTrackPointer
	}
}
func (s *decorationState) clearPointer() {
	s.cells = nil
	if s.fluidWorker != nil {
		s.fluidWorker.close()
	}
	s.fluidWorker, s.fluidBitmap = nil, nil
	s.fluidError = nil
	s.fluidInput = splashRequest{}
	s.fluidBusy = false
	s.fluidLastInput, s.fluidNext = time.Time{}, time.Time{}
	s.fluidSubmitted = time.Time{}
	s.fluidPointerX, s.fluidPointerY = 0, 0
	s.hasPointer = false
	s.pendingPress = false
	s.lastTime = time.Time{}
}

const maxQueuedSplashPresses = 64

func (s *decorationState) pointerDown(x, y float32) {
	if s.config.PointerEnabled {
		if s.config.PointerEffect == "splash" && s.fluidError == nil && s.width > 0 && s.height > 0 {
			// Original pointerDown updates the pointer's last texture coordinate;
			// the following motion must not inject a delta from before the click.
			s.fluidPointerX, s.fluidPointerY = x/s.width, 1-y/s.height
			s.fluidInput.x, s.fluidInput.y = s.fluidPointerX, s.fluidPointerY
			if len(s.fluidInput.presses) >= maxQueuedSplashPresses {
				copy(s.fluidInput.presses, s.fluidInput.presses[1:])
				s.fluidInput.presses = s.fluidInput.presses[:maxQueuedSplashPresses-1]
			}
			s.fluidInput.presses = append(s.fluidInput.presses, splashPress{s.fluidPointerX, s.fluidPointerY, s.config})
		}
		s.pendingPress = true
		s.pressX = x
		s.pressY = y
	}
}
func (s *decorationState) color() Color {
	if s.config.ColorCount == 0 {
		return RGB(136, 136, 136)
	}
	if s.random == nil {
		seed := uint64(time.Now().UnixNano())
		s.random = rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	}
	return s.config.Colors[s.random.IntN(s.config.ColorCount)]
}
func (s *decorationState) stampTrail(x, y float32, now time.Time) {
	cell := float32(math.Max(6, math.Round(float64(24*s.config.Size/100))))
	radius := cell * 1.2
	if s.cells == nil {
		s.cells = map[decorationPoint]decorationCell{}
	}
	for gy := int(math.Floor(float64((y - radius) / cell))); gy <= int((y+radius)/cell); gy++ {
		for gx := int(math.Floor(float64((x - radius) / cell))); gx <= int((x+radius)/cell); gx++ {
			dx, dy := (float32(gx)+.5)*cell-x, (float32(gy)+.5)*cell-y
			if dx*dx+dy*dy > radius*radius {
				continue
			}
			if len(s.cells) >= 4000 {
				return
			}
			s.cells[decorationPoint{gx, gy}] = decorationCell{now, s.color()}
		}
	}
}
func (s *decorationState) feed(x, y, dx, dy float32, now time.Time) {
	if s.config.PointerEffect == "splash" {
		s.queueSplash(x, y, dx, dy, now)
		return
	}
	cell := float32(math.Max(6, math.Round(float64(24*s.config.Size/100))))
	steps := max(1, int(math.Hypot(float64(dx), float64(dy))/float64(cell*.75)))
	for i := 1; i <= steps; i++ {
		fraction := float32(i) / float32(steps)
		s.stampTrail(x-dx+dx*fraction, y-dy+dy*fraction, now)
	}
}
func (rt *engine) paintPointerDecoration(p *Painter, w, h float32) {
	rt.decorations.paintPointer(p, w, h, rt.pointerIn, rt.pointerX, rt.pointerY, rt.preferences().ReduceMotion)
}
func (s *decorationState) paintPointer(p *Painter, w, h float32, pointerIn bool, pointerX, pointerY float32, reduced bool) {
	if !s.config.PointerEnabled || reduced {
		s.clearPointer()
		return
	}
	now := p.Now()
	if s.width != w || s.height != h {
		if s.config.PointerEffect != "splash" {
			s.cells, s.hasPointer = nil, false
		}
		s.width, s.height = w, h
	}
	if pointerIn && s.config.PointerEffect != "splash" {
		x, y := pointerX, pointerY
		if !s.hasPointer {
			s.feed(x, y, 0, 0, now)
		} else if x != s.lastX || y != s.lastY {
			s.feed(x, y, x-s.lastX, y-s.lastY, now)
		}
		s.lastX, s.lastY, s.hasPointer = x, y, true
	} else if s.config.PointerEffect != "splash" {
		s.hasPointer = false
	}
	if s.pendingPress && s.config.PointerEffect != "splash" {
		s.feed(s.pressX, s.pressY, 0, 0, now)
		s.pendingPress = false
	}
	if s.config.PointerEffect == "splash" {
		s.paintSplash(p, w, h, now)
	} else {
		cell := float32(math.Max(6, math.Round(float64(24*s.config.Size/100))))
		gap := max(float32(1), float32(math.Round(float64(cell*.12))))
		life := max(.15, .7*100/float64(s.config.Speed))
		for key, value := range s.cells {
			age := now.Sub(value.born).Seconds()
			if age >= life {
				delete(s.cells, key)
				continue
			}
			p.Fill(Rect{X: float32(key.x)*cell + gap, Y: float32(key.y)*cell + gap, W: cell - 2*gap, H: cell - 2*gap}, value.color.Alpha(float32(1-age/life)), 0)
		}
		if len(s.cells) > 0 {
			p.AnimationFrame()
		}
	}
	s.lastTime = now
}

// Sheen is clipped to the button's own corners and painted after its hover
// background but before its label. Disabled buttons and non-button rows (such
// as disclosures, menu items, tabs and selects) retain only their plain fill.
func (p *Painter) paintMetallicButton(e *node, r Rect) {
	if !p.rt.decorations.config.MetallicButtons || !e.decorativeButton || !e.Hovered() || e.IsDisabled() {
		return
	}
	if e.role != RoleAuto && e.role != RoleButton && e.role != RoleToggleButton {
		return
	}
	t := p.Theme()
	phase := float32(.5)
	if !p.rt.preferences().ReduceMotion {
		phase = float32(.5 - .5*math.Cos(float64(p.Now().UnixNano())/1e9*2*math.Pi/4.5))
		p.AnimationFrame()
	}
	savedClip := p.clip
	p.pushClip(r, e.radius)
	// Match metallic-paint.css: 115deg multi-stop sheen at 320% span sweeping
	// 0% -> 100% -> 0% over 4.5s ease-in-out (static midpoint under ReduceMotion).
	span := max(float32(72), r.W*3.2)
	origin := r.X - phase*max(float32(0), span-r.W)
	mid := t.TextMuted.Alpha(.14)
	shineBright := t.AccentText.Alpha(.42)
	shineSoft := t.AccentText.Alpha(.36)
	ink := t.Text.Alpha(.20)
	stops := [7]struct {
		at    float32
		color Color
	}{
		{0, mid},
		{.18, shineBright},
		{.34, ink},
		{.52, shineSoft},
		{.68, ink},
		{.84, shineSoft},
		{1, mid},
	}
	for i := 0; i < len(stops)-1; i++ {
		x0 := origin + stops[i].at*span
		x1 := origin + stops[i+1].at*span
		p.FillGradient(Rect{X: x0, Y: r.Y, W: x1 - x0, H: r.H}, LinearGradient{From: stops[i].color, To: stops[i+1].color, Angle: 115}, 0)
	}
	p.popClip()
	p.clip = savedClip
}

// DecorativePreview owns an isolated, clipped instance of the same native
// pointer renderer. Close releases its worker when the settings pane leaves.
type DecorativePreview struct {
	state  decorationState
	x, y   float32
	inside bool
	bounds Rect
}

func (d *DecorativePreview) Configure(config DecorativeEffects) {
	if config.Speed <= 0 {
		config.Speed = 100
	}
	if config.Size <= 0 {
		config.Size = 100
	}
	if d.state.config.PointerEnabled != config.PointerEnabled || d.state.config.PointerEffect != config.PointerEffect {
		d.state.clearPointer()
	}
	d.state.config = config
}
func (d *DecorativePreview) Input(e InputEvent) bool {
	d.x, d.y = d.bounds.X+e.X, d.bounds.Y+e.Y
	switch e.Kind {
	case InputPointerMove:
		d.inside = true
		if d.state.config.PointerEffect == "splash" {
			d.state.feed(d.x, d.y, d.x-d.state.lastX, d.y-d.state.lastY, time.Now())
			d.state.lastX, d.state.lastY = d.x, d.y
		}
		return true
	case InputPointerDown:
		d.inside = true
		d.state.pointerDown(d.x, d.y)
		return true
	case InputPointerUp:
		return true
	case InputBlur:
		d.inside = false
	}
	return false
}
func (d *DecorativePreview) Draw(p *Painter, r Rect) {
	d.bounds = r
	p.Clip(r, 0, func() { d.state.paintPointer(p, r.X+r.W, r.Y+r.H, d.inside, d.x, d.y, p.rt.preferences().ReduceMotion) })
}
func (d *DecorativePreview) Close() { d.state.clearPointer(); d.inside = false }
