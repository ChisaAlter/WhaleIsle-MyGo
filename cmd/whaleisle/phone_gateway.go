package main

import (
	"errors"
	"whaleisle/internal/config"
)

func (s *mainState) savePhoneGateway(expected, p config.RemoteGatewayPreferences) error {
	if s.cfgStore == nil {
		return errors.New("remote configuration unavailable")
	}
	if err := s.cfgStore.SaveRemoteGateway(expected, p); err != nil {
		return err
	}
	// The old Git grant must not survive a topology change. Refresh serializes
	// lifecycle changes through phoneOp and reads the latest committed config.
	s.revokePhoneGit()
	s.mu.Lock()
	s.phoneActionMsg = "网关设置已保存，正在应用 / Gateway settings saved; applying"
	s.mu.Unlock()
	go func() {
		s.stopPhoneRuntime()
		s.refreshPhoneRemote()
		s.mu.Lock()
		win := s.win
		s.mu.Unlock()
		if win != nil {
			win.Update(func() {})
		}
	}()
	return nil
}
