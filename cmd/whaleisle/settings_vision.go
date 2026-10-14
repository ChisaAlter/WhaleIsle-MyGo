package main

import (
	"context"
	"whaleisle/internal/ui/settings"
)

func (s *mainState) refreshVisionSettings() {
	s.mu.Lock()
	store := s.settings
	s.mu.Unlock()
	if store != nil {
		store.RefreshVision(context.Background())
	}
}

func (s *mainState) visionSettingsAccess() *settings.VisionAccess {
	// Called while constructing the settings frame, like providerProfilesAccess.
	store := s.settings
	if store == nil {
		return nil
	}
	return &settings.VisionAccess{Snapshot: store.VisionSnapshot(), OnRefresh: s.refreshVisionSettings,
		OnSelect: func(provider, model string) { store.SaveVision(context.Background(), provider, model) },
	}
}
