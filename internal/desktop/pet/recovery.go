package pet

import "github.com/egoist/mygo"

// A display/power transition may invalidate the native transparent surface
// without a web navigation error. Recreate that surface, not just its page.
// Each listener belongs to one window; stale queued work cannot revive a pet
// that was disabled or replaced in the meantime.
func (p *Companion) watchRecovery(w *mygo.Window) {
	pending := false
	recover := func() {
		if pending || p.win != w || !p.ready || !p.Store.DesktopPet().Enabled {
			return
		}
		pending = true
		w.Update(func() {
			pending = false
			if !petRecoveryCurrent(p.win == w, p.ready, p.Store.DesktopPet().Enabled, w.IsDestroyed()) {
				return
			}
			bounds := p.petBounds()
			p.close()
			p.sync()
			if p.win == nil || p.win.IsDestroyed() {
				return
			}
			current := p.petBounds()
			current.X, current.Y = bounds.X, bounds.Y
			if area, ok := motionWorkArea(current); ok {
				p.setMotionPosition(clampMotionPoint(motionPoint{x: float64(bounds.X), y: float64(bounds.Y)}, float64(current.Width), float64(current.Height), area))
			}
		})
	}
	offs := []func(){
		w.Page().OnRenderProcessGone(func(string) { recover() }),
		mygo.Screen.OnDisplaysChanged(recover),
		mygo.Power.OnResume(recover),
		mygo.Power.OnUnlockScreen(recover),
	}
	w.OnClosed(func() {
		for _, off := range offs {
			off()
		}
		offs = nil
	})
}

func petRecoveryCurrent(sameWindow, ready, enabled, destroyed bool) bool {
	return sameWindow && ready && enabled && !destroyed
}
