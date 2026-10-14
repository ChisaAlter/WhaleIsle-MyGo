package main

import (
	"time"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
)

// refreshPetGrowth is an explicit native action. The renderer cannot choose a
// path/runtime. Only this app's already-authorized DSH home is scanned; no
// account credential, model invocation or external usage source is involved.
func (s *mainState) refreshPetGrowth() {
	s.mu.Lock()
	p, cfg, win, generation := s.desktopPet, s.cfgStore, s.win, s.bootStatus.Generation
	quitting := s.quitting
	s.mu.Unlock()
	if p == nil || cfg == nil || win == nil || quitting {
		return
	}
	ctx, finish := p.StartGrowth()
	if ctx == nil {
		return
	}
	node, home := cfg.Get().NodeBin, paths.DshHome()
	go func() {
		samples, err := harness.ScanPetUsage(ctx, node, home)
		win.Update(func() {
			s.mu.Lock()
			current := !s.quitting && s.desktopPet == p && s.cfgStore == cfg && s.win == win && s.bootStatus.Generation == generation
			s.mu.Unlock()
			if !current || cfg.Get().NodeBin != node || paths.DshHome() != home {
				finish("运行环境已变化，请重试 / Runtime changed", nil)
				return
			}
			if ctx.Err() != nil || err != nil {
				finish("用量读取未完成，账本保持不变 / Scan unavailable; retry", nil)
				return
			}
			// Closing/replacing the pet invalidates the guarded commit before
			// any storage writes; success is published only after atomic save.
			finish("粮碗已刷新 / Local food refreshed", func() error {
				return cfg.RefreshPetGrowth(samples, time.Now())
			})
		})
	}()
}
