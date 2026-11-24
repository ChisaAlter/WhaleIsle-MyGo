package main

import (
	"errors"
	"log"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/config"
	"whaleisle/internal/renderpolicy"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/settings"
	"whaleisle/internal/ui/widgets"
)

// minimalModeText localizes copy that belongs to the running shell rather than
// to a vendor catalog, in the same zh/en pair the rest of the app uses.
func minimalModeText(zh, en string) string {
	return i18n.Pick(zh, en)
}

// applyMinimalMode is the one place the switch takes effect: the render policy
// the chrome reads on every frame, and the framework's motion policy, so
// interface transitions stop without threading a flag through each surface.
// Both are process-wide, so a window that rebuilds arrives already minimal.
func applyMinimalMode(on bool) {
	renderpolicy.SetMinimal(on)
	ui.SetReduceMotion(on)
	// The design system's overlay-out: closing dialogs fade over the fast
	// token. Setting it next to the motion policy keeps it in force from
	// the first frame; less motion still removes a dialog at once.
	ui.SetDialogExit(ui.DialogExit{Duration: chrome.OverlayExitDuration, Ease: widgets.DesignEase})
}

// errMinimalModeUnavailable is the refusal before the profile's settings are
// open. It stays technical on purpose: the carrier paints the shared
// classifier's sentence (internal/errcopy) and logs this text.
var errMinimalModeUnavailable = errors.New("minimal mode: the settings store is not open yet")

// setMinimalMode applies the switch optimistically in memory and persists it
// asynchronously on the single-writer debounced config queue.
func (s *mainState) setMinimalMode(on bool) error {
	if s.cfgStore == nil {
		return errMinimalModeUnavailable
	}
	applyMinimalMode(on)
	s.enqueueConfigUpdate(s.cfgStore, func(c *config.Config) { c.MinimalMode = on })
	s.invalidateAppearance()
	return nil
}

// invalidateAppearance repaints the main window after an appearance-policy
// change. The next frame reconfigures the backdrop, the pointer decoration and
// the closing overlay from the new policy; nothing else has to be notified.
func (s *mainState) invalidateAppearance() {
	s.mu.Lock()
	win := s.win
	s.mu.Unlock()
	if win != nil {
		win.Invalidate()
	}
}

// minimalModeAccess is the appearance page's minimal-mode block. The values are
// read from the effective config every frame, so optimistic updates appear at
// once while disk persistence runs off the UI thread.
func (s *mainState) minimalModeAccess() *settings.MinimalModeAccess {
	if s.cfgStore == nil {
		return nil
	}
	cfg := s.effectiveConfigFor(s.cfgStore)
	enabled, suggestions := cfg.MinimalMode, !cfg.MinimalModeSuggestionsOff
	access := &settings.MinimalModeAccess{Enabled: &enabled, Suggestions: &suggestions}
	access.OnToggle = s.setMinimalMode
	access.OnToggleSuggestions = func(on bool) error {
		s.enqueueConfigUpdate(s.cfgStore, func(c *config.Config) {
			c.MinimalModeSuggestionsOff = !on
			if on {
				// Turning the offer back on clears a snooze it may still carry.
				c.MinimalModeSnoozedUntil = ""
			}
		})
		s.invalidateAppearance()
		return nil
	}
	return access
}

// observeStartupSlowdown feeds the bounded startup observation one built frame.
// The application's frame hook already measures this cost, so nothing here
// samples on its own; a decided observer is one pointer load per frame.
func (s *mainState) observeStartupSlowdown(build time.Duration) {
	observer := s.minimalSuggestion
	if observer == nil || observer.Done() {
		return
	}
	s.mu.Lock()
	win := s.win
	s.mu.Unlock()
	foreground := true
	if win != nil {
		foreground = win.IsVisible() && !win.IsMinimized() && win.IsFocused()
	}
	if !observer.Observe(time.Now(), build, foreground) {
		return
	}
	if s.minimalSuggestionWanted() {
		s.openMinimalModeOffer()
	}
}

// minimalSuggestionWanted is the offer's own gate: it is never made while the
// interface is already minimal, after the user turned the offer off, or during
// a snooze.
func (s *mainState) minimalSuggestionWanted() bool {
	if s.cfgStore == nil || renderpolicy.Minimal() {
		return false
	}
	cfg := s.effectiveConfigFor(s.cfgStore)
	if cfg.MinimalMode || cfg.MinimalModeSuggestionsOff {
		return false
	}
	if cfg.MinimalModeSnoozedUntil != "" {
		if until, err := time.Parse(time.RFC3339, cfg.MinimalModeSnoozedUntil); err == nil && time.Now().Before(until) {
			return false
		}
	}
	return true
}

// minimalModeSnooze is how long "暂不开启" defers the offer.
const minimalModeSnooze = 7 * 24 * time.Hour

func (s *mainState) snoozeMinimalModeOffer() {
	if s.cfgStore == nil {
		return
	}
	until := time.Now().Add(minimalModeSnooze).UTC().Format(time.RFC3339)
	if err := s.cfgStore.Update(func(c *config.Config) { c.MinimalModeSnoozedUntil = until }); err != nil {
		// The offer was on screen, so the deferral is the user's answer: a
		// failed write has to be visible in the log rather than silently
		// dropping it and asking again on the next launch.
		log.Printf("minimal mode: defer the startup offer: %v", err)
	}
}

func (s *mainState) stopMinimalModeOffers() {
	if s.cfgStore == nil {
		return
	}
	s.enqueueConfigUpdate(s.cfgStore, func(c *config.Config) { c.MinimalModeSuggestionsOff = true })
	s.invalidateAppearance()
}
