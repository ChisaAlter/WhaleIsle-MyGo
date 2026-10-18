package pet

import (
	"github.com/egoist/mygo"
	"time"
)

// SetCareRuntime is called on the UI thread for the current Host generation.
// Local care remains usable offline; switching Host invalidates the previous
// timer before a fresh idle period starts. No Host/model request is performed.
func (p *Companion) SetCareRuntime(owner string, generation int) {
	c := &p.restClock
	if c.runtimeKnown && c.runtimeOwner == owner && c.runtimeGeneration == generation {
		return
	}
	c.runtimeOwner, c.runtimeGeneration, c.runtimeKnown = owner, generation, true
	p.startRestClock()
}

func (p *Companion) startRestClock() {
	p.stopRestClock()
	w := p.win
	if !p.restClockCurrent(w) {
		return
	}
	p.touchRest(time.Now())
	p.armRestClock(w, p.restClock.generation)
}

func (p *Companion) stopRestClock() { p.restClock.stop() }

func (p *Companion) restClockCurrent(w *mygo.Window) bool {
	return w != nil && p.win == w && !w.IsDestroyed() && w.IsVisible() && p.ready && p.Store != nil && p.Store.DesktopPet().Enabled
}

func (p *Companion) touchRest(now time.Time) {
	p.assetMu.Lock()
	p.rest.touch(now)
	p.assetMu.Unlock()
}

func (p *Companion) armRestClock(w *mygo.Window, generation uint64) {
	if p.restClock.generation != generation || !p.restClockCurrent(w) {
		return
	}
	timer := time.AfterFunc(time.Second, func() {
		// This goroutine reads no mutable Companion fields.
		w.Update(func() {
			if p.restClock.generation != generation || !p.restClockCurrent(w) {
				return
			}
			p.restClock.cancel = nil
			now := time.Now()
			p.assetMu.Lock()
			// An open compact chat, modal/action, drag or active task must not have
			// a nap inserted into it. Suppressed slots never become an overdue nap.
			blocked := p.quickChatOpen || p.feeding || p.chat.status.Busy || p.look.status.Busy || (p.growth.status.Busy && !p.growthAutomatic) || p.activityTaskLocked().State == "running" || p.motion.manualPending || now.Before(p.motion.contactUntil) || !w.IsVisible()
			p.rest.advance(now, blocked)
			p.assetMu.Unlock()
			p.armRestClock(w, generation)
		})
	})
	p.restClock.cancel = func() { timer.Stop() }
}
