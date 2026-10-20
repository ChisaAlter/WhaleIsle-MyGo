package pet

import (
	"encoding/json"
	"github.com/egoist/mygo"
	"math"
	"net/http"
	"time"
	"whaleisle/internal/store/storelocale"
)

// Native cursor sampling owns all desktop movement. The fixed-origin page
// sends gesture names only, never screen coordinates or arbitrary move calls.
type interactionSample struct {
	at   time.Time
	x, y float64
}
type interactionView struct {
	Epoch            uint64
	CursorX, CursorY float64
	Mode             string
	Roaming          bool
	Facing           int
	VX, VY           float64
	Landed           int64
}
type interactionState struct {
	timer                  *time.Timer
	generation             uint64
	pressed                bool
	mode                   string
	origin, grab, position motionPoint
	trail                  []interactionSample
	last                   time.Time
	vx, vy                 float64
	view                   interactionView
	hit                    petHitState
}

func (p *Companion) startInteraction() {
	p.stopInteraction()
	p.interaction.last = time.Now()
	p.armInteraction()
}
func (p *Companion) stopInteraction() {
	s := &p.interaction
	if s.timer != nil {
		s.timer.Stop()
	}
	generation := s.generation + 1
	p.assetMu.Lock()
	*s = interactionState{generation: generation, view: interactionView{Epoch: generation}}
	p.assetMu.Unlock()
	p.resetHitRegions()
}
func (p *Companion) armInteraction() {
	w, generation := p.win, p.interaction.generation
	if w == nil || w.IsDestroyed() || !w.IsVisible() {
		return
	}
	p.interaction.timer = time.AfterFunc(time.Second/60, func() {
		w.Update(func() {
			if p.win != w || w.IsDestroyed() || p.interaction.generation != generation {
				return
			}
			p.interaction.timer = nil
			p.tickInteraction(time.Now())
			p.armInteraction()
		})
	})
}
func (p *Companion) tickInteraction(now time.Time) {
	s := &p.interaction
	cursor := mygo.Screen.CursorScreenPoint()
	bounds := p.win.Bounds()
	dt := math.Min(.05, now.Sub(s.last).Seconds())
	s.last = now
	if s.pressed {
		sample := interactionSample{now, float64(cursor.X), float64(cursor.Y)}
		if len(s.trail) == 0 || s.trail[len(s.trail)-1].x != sample.x || s.trail[len(s.trail)-1].y != sample.y {
			s.trail = append(s.trail, sample)
		}
		for len(s.trail) > 0 && now.Sub(s.trail[0].at) > 400*time.Millisecond {
			s.trail = s.trail[1:]
		}
	}
	if s.mode == "pickup" {
		p.motionContact()
		target := motionPoint{x: float64(cursor.X) - s.grab.x, y: float64(cursor.Y) - s.grab.y}
		if dt > 0 {
			s.vx, s.vy = (target.x-s.position.x)/dt, (target.y-s.position.y)/dt
		}
		s.position = target
		p.setMotionPosition(target)
	} else if s.mode == "throw" && dt > 0 {
		p.motionContact()
		if area, ok := motionWorkArea(bounds); ok {
			left, top := area.x, area.y
			right, floor := math.Max(left, area.x+area.width-float64(bounds.Width)), math.Max(top, area.y+area.height-float64(bounds.Height))
			s.vy += 1400 * dt
			s.position.x += s.vx * dt
			s.position.y += s.vy * dt
			if s.position.x < left {
				s.position.x = left
				s.vx = math.Abs(s.vx) * .78
			}
			if s.position.x > right {
				s.position.x = right
				s.vx = -math.Abs(s.vx) * .78
			}
			if s.position.y < top {
				s.position.y = top
				s.vy = math.Abs(s.vy) * .78
			}
			if s.position.y >= floor {
				s.position.y = floor
				s.vx *= math.Max(0, 1-2.5*dt)
				if math.Abs(s.vy) < 40 {
					s.vy = 0
				} else {
					s.vy = -math.Abs(s.vy) * .78
				}
			}
			p.setMotionPosition(s.position)
			if s.position.y >= floor-1 && math.Abs(s.vy) < 1 && math.Abs(s.vx) < 15 {
				p.landInteraction(now)
			}
		} else {
			p.landInteraction(now)
		}
	}
	bounds = p.win.Bounds()
	p.assetMu.Lock()
	s.view.CursorX, s.view.CursorY = float64(cursor.X-bounds.X), float64(cursor.Y-bounds.Y)
	s.view.Mode, s.view.VX, s.view.VY = s.mode, s.vx, s.vy
	s.view.Roaming, s.view.Facing = p.motion.glide != nil, p.motion.facing
	p.assetMu.Unlock()
	p.updateHitRegions(cursor)
}
func (p *Companion) landInteraction(now time.Time) {
	s := &p.interaction
	s.mode, s.vx, s.vy = "", 0, 0
	if area, ok := motionWorkArea(p.win.Bounds()); ok {
		b := p.win.Bounds()
		s.position = clampMotionPoint(s.position, float64(b.Width), float64(b.Height), area)
		p.setMotionPosition(s.position)
	}
	p.saveMotionAnchor(s.position)
	p.assetMu.Lock()
	s.view.Landed = now.UnixMilli()
	p.assetMu.Unlock()
}
func (p *Companion) interactionAction(raw string) bool {
	if p.hitRegionAction(raw) {
		return true
	}
	action := ""
	for _, candidate := range []string{"gesture-press", "gesture-pickup", "gesture-release", "gesture-cancel"} {
		if raw == "whale-pet://localhost/"+candidate {
			action = candidate
			break
		}
	}
	if action == "" {
		return false
	}
	if p.win == nil || p.win.IsDestroyed() || p.Store == nil || !p.ready || !p.Store.DesktopPet().Enabled {
		return true
	}
	p.motionContact()
	s, now := &p.interaction, time.Now()
	switch action {
	case "gesture-press":
		cursor := mygo.Screen.CursorScreenPoint()
		s.pressed, s.mode = true, ""
		s.origin = motionWindowPoint(p.win)
		s.position = s.origin
		s.grab = motionPoint{x: float64(cursor.X) - s.origin.x, y: float64(cursor.Y) - s.origin.y}
		s.trail = []interactionSample{{now, float64(cursor.X), float64(cursor.Y)}}
	case "gesture-pickup":
		if s.pressed && !p.savedBehavior().LockPosition {
			s.mode = "pickup"
			p.restPet("wake-contact")
		}
	case "gesture-release", "gesture-cancel":
		if !s.pressed {
			return true
		}
		s.pressed = false
		moved := s.mode == "pickup"
		if moved {
			s.vx, s.vy = interactionRelease(s.trail, now)
			if action == "gesture-release" && math.Hypot(s.vx, s.vy) >= 500 {
				s.mode = "throw"
				p.interactionCare(true)
			} else {
				p.landInteraction(now)
			}
		}
		if !moved && action == "gesture-release" {
			p.interactionCare(false)
		}
		s.trail = nil
	}
	p.updateHitRegions(mygo.Screen.CursorScreenPoint())
	return true
}
func interactionRelease(trail []interactionSample, now time.Time) (float64, float64) {
	if len(trail) == 0 || now.Sub(trail[len(trail)-1].at) > 150*time.Millisecond {
		return 0, 0
	}
	for len(trail) > 0 && now.Sub(trail[0].at) > 120*time.Millisecond {
		trail = trail[1:]
	}
	if len(trail) < 2 {
		return 0, 0
	}
	a, b := trail[0], trail[len(trail)-1]
	span := b.at.Sub(a.at).Seconds()
	if span < .02 {
		return 0, 0
	}
	vx, vy := (b.x-a.x)/span, (b.y-a.y)/span
	base := math.Hypot(vx, vy)
	peak, first, last := base, 0.0, 0.0
	var firstAt, lastAt time.Time
	previous := a
	for _, sample := range trail[1:] {
		dt := sample.at.Sub(previous.at).Seconds()
		if dt < .008 {
			continue
		}
		speed := math.Hypot(sample.x-previous.x, sample.y-previous.y) / dt
		if firstAt.IsZero() {
			first, firstAt, peak = speed, sample.at, speed
		}
		last, lastAt = speed, sample.at
		peak = math.Max(peak, speed)
		previous = sample
	}
	accel := 0.0
	if !firstAt.IsZero() && lastAt.After(firstAt) {
		accel = (last - first) / math.Max(.02, lastAt.Sub(firstAt).Seconds())
	}
	speed := (base + peak) * .5 * (1 + math.Min(1, math.Max(0, accel)/8000)*.6)
	speed = 4800 * (1 - math.Exp(-speed/4800))
	if base < 1e-6 {
		return 0, speed
	}
	return vx / base * speed, vy / base * speed
}
func (p *Companion) interactionCare(thrown bool) {
	p.assetMu.Lock()
	defer p.assetMu.Unlock()
	if p.feeding {
		return
	}
	var err error
	if thrown {
		err = p.Store.ThrowPet(time.Now())
	} else {
		err = p.Store.PokePet(time.Now())
	}
	if err != nil {
		p.feedMessage = storelocale.T(storelocale.KeyDesktopPetInteractionWasNotSaved)
		return
	}
	_ = p.rest.wake(nil)
	if thrown {
		p.feedMessage = "抛掷：心情 −9；亲密按同类冷却计入"
	} else {
		p.feedMessage = "点按：心情 +2；亲密按同类冷却计入"
	}
}
func (p *Companion) serveInteraction(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path != "/interaction.json" {
		return false
	}
	p.assetMu.RLock()
	view := p.interaction.view
	p.assetMu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "HEAD" {
		_ = json.NewEncoder(w).Encode(view)
	}
	return true
}
