package session

import (
	"context"
	"time"
	"whaleisle/internal/dsh"
)

type stopSequenceConfig struct {
	client     *dsh.Client
	cancel     context.CancelFunc
	intervalMS int64
}

func (s *Store) clearStopSequenceConfigLocked() {
	if s.stopSequenceConfig.cancel != nil {
		s.stopSequenceConfig.cancel()
	}
	s.stopSequenceConfig = stopSequenceConfig{}
}

// LoadStopSequenceConfig adopts the Host's boot-time fixed-input settings once
// per client binding. Failed/invalid boot reads leave the shortcut disabled;
// the ordinary stop button remains available. A new Host binding retries.
func (s *Store) LoadStopSequenceConfig() {
	s.mu.Lock()
	client := s.client
	if client == nil || s.stopSequenceConfig.client == client {
		s.mu.Unlock()
		return
	}
	s.clearStopSequenceConfigLocked()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	s.stopSequenceConfig = stopSequenceConfig{client: client, cancel: cancel}
	generation := s.composerHostGeneration
	s.mu.Unlock()
	go func() {
		defer cancel()
		interval, err := client.ReadStopSequenceInterval(ctx)
		s.mu.Lock()
		if s.client != client || s.composerHostGeneration != generation || s.stopSequenceConfig.client != client {
			s.mu.Unlock()
			return
		}
		s.stopSequenceConfig.cancel = nil
		if err == nil {
			s.stopSequenceConfig.intervalMS = interval
		} else {
			s.lastAction = "Double Escape unavailable: Host shortcut configuration could not be loaded"
		}
		s.mu.Unlock()
		s.fire()
	}()
}
