package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo"

	"whaleisle/internal/config"
	"whaleisle/internal/desktop/deeplink"
)

// The phone shell's host routes (native remote-shell.js:84-96) run on the
// desktop. openSettings reuses the whaleisle://settings deeplink path, and
// saveConfig merges a whitelisted patch into the app config store. Neither
// echoes credentials.

var remoteConfigPatchFields = map[string]bool{
	"remoteEnabled":     true,
	"remoteMode":        true,
	"remotePort":        true,
	"remoteBindAddress": true,
	"remoteLanTls":      true,
	"remoteAppBaseUrl":  true,
	"locale":            true,
	"theme":             true,
}

// applyRemoteConfigPatch merges the fields the phone may change. Unknown keys are
// rejected so a patch can never touch tokens, credentials or unrelated settings.
func applyRemoteConfigPatch(c *config.Config, patch map[string]any) error {
	for key, value := range patch {
		if !remoteConfigPatchFields[key] {
			return fmt.Errorf("unsupported config field %q", key)
		}
		switch key {
		case "remoteEnabled":
			b, ok := value.(bool)
			if !ok {
				return errors.New("remoteEnabled must be a boolean")
			}
			c.RemoteEnabled = b
		case "remoteLanTls":
			b, ok := value.(bool)
			if !ok {
				return errors.New("remoteLanTls must be a boolean")
			}
			c.RemoteLanTls = b
		case "remotePort":
			switch n := value.(type) {
			case float64:
				c.RemotePort = int(n)
			case int:
				c.RemotePort = n
			default:
				return errors.New("remotePort must be a number")
			}
		case "remoteMode", "remoteBindAddress", "remoteAppBaseUrl", "locale", "theme":
			s, ok := value.(string)
			if !ok || len(s) > 512 {
				return fmt.Errorf("%s must be a string", key)
			}
			switch key {
			case "remoteMode":
				c.RemoteMode = strings.TrimSpace(s)
			case "remoteBindAddress":
				c.RemoteBindAddress = strings.TrimSpace(s)
			case "remoteAppBaseUrl":
				c.RemoteAppBaseUrl = strings.TrimSpace(s)
			case "locale":
				c.Locale = strings.TrimSpace(s)
			case "theme":
				c.Theme = strings.TrimSpace(s)
			}
		}
	}
	return nil
}

// redactedRemoteConfig is the saveConfig/getConfig result: the remote surface a
// phone may see, never tokens or credentials.
func redactedRemoteConfig(c config.Config) map[string]any {
	return map[string]any{
		"remoteEnabled":     c.RemoteEnabled,
		"remoteMode":        c.RemoteMode,
		"remotePort":        c.RemotePort,
		"remoteBindAddress": c.RemoteBindAddress,
		"remoteLanTls":      c.RemoteLanTls,
		"remoteAppBaseUrl":  c.RemoteAppBaseUrl,
		"locale":            c.Locale,
		"theme":             c.Theme,
	}
}

// openPhoneSettingsSection opens the desktop settings page through the same
// deeplink path whaleisle://settings uses. The bridge reader is not the UI
// goroutine, so the work is dispatched to main and bounded.
func (s *mainState) openPhoneSettingsSection(section string) bool {
	done := make(chan bool, 1)
	mygo.RunOnMain(func() {
		if s.quitting {
			done <- false
			return
		}
		s.applyDeeplink(deeplink.Target{Action: deeplink.ActionSettings, Section: strings.TrimSpace(section)})
		done <- true
	})
	select {
	case opened := <-done:
		return opened
	case <-time.After(2 * time.Second):
		return false
	}
}

func (s *mainState) savePhoneConfigPatch(patch map[string]any) (map[string]any, error) {
	if s.cfgStore == nil {
		return nil, errors.New("config store unavailable")
	}
	if err := s.cfgStore.Update(func(c *config.Config) { _ = applyRemoteConfigPatch(c, patch) }); err != nil {
		return nil, err
	}
	return redactedRemoteConfig(s.cfgStore.Get()), nil
}
