package main

import (
	"context"
	"sync"

	"whaleisle/internal/dsh"
)

// requestConnectionReconnect retries only the existing authenticated Mux. It
// neither replaces Client/Stores nor enters the runtime process restart path.
func (s *mainState) requestConnectionReconnect() {
	s.dshMu.Lock()
	mux := s.dshMux
	accepted := mux != nil && mux.RequestReconnect()
	if !accepted {
		s.dshMu.Unlock()
		return
	}
	s.mu.Lock()
	s.dshMuxReady, s.dshMuxLost = false, false
	s.dshBootstrapError = ""
	win := s.win
	s.mu.Unlock()
	s.dshMu.Unlock()
	if win != nil {
		win.Update(func() {})
	}
}

// connectionAttemptState serializes the application's existing $events
// projections across attempts without replacing their Client or Stores.
type connectionAttemptState struct {
	mu      sync.Mutex
	state   *mainState
	host    context.Context
	attempt context.Context
	client  *dsh.Client
	mux     *dsh.Mux
	refresh func()
}

func (s *mainState) newConnectionAttemptState(host context.Context, client *dsh.Client, mux *dsh.Mux, refresh func()) *connectionAttemptState {
	return &connectionAttemptState{state: s, host: host, client: client, mux: mux, refresh: refresh}
}

func (a *connectionAttemptState) ownsHost() bool {
	a.state.dshMu.Lock()
	defer a.state.dshMu.Unlock()
	return a.host.Err() == nil && a.state.dshClient == a.client && a.state.dshMux == a.mux
}

// Call with a.mu held. Terminal Store independently rejects replaced Clients.
func (a *connectionAttemptState) publish(ready, lost bool) bool {
	a.state.dshMu.Lock()
	if a.host.Err() != nil || (ready && (a.attempt == nil || a.attempt.Err() != nil)) || a.state.dshClient != a.client || a.state.dshMux != a.mux {
		a.state.dshMu.Unlock()
		return false
	}
	a.state.mu.Lock()
	a.state.dshMuxReady, a.state.dshMuxLost = ready, lost
	a.state.mu.Unlock()
	a.state.dshMu.Unlock()
	a.state.hostTerminals.Transport(a.client, ready)
	a.refresh()
	return true
}

func (a *connectionAttemptState) begin(ctx context.Context) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if ctx.Err() != nil || !a.ownsHost() {
		return false
	}
	a.attempt = ctx
	return a.publish(false, false)
}

func (a *connectionAttemptState) withCurrent(ctx context.Context, fn func()) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.attempt != ctx || ctx.Err() != nil || !a.ownsHost() {
		return false
	}
	fn()
	return true
}

func (a *connectionAttemptState) ready(ctx context.Context, hydrate func()) {
	a.withCurrent(ctx, func() {
		if a.mux.MarkReady(ctx) && a.publish(true, false) {
			hydrate()
		}
	})
}

func (a *connectionAttemptState) retry(ctx context.Context, err error) {
	a.withCurrent(ctx, func() {
		if a.publish(false, true) {
			a.mux.GenerationLost(ctx, err)
		}
	})
}

func (a *connectionAttemptState) lost() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.attempt = nil
	if a.ownsHost() {
		a.publish(false, true)
	}
}

func (a *connectionAttemptState) bootstrap(err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.state.dshMu.Lock()
	if a.host.Err() != nil || a.state.dshClient != a.client || a.state.dshMux != a.mux {
		a.state.dshMu.Unlock()
		return
	}
	a.state.mu.Lock()
	a.state.dshMuxReady = false
	a.state.dshMuxLost = err != nil
	a.state.dshBootstrapError = ""
	if err != nil {
		a.state.dshBootstrapError = err.Error()
	}
	a.state.mu.Unlock()
	a.state.dshMu.Unlock()
	a.state.hostTerminals.Transport(a.client, false)
	a.refresh()
}
