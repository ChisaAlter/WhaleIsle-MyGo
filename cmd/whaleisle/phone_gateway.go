package main

import (
	"errors"
	"whaleisle/internal/config"
	"whaleisle/internal/credentials"
	"whaleisle/internal/paths"
	"whaleisle/internal/ui/i18n"
)

// chisacode-v2 pairing is owned by the original runtime. Its legacy settings
// disclosure only offers removal of an old desktop relay credential.
func (s *mainState) clearLegacyRelayToken(value string) error {
	if value != "" {
		return errors.New(i18n.T(i18n.KeyCmdLegacyRelayOnlyClear))
	}
	s.phoneOp.Lock()
	defer s.phoneOp.Unlock()
	creds, err := credentials.Load(paths.UserDataDir())
	if err != nil {
		return err
	}
	creds.RemoteRelayToken = ""
	if err = credentials.SaveSharedProfile(paths.UserDataDir(), creds); err != nil {
		return err
	}
	s.mu.Lock()
	s.hasRemoteRelayToken = false
	s.credsDraft.remoteRelayToken = ""
	s.mu.Unlock()
	return nil
}

func (s *mainState) savePhoneGateway(expected, p config.RemoteGatewayPreferences) error {
	if s.cfgStore == nil {
		return errors.New(i18n.T(i18n.KeyCmdRemoteConfigUnavailable))
	}
	if err := s.cfgStore.SaveRemoteGateway(expected, p); err != nil {
		return err
	}
	s.applySavedPhoneGateway()
	return nil
}

func (s *mainState) savePhoneEnabled(enabled bool) error {
	if s.cfgStore == nil {
		return errors.New(i18n.T(i18n.KeyCmdRemoteConfigUnavailable))
	}
	if err := s.cfgStore.Update(func(c *config.Config) { c.RemoteEnabled = enabled }); err != nil {
		return err
	}
	s.applySavedPhoneGateway()
	return nil
}
