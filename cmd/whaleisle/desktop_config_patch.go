package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/egoist/mygo"
	"whaleisle/internal/config"
	"whaleisle/internal/desktop/importer"
	"whaleisle/internal/paths"
	"whaleisle/internal/ui/i18n"
)

func (s *mainState) applyDesktopConfigPatch(_ context.Context, patch map[string]any) (any, error) {
	s.mu.Lock()
	store := s.cfgStore
	sup := s.sup
	win := s.win
	busy := s.launcherImportBusy || s.quitting
	s.mu.Unlock()
	if store == nil {
		return nil, errors.New("desktop config store unavailable")
	}
	if busy {
		return nil, errors.New("desktop maintenance is in progress")
	}
	for key, value := range patch {
		switch key {
		case "theme", "locale":
			if _, ok := value.(string); !ok {
				return nil, fmt.Errorf("%s must be a string", key)
			}
		case "closeToTray", "openAtLogin", "dshbotEnabled", "whaleAssistantEnabled", "remoteWorkspaceEnabled", "harnessAutoRestart":
			if _, ok := value.(bool); !ok {
				return nil, fmt.Errorf("%s must be a boolean", key)
			}
		case "harnessRestartMaxAttempts", "harnessRestartBaseDelayMs":
			if _, ok := value.(float64); !ok {
				return nil, fmt.Errorf("%s must be a number", key)
			}
		default:
			return nil, fmt.Errorf("config field is not desktop-writable: %s", key)
		}
	}
	release, err := importer.AcquireExternalMaintenance(paths.DshHome())
	if err != nil {
		return nil, err
	}
	restart := false
	err = store.Update(func(c *config.Config) {
		for key, value := range patch {
			switch key {
			case "theme":
				c.Theme = value.(string)
			case "locale":
				c.Locale = value.(string)
			case "closeToTray":
				c.CloseToTray = value.(bool)
			case "openAtLogin":
				c.OpenAtLogin = value.(bool)
			case "dshbotEnabled":
				c.DshbotEnabled, restart = value.(bool), true
			case "whaleAssistantEnabled":
				c.WhaleAssistantEnabled, restart = value.(bool), true
			case "remoteWorkspaceEnabled":
				c.RemoteWorkspaceEnabled, restart = value.(bool), true
			case "harnessAutoRestart":
				c.HarnessAutoRestart = value.(bool)
			case "harnessRestartMaxAttempts":
				c.HarnessRestartMaxAttempts = int(value.(float64))
			case "harnessRestartBaseDelayMs":
				c.HarnessRestartBaseDelayMs = int(value.(float64))
			}
		}
	})
	release()
	if err != nil {
		return nil, err
	}
	if err := s.applyDesktopConfigEffects(store.Get(), patch, win); err != nil {
		return nil, fmt.Errorf("settings saved but could not be fully applied: %w", err)
	}
	restartRequested := false
	if restart && sup != nil {
		s.syncHarnessFromConfig()
		restartRequested = s.restartUnlessImport(sup)
	}
	return map[string]any{"ok": true, "restarting": restartRequested, "restartRequired": restart && !restartRequested}, nil
}

// Apply only patched desktop fields. Other settings drafts remain owned by
// their explicit save operation, as in the native settings controls.
func (s *mainState) applyDesktopConfigEffects(cfg config.Config, patch map[string]any, win *mygo.Window) error {
	var loginErr error
	mygo.RunOnMain(func() {
		s.mu.Lock()
		if _, changed := patch["theme"]; changed {
			s.themeID = cfg.Theme
			s.desktopDraft.theme = cfg.Theme
		}
		if _, changed := patch["locale"]; changed {
			s.desktopDraft.locale = cfg.Locale
		}
		if _, changed := patch["closeToTray"]; changed {
			s.closeToTray = cfg.CloseToTray
			s.desktopDraft.closeToTray = cfg.CloseToTray
		}
		if _, changed := patch["openAtLogin"]; changed {
			s.desktopDraft.openAtLogin = cfg.OpenAtLogin
		}
		s.mu.Unlock()
		if _, changed := patch["locale"]; changed {
			i18n.SetLocale(cfg.Locale)
		}
		if _, changed := patch["theme"]; changed {
			applyAppearance(cfg.Theme)
		}
		if _, changed := patch["openAtLogin"]; changed {
			loginErr = desktopSystemIntegration(func() error { return mygo.App.SetOpenAtLogin(cfg.OpenAtLogin) })
		}
		if win != nil {
			win.Invalidate()
		}
	})
	return loginErr
}
