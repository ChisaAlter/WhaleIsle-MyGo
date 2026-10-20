package pet

import (
	"encoding/json"
	"io"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"time"

	"github.com/egoist/mygo"
	"whaleisle/internal/config"
	"whaleisle/internal/store/storelocale"
)

// motionState belongs exclusively to the UI thread, including all callbacks.
// Timers only enqueue generation-checked work through Window.Update. Nothing
// in ServeHTTP reads this state and the page has no position/move capability.
type motionState struct {
	window         *mygo.Window
	generation     uint64
	timer          *time.Timer
	offMove        func()
	offFocus       func()
	offDisplays    func()
	legacyChecked  bool
	lastContact    time.Time
	contactUntil   time.Time
	nextAt         time.Time
	glideAt        time.Time
	glide          *wanderGlide
	area           motionRect
	facing         int
	moves          motionProvenance
	manualPending  bool
	manualPosition motionPoint
	manualSaveAt   time.Time
}

// startMotion must run after p.win = w and the final window size is applied.
// The companion must NOT have a StateKey: MyGo otherwise records every roam
// position when it closes, overwriting the user's deliberate anchor.
func (p *Companion) startMotion(w *mygo.Window) {
	p.stopMotion()
	if w == nil || w.IsDestroyed() || p.win != w || p.Store == nil {
		return
	}
	m := &p.motion
	m.window = w
	m.facing = 1
	m.lastContact = time.Now()
	cfg := p.Store.DesktopPet()
	bounds := w.Bounds()
	position := motionPoint{x: float64(bounds.X), y: float64(bounds.Y)}
	migrate := false
	if cfg.Position != nil {
		position = motionPoint{x: float64(cfg.Position.X), y: float64(cfg.Position.Y)}
	} else if !m.legacyChecked {
		m.legacyChecked = true
		if old := legacyMotionAnchor(); old != nil {
			position = motionPoint{x: float64(old.X), y: float64(old.Y)}
			migrate = true
		}
	}
	// Resolve a disconnected monitor against the actual connected displays.
	requested := bounds
	requested.X = int(motionClamp(position.x, -motionCoordinateLimit, motionCoordinateLimit))
	requested.Y = int(motionClamp(position.y, -motionCoordinateLimit, motionCoordinateLimit))
	if area, ok := motionWorkArea(requested); ok {
		position = clampMotionPoint(position, float64(bounds.Width), float64(bounds.Height), area)
		p.setMotionPosition(position)
		if migrate {
			p.saveMotionAnchor(position)
		}
	}
	m.moves.expect(motionWindowPoint(w))
	m.offMove = w.OnMove(func() {
		if p.motionCurrent(w) {
			p.motionMoved(time.Now())
		}
	})
	m.offFocus = w.OnFocus(func() {
		if p.motionCurrent(w) {
			p.motionContact()
		}
	})
	m.offDisplays = mygo.Screen.OnDisplaysChanged(func() {
		if !p.motionCurrent(w) {
			return
		}
		// Display removal is an automatic relocation, never a new anchor.
		m.glide, m.manualPending = nil, false
		bounds := w.Bounds()
		if area, ok := motionWorkArea(bounds); ok {
			p.setMotionPosition(clampMotionPoint(motionWindowPoint(w), float64(bounds.Width), float64(bounds.Height), area))
		}
		m.nextAt = time.Now().Add(wanderInterval(p.Store.DesktopPet().Behavior.Activity, rand.Float64()))
	})
	m.nextAt = time.Now().Add(wanderInterval(cfg.Behavior.Activity, rand.Float64()))
	p.armMotion(250 * time.Millisecond)
}

// stopMotion is idempotent and runs before Destroy and from OnClosed. Pending
// drag samples are discarded instead of sampling an automatic close position.
func (p *Companion) stopMotion() {
	m := &p.motion
	m.generation++
	if m.timer != nil {
		m.timer.Stop()
	}
	if m.offMove != nil {
		m.offMove()
	}
	if m.offFocus != nil {
		m.offFocus()
	}
	if m.offDisplays != nil {
		m.offDisplays()
	}
	checked, generation := m.legacyChecked, m.generation
	*m = motionState{legacyChecked: checked, generation: generation}
}

func (p *Companion) motionCurrent(w *mygo.Window) bool {
	return w != nil && p.motion.window == w && p.win == w && !w.IsDestroyed()
}

// motionContact is the exact local /motion-contact action. It takes no page
// arguments. Pointer/focus contact and handled user actions call it; a page
// heartbeat while hovered extends the lease without granting native moves.
func (p *Companion) motionContact() {
	m := &p.motion
	if !p.motionCurrent(m.window) {
		return
	}
	now := time.Now()
	p.touchRest(now)
	m.lastContact = now
	m.contactUntil = now.Add(2 * time.Second)
	if m.glide != nil {
		m.glide = nil
		m.nextAt = now.Add(wanderInterval(p.Store.DesktopPet().Behavior.Activity, rand.Float64()))
	}
}

func (p *Companion) armMotion(delay time.Duration) {
	m := &p.motion
	w, generation := m.window, m.generation
	if !p.motionCurrent(w) {
		return
	}
	if m.timer != nil {
		m.timer.Stop()
	}
	delay = max(16*time.Millisecond, min(delay, time.Second))
	m.timer = time.AfterFunc(delay, func() {
		// No mutable Companion or motion fields are read on this goroutine.
		w.Update(func() {
			if p.motion.generation == generation && p.motionCurrent(w) {
				p.motion.timer = nil
				p.tickMotion(time.Now())
			}
		})
	})
}

func (p *Companion) tickMotion(now time.Time) {
	m := &p.motion
	if !p.motionCurrent(m.window) || p.Store == nil {
		return
	}
	cfg := p.Store.DesktopPet()
	if !cfg.Enabled || !p.ready {
		p.stopMotion()
		return
	}
	if m.manualPending && !now.Before(m.manualSaveAt) {
		// Persist the classified manual snapshot, NEVER current Bounds.
		position := m.manualPosition
		m.manualPending = false
		p.saveMotionAnchor(position)
	}
	p.assetMu.RLock()
	sleeping := p.rest.sleeping
	busy := p.quickChatOpen || p.feeding || p.chat.status.Busy || p.look.status.Busy || p.growth.status.Busy || p.activityTaskLocked().State == "running"
	p.assetMu.RUnlock()
	paused := wanderPaused(cfg.Behavior.Wander, sleeping, busy, cfg.Behavior.PowerSave, m.lastContact, now)
	if paused || !m.window.IsVisible() || now.Before(m.contactUntil) || m.manualPending {
		if m.glide != nil || m.nextAt.IsZero() || !now.Before(m.nextAt) {
			m.glide = nil
			m.nextAt = now.Add(wanderInterval(cfg.Behavior.Activity, rand.Float64()))
		}
		p.armMotion(250 * time.Millisecond)
		return
	}
	if m.glide != nil {
		bounds := m.window.Bounds()
		area, ok := motionWorkArea(bounds)
		if !ok || area != m.area {
			m.glide = nil
			m.nextAt = now.Add(wanderInterval(cfg.Behavior.Activity, rand.Float64()))
			if ok {
				p.setMotionPosition(clampMotionPoint(motionWindowPoint(m.window), float64(bounds.Width), float64(bounds.Height), area))
			}
		} else {
			position, done := m.glide.position(now.Sub(m.glideAt).Seconds())
			p.setMotionPosition(clampMotionPoint(position, float64(bounds.Width), float64(bounds.Height), area))
			if done {
				m.glide = nil
				m.nextAt = now.Add(wanderInterval(cfg.Behavior.Activity, rand.Float64()))
			}
		}
	} else if !now.Before(m.nextAt) {
		bounds := m.window.Bounds()
		area, ok := motionWorkArea(bounds)
		if ok {
			from := motionWindowPoint(m.window)
			target := wanderTarget(from, float64(bounds.Width), float64(bounds.Height), area, m.facing, cfg.Behavior.Scale, rand.Float64)
			if math.Abs(target.x-from.x) > 2 || math.Abs(target.y-from.y) > 2 {
				glide := newWanderGlide(from, target, wanderSpeed(cfg.Behavior.Activity, cfg.Behavior.Scale, rand.Float64()))
				m.glide, m.glideAt, m.area = &glide, now, area
				if target.x != from.x {
					m.facing = 1
					if target.x < from.x {
						m.facing = -1
					}
				}
			}
		}
		if m.glide == nil {
			m.nextAt = now.Add(wanderInterval(cfg.Behavior.Activity, rand.Float64()))
		}
	}
	if m.glide != nil {
		p.armMotion(time.Second / 30)
	} else {
		p.armMotion(250 * time.Millisecond)
	}
}

func (p *Companion) setMotionPosition(position motionPoint) {
	m := &p.motion
	if !p.motionCurrent(m.window) {
		return
	}
	position.x = math.Round(motionClamp(position.x, -motionCoordinateLimit, motionCoordinateLimit))
	position.y = math.Round(motionClamp(position.y, -motionCoordinateLimit, motionCoordinateLimit))
	if position == motionWindowPoint(m.window) {
		return
	}
	m.moves.expect(position)
	m.moves.writing = true
	m.window.SetPosition(int(position.x), int(position.y))
	// Record the actual native result too: some backends constrain positions.
	m.moves.expect(motionWindowPoint(m.window))
	m.moves.writing = false
}

func (p *Companion) motionMoved(now time.Time) {
	m := &p.motion
	position := motionWindowPoint(m.window)
	contact := now.Before(m.contactUntil) || m.manualPending
	if !m.moves.manual(position, contact) {
		return
	}
	// Native app-region dragging may consume page pointer events. Once a real
	// contacted move begins, each native move renews the interaction lease.
	p.touchRest(now)
	m.lastContact = now
	m.contactUntil = now.Add(2 * time.Second)
	m.glide = nil
	m.nextAt = now.Add(wanderInterval(p.Store.DesktopPet().Behavior.Activity, rand.Float64()))
	bounds := m.window.Bounds()
	if area, ok := motionWorkArea(bounds); ok {
		m.manualPosition = clampMotionPoint(position, float64(bounds.Width), float64(bounds.Height), area)
		m.manualPending = true
		m.manualSaveAt = now.Add(600 * time.Millisecond)
	}
}

func (p *Companion) saveMotionAnchor(position motionPoint) {
	if p.Store == nil {
		return
	}
	cfg := p.Store.DesktopPet()
	if !cfg.Enabled {
		return
	}
	anchor := config.PetPosition{
		X: int(math.Round(motionClamp(position.x, -motionCoordinateLimit, motionCoordinateLimit))),
		Y: int(math.Round(motionClamp(position.y, -motionCoordinateLimit, motionCoordinateLimit))),
	}
	if cfg.Position != nil && *cfg.Position == anchor {
		return
	}
	cfg.Position = &anchor
	if err := p.Store.SaveDesktopPet(cfg); err != nil {
		// Preserve the store's conflict/error semantics; never reload and retry
		// a stale whole-config write or continuously retry an idle timer.
		p.Error = storelocale.T(storelocale.KeyDesktopPetPetPositionWasNotSaved) + err.Error()
	}
}

func motionWindowPoint(w *mygo.Window) motionPoint {
	bounds := w.Bounds()
	return motionPoint{x: float64(bounds.X), y: float64(bounds.Y)}
}

func motionWorkArea(bounds mygo.Rectangle) (motionRect, bool) {
	display := mygo.Screen.DisplayMatching(bounds)
	area := display.WorkArea
	if area.Width <= 0 || area.Height <= 0 {
		return motionRect{}, false
	}
	return boundedMotionRect(motionRect{x: float64(area.X), y: float64(area.Y), width: float64(area.Width), height: float64(area.Height)}), true
}

// Read the previous MyGo anchor once, only when the new position is absent.
// Do not migrate Electron pet/live2dPet coordinates, and never write MyGo's
// shared window-state.json. Its bounds schema is pinned in window_state.go.
func legacyMotionAnchor() *config.PetPosition {
	dir, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return nil
	}
	file, err := os.Open(filepath.Join(dir, "window-state.json"))
	if err != nil {
		return nil
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > 1<<20 {
		return nil
	}
	data, err := io.ReadAll(io.LimitReader(file, (1<<20)+1))
	if err != nil || len(data) > 1<<20 {
		return nil
	}
	var windows map[string]json.RawMessage
	if json.Unmarshal(data, &windows) != nil {
		return nil
	}
	var old struct {
		X      *int `json:"x"`
		Y      *int `json:"y"`
		Width  int  `json:"width"`
		Height int  `json:"height"`
	}
	if json.Unmarshal(windows["mygo-desktop-pet"], &old) != nil || old.X == nil || old.Y == nil || old.Width <= 0 || old.Height <= 0 || old.Width > motionCoordinateLimit || old.Height > motionCoordinateLimit {
		return nil
	}
	if *old.X < -motionCoordinateLimit || *old.X > motionCoordinateLimit || *old.Y < -motionCoordinateLimit || *old.Y > motionCoordinateLimit {
		return nil
	}
	return &config.PetPosition{X: *old.X, Y: *old.Y}
}
