package main

import (
	"context"
	"whaleisle/internal/harness"
)

// Explicit operator action; the shared path checks import and Host tasks before
// stopping anything. Cancellation leaves the saved change pending.
func (s *mainState) restartMarketPlugins() {
	s.mu.Lock()
	sup, win := s.sup, s.win
	allowed := s.marketActivation.Pending && s.marketWorker == nil && !s.marketInstallBusy && !s.taskRestartPending && !s.quitting
	s.mu.Unlock()
	if !allowed || sup == nil {
		return
	}
	s.syncHarnessFromConfig()
	s.restartUnlessImport(sup)
	if win != nil {
		win.Update(func() {})
	}
}

// Called only after the status snapshot has been accepted by refreshBoot.
func (s *mainState) observeMarketActivation(st harness.Status) {
	s.mu.Lock()
	// OnChange callbacks run on goroutines. A delayed Ready from the previous
	// process must not acknowledge a pending change after a failed restart.
	if s.sup != nil {
		actual := s.sup.Status()
		if actual.Generation != st.Generation || actual.State != st.State {
			s.mu.Unlock()
			return
		}
	}
	completed := s.marketActivation.Observe(st.Generation, st.State.String())
	store := s.market
	s.mu.Unlock()
	if completed && store != nil {
		store.Refresh(context.Background())
	}
}
