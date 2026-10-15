package main

import (
	"time"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
)

// All entry points share one cancellable scan. Only the explicit feed entry
// may consume; minute/show refreshes only settle the original local food ledger.
func (s *mainState) refreshPetGrowth()     { s.refreshPetGrowthAction(false, false) }
func (s *mainState) autoRefreshPetGrowth() { s.refreshPetGrowthAction(false, true) }
func (s *mainState) feedPetTokens()        { s.refreshPetGrowthAction(true, false) }
func (s *mainState) refreshPetGrowthAction(wantsFeed, automatic bool) {
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	s.mu.Lock()
	p, cfg, win, generation := s.desktopPet, s.cfgStore, s.win, s.bootStatus.Generation
	quitting, sup := s.quitting, s.sup
	s.mu.Unlock()
	if p == nil || cfg == nil || win == nil || quitting {
		return
	}
	ctx, finish := p.StartGrowthRefresh(automatic)
	if ctx == nil {
		return
	}
	node, home := cfg.Get().NodeBin, paths.DshHome()
	go func() {
		samples, err := harness.ScanPetUsage(ctx, node, home)
		win.Update(func() {
			s.dshMu.Lock()
			currentClient := s.dshClient
			s.dshMu.Unlock()
			s.mu.Lock()
			current := !s.quitting && s.desktopPet == p && s.cfgStore == cfg && s.win == win && s.bootStatus.Generation == generation && s.sup == sup && currentClient == client
			s.mu.Unlock()
			if !current || cfg.Get().NodeBin != node || paths.DshHome() != home {
				finish("运行环境已变化，请重试 / Runtime changed", nil)
				return
			}
			finishPetGrowthRefresh(ctx, err, wantsFeed, finish, func() error {
				return cfg.RefreshPetGrowth(samples, time.Now())
			}, p.FeedTokensAfterRefresh)
		})
	}()
}
