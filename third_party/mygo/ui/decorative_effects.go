package ui

import (
	"image"
	"math"
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
type dyeCell struct{ r, g, b, a, vx, vy float32 }
type decorationState struct {
	config         DecorativeEffects
	cells          map[decorationPoint]decorationCell
	fluid, scratch []dyeCell
	width, height  float32
	lastX, lastY   float32
	hasPointer     bool
	pendingPress   bool
	pressX, pressY float32
	lastTime       time.Time
	serial         int
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
	if config.ColorCount == 0 {
		config.Colors[0] = raw.theme.Accent
		config.ColorCount = 1
	}
	if raw.rt.preferences().ReduceMotion {
		config.PointerEnabled = false
	}
	if s.config.PointerEnabled != config.PointerEnabled || s.config.PointerEffect != config.PointerEffect || s.config.Size != config.Size {
		s.clearPointer()
	}
	s.config = config
	if config.PointerEnabled {
		raw.root.flags |= flagTrackPointer
	}
}
func (s *decorationState) clearPointer() {
	s.cells = nil
	s.fluid = nil
	s.scratch = nil
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
	s.serial++
	return s.config.Colors[s.serial%s.config.ColorCount]
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
		s.splat(x, y, dx, dy)
		return
	}
	cell := max(float32(6), 24*s.config.Size/100)
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
		s.clearPointer()
		s.width, s.height = w, h
	}
	if rt.pointerIn {
		x, y := rt.pointerX, rt.pointerY
		if !s.hasPointer {
			s.feed(x, y, 0, 0, now)
		} else if x != s.lastX || y != s.lastY {
			s.feed(x, y, x-s.lastX, y-s.lastY, now)
		}
		s.lastX, s.lastY, s.hasPointer = x, y, true
	} else {
		s.hasPointer = false
	}
	if s.pendingPress {
		s.feed(s.pressX, s.pressY, 0, 0, now)
		s.pendingPress = false
	}
	if s.config.PointerEffect == "splash" {
		s.paintFluid(p, w, h, now)
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

// Native splash uses CPU dye advection, directional momentum, diffusion and
// dissipation. This is not the original WebGL Navier-Stokes pressure solver.
const dyeWidth, dyeHeight = 96, 64

func (s *decorationState) splat(x, y, dx, dy float32) {
	if s.width <= 0 || s.height <= 0 {
		return
	}
	if len(s.fluid) == 0 {
		s.fluid = make([]dyeCell, dyeWidth*dyeHeight)
		s.scratch = make([]dyeCell, len(s.fluid))
	}
	cx, cy := x/s.width*dyeWidth, y/s.height*dyeHeight
	radius := max(float32(1.1), 2.8*s.config.Size/100)
	color := s.color()
	for yy := max(0, int(cy-radius*3)); yy < min(dyeHeight, int(cy+radius*3)+1); yy++ {
		for xx := max(0, int(cx-radius*3)); xx < min(dyeWidth, int(cx+radius*3)+1); xx++ {
			ox, oy := float32(xx)-cx, float32(yy)-cy
			weight := float32(math.Exp(-float64((ox*ox+oy*oy)/(radius*radius)))) * .8
			d := &s.fluid[yy*dyeWidth+xx]
			keep := 1 - weight
			d.r = d.r*keep + float32(color.R)/255*weight
			d.g = d.g*keep + float32(color.G)/255*weight
			d.b = d.b*keep + float32(color.B)/255*weight
			d.a = min(float32(.85), d.a+weight)
			d.vx += dx/s.width*dyeWidth*weight*6 - oy*weight*.8
			d.vy += dy/s.height*dyeHeight*weight*6 + ox*weight*.8
		}
	}
}
func (s *decorationState) sample(x, y float32) dyeCell {
	x = max(0, min(dyeWidth-1, x))
	y = max(0, min(dyeHeight-1, y))
	ix, iy := int(x), int(y)
	fx, fy := x-float32(ix), y-float32(iy)
	a, b := s.fluid[iy*dyeWidth+ix], s.fluid[iy*dyeWidth+min(ix+1, dyeWidth-1)]
	c, d := s.fluid[min(iy+1, dyeHeight-1)*dyeWidth+ix], s.fluid[min(iy+1, dyeHeight-1)*dyeWidth+min(ix+1, dyeWidth-1)]
	mix := func(a, b, c, d float32) float32 { return (a+(b-a)*fx)*(1-fy) + (c+(d-c)*fx)*fy }
	return dyeCell{mix(a.r, b.r, c.r, d.r), mix(a.g, b.g, c.g, d.g), mix(a.b, b.b, c.b, d.b), mix(a.a, b.a, c.a, d.a), mix(a.vx, b.vx, c.vx, d.vx), mix(a.vy, b.vy, c.vy, d.vy)}
}
func (s *decorationState) paintFluid(p *Painter, w, h float32, now time.Time) {
	if len(s.fluid) == 0 {
		return
	}
	dt := float32(now.Sub(s.lastTime).Seconds())
	if s.lastTime.IsZero() {
		dt = 1.0 / 60
	}
	dt = min(dt, .05)
	speed := s.config.Speed / 100
	fade := float32(math.Exp(-float64(dt * speed * 1.8)))
	drag := float32(math.Exp(-float64(dt * 2.2)))
	active := false
	img := image.NewRGBA(image.Rect(0, 0, dyeWidth, dyeHeight))
	for y := 0; y < dyeHeight; y++ {
		for x := 0; x < dyeWidth; x++ {
			i := y*dyeWidth + x
			old := s.fluid[i]
			d := s.sample(float32(x)-old.vx*dt, float32(y)-old.vy*dt)
			neighbors := [4]dyeCell{s.fluid[y*dyeWidth+max(0, x-1)], s.fluid[y*dyeWidth+min(dyeWidth-1, x+1)], s.fluid[max(0, y-1)*dyeWidth+x], s.fluid[min(dyeHeight-1, y+1)*dyeWidth+x]}
			spread := min(float32(.2), dt*2)
			for _, n := range neighbors {
				d.r += (n.r - d.r) * spread
				d.g += (n.g - d.g) * spread
				d.b += (n.b - d.b) * spread
				d.a += (n.a - d.a) * spread
			}
			d.a *= fade
			d.vx *= drag
			d.vy *= drag
			s.scratch[i] = d
			if d.a > .003 {
				active = true
				alpha := min(float32(.75), d.a)
				idx := img.PixOffset(x, y)
				img.Pix[idx] = byte(d.r * alpha * 255)
				img.Pix[idx+1] = byte(d.g * alpha * 255)
				img.Pix[idx+2] = byte(d.b * alpha * 255)
				img.Pix[idx+3] = byte(alpha * 255)
			}
		}
	}
	s.fluid, s.scratch = s.scratch, s.fluid
	if active {
		p.Image(NewBitmap(img), Rect{W: w, H: h}, FillBox)
		p.AnimationFrame()
	} else {
		s.fluid = nil
		s.scratch = nil
	}
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
