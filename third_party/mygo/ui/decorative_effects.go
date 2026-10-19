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
	s.fluidInput = splashRequest{}
	s.fluidBusy = false
	s.fluidLastInput, s.fluidNext = time.Time{}, time.Time{}
	s.fluidSubmitted = time.Time{}
	s.fluidPointerX, s.fluidPointerY = 0, 0
	s.hasPointer = false
	s.pendingPress = false
	s.lastTime = time.Time{}
}
func (s *decorationState) pointerDown(x, y float32) {
	if s.config.PointerEnabled {
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
	s := &rt.decorations
	if !s.config.PointerEnabled || rt.preferences().ReduceMotion {
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
	if rt.pointerIn && s.config.PointerEffect != "splash" {
		x, y := rt.pointerX, rt.pointerY
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
// background but before its label. Disabled buttons retain the original fill.
func (p *Painter) paintMetallicButton(e *node, r Rect) {
	if !p.rt.decorations.config.MetallicButtons || !e.decorativeButton || !e.Hovered() || e.IsDisabled() {
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
	width := max(float32(12), r.W*.24)
	origin := r.X - r.W*1.2 + phase*r.W*1.2
	ink := t.Text.Alpha(.20)
	shine := t.AccentText.Alpha(.38)
	for band := 0; band < 8; band++ {
		x := origin + float32(band)*width*1.3
		p.FillGradient(Rect{X: x, Y: r.Y, W: width, H: r.H}, LinearGradient{From: ink, To: shine, Angle: 115}, 0)
	}
	p.popClip()
	p.clip = savedClip
}
