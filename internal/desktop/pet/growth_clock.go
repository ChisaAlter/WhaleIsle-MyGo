package pet

import (
	"github.com/egoist/mygo"
	"time"
)

// The owner includes native Supervisor/client identity and fixed data/runtime
// paths. It is never exposed to the page, and no page-selected path is accepted.
func (p *Companion) SetGrowthRuntime(owner string, generation int) {
	c := &p.growthClock
	if c.runtimeKnown && c.runtimeOwner == owner && c.runtimeGeneration == generation {
		return
	}
	c.runtimeOwner, c.runtimeGeneration, c.runtimeKnown = owner, generation, true
	p.startGrowthClock()
}
func (p *Companion) stopGrowthClock() {
	p.growthClock.stop()
	p.assetMu.Lock()
	p.growth.close()
	p.growthAutomatic = false
	p.assetMu.Unlock()
}
func (p *Companion) startGrowthClock() {
	p.stopGrowthClock()
	w := p.win
	if !p.restClockCurrent(w) {
		return
	}
	p.armGrowthClock(w, p.growthClock.generation)
	// The existing native card is always visible: show/reload is the original
	// open-panel refresh. Neither this path nor the minute timer can consume food.
	if p.OnAutoRefreshGrowth != nil {
		p.OnAutoRefreshGrowth()
	}
}
func (p *Companion) armGrowthClock(w *mygo.Window, generation uint64) {
	if p.growthClock.generation != generation || !p.restClockCurrent(w) {
		return
	}
	timer := time.AfterFunc(time.Minute, func() {
		w.Update(func() {
			if p.growthClock.generation != generation || !p.restClockCurrent(w) {
				return
			}
			p.growthClock.cancel = nil
			if p.OnAutoRefreshGrowth != nil {
				p.OnAutoRefreshGrowth()
			}
			p.armGrowthClock(w, generation)
		})
	})
	p.growthClock.cancel = func() { timer.Stop() }
}
