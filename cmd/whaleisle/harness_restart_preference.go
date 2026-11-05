package main

import (
	"errors"
	"strconv"
	"whaleisle/internal/config"
)

// The General recovery controls submit one committed field, not the diagnostics
// form. The supervisor reads this committed policy at its next retry decision.
func (s *mainState) setHarnessRestartPreference(key, value string) (config.RestartPreferences, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cfgStore == nil {
		return config.RestartPreferences{}, errors.New(lifecycleText("本地配置尚不可用。", "Local configuration is unavailable."))
	}
	store := s.cfgStore
	err := store.UpdateChecked(func(c *config.Config) error {
		current := config.RestartPreferencesFrom(*c)
		enabled, attempts, delay := current.AutoRestart, strconv.Itoa(current.MaxAttempts), strconv.Itoa(current.BaseDelayMs)
		switch key {
		case "harnessAutoRestart":
			if value != "true" && value != "false" {
				return errors.New("invalid Harness auto-restart value")
			}
			enabled = value == "true"
		case "harnessRestartMaxAttempts":
			attempts = value
		case "harnessRestartBaseDelayMs":
			delay = value
		default:
			return errors.New("unknown Harness recovery preference")
		}
		p, err := config.ParseRestartPreferences(enabled, attempts, delay)
		if err != nil {
			return err
		}
		p.Apply(c)
		return nil
	})
	accepted := config.RestartPreferencesFrom(store.Get())
	if err == nil {
		s.desktopDraft.harnessAutoRestart = accepted.AutoRestart
		s.desktopDraft.harnessRestartMaxAttempts = strconv.Itoa(accepted.MaxAttempts)
		s.desktopDraft.harnessRestartBaseDelayMs = strconv.Itoa(accepted.BaseDelayMs)
	}
	return accepted, err
}
