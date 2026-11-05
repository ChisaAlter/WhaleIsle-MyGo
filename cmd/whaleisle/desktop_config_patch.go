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
	leaseOwned := true
	defer func() {
		if leaseOwned {
			release()
		}
	}()
	restart := false
	var features desktopConfigFeaturePatch
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
				features.dshbot = &desktopConfigFeatureValue{previous: c.DshbotEnabled, expected: value.(bool)}
				c.DshbotEnabled, restart = features.dshbot.expected, true
			case "whaleAssistantEnabled":
				features.assistant = &desktopConfigFeatureValue{previous: c.WhaleAssistantEnabled, expected: value.(bool)}
				c.WhaleAssistantEnabled, restart = features.assistant.expected, true
			case "remoteWorkspaceEnabled":
				features.remote = &desktopConfigFeatureValue{previous: c.RemoteWorkspaceEnabled, expected: value.(bool)}
				c.RemoteWorkspaceEnabled, restart = features.remote.expected, true
			case "harnessAutoRestart":
				c.HarnessAutoRestart = value.(bool)
			case "harnessRestartMaxAttempts":
				c.HarnessRestartMaxAttempts = int(value.(float64))
			case "harnessRestartBaseDelayMs":
				c.HarnessRestartBaseDelayMs = int(value.(float64))
			}
		}
	})
	if err != nil {
		return nil, err
	}
	// Preserve write/effect ordering across concurrent desktop requests.
	saved := store.Get()
	err = s.applyDesktopConfigEffects(saved, patch, win)
	if err != nil {
		// Desktop side effects keep their existing partial-save contract, but a
		// feature change cannot be accepted without its protected restart.
		if restart {
			err = errors.Join(err, s.rollbackDesktopConfigFeatures(store, features))
		}
		return nil, fmt.Errorf("settings saved but could not be fully applied: %w", err)
	}
	restartRequested := false
	if restart {
		if sup != nil {
			// Never await the requesting Host's restart in its bridge handler.
			// The detached result owns this lease until acceptance or rollback.
			restartRequested = s.restartUnlessImportWithResult(sup, true, func(did bool, restartErr error) {
				defer release()
				s.completeDesktopConfigFeatures(store, features, win, did, restartErr)
			})
		}
		if restartRequested {
			leaseOwned = false
		} else {
			s.completeDesktopConfigFeatures(store, features, win, false, errors.New(lifecycleText("当前无法开始功能重启。", "The feature restart could not be started.")))
			saved = store.Get()
		}
	}
	return map[string]any{"ok": true, "config": desktopPublicConfig(saved), "restarting": restartRequested, "restartRequired": restart && !restartRequested}, nil
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

// Keep both desktop control responses on the same existing public whitelist.
func desktopPublicConfig(cfg config.Config) map[string]any {
	return map[string]any{
		"workspace":                 cfg.Workspace,
		"host":                      cfg.Host,
		"port":                      cfg.Port,
		"theme":                     cfg.Theme,
		"locale":                    cfg.Locale,
		"closeToTray":               cfg.CloseToTray,
		"openAtLogin":               cfg.OpenAtLogin,
		"dshbotEnabled":             cfg.DshbotEnabled,
		"whaleAssistantEnabled":     cfg.WhaleAssistantEnabled,
		"remoteWorkspaceEnabled":    cfg.RemoteWorkspaceEnabled,
		"harnessAutoRestart":        cfg.HarnessAutoRestart,
		"harnessRestartMaxAttempts": cfg.HarnessRestartMaxAttempts,
		"harnessRestartBaseDelayMs": cfg.HarnessRestartBaseDelayMs,
	}
}
