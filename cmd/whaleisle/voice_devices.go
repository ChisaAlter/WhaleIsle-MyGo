package main

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/config"
	"whaleisle/internal/desktop/voicecapture"
	"whaleisle/internal/desktop/voiceinput"
	"whaleisle/internal/ui/i18n"
	settingsui "whaleisle/internal/ui/settings"
)

const voiceInputBundle = "@deepseek-ai/dsh-experimental-voice-input-bundle"

// One client-local device seat survives Host and Session changes. Hardware
// ownership is restricted to the rendered voice settings page.
type composerVoiceDevices struct {
	microphone     *voiceinput.Microphone
	rendered       bool
	detailsPending bool
	detailsOwner   uint64
}

func (s *mainState) voiceMicrophone() *voiceinput.Microphone {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.voiceDevices != nil {
		return s.voiceDevices.microphone
	}
	cfg, win := s.cfgStore, s.win
	selected := voicecapture.Device{}
	if cfg != nil {
		value := cfg.Get()
		selected = voicecapture.Device{ID: value.GoVoiceMicrophoneID, Label: value.GoVoiceMicrophoneLabel}
	}
	microphone := voiceinput.NewMicrophone(selected, func(device voicecapture.Device) error {
		if cfg == nil {
			return fmt.Errorf("%s", i18n.T(i18n.KeyCmdMicrophoneSettingsUnavailable))
		}
		return cfg.Update(func(c *config.Config) {
			c.GoVoiceMicrophoneID, c.GoVoiceMicrophoneLabel = device.ID, device.Label
		})
	}, func() {
		if win != nil {
			win.Update(func() {})
		}
	})
	s.voiceDevices = &composerVoiceDevices{microphone: microphone}
	return microphone
}

// This is the exact plugins.bundle.config key registered by the optional
// original VoiceInput mount. The manager calls it only for that live page.
func (s *mainState) renderVoiceDeviceSettings(c *ui.Context, key string) bool {
	if key != voiceInputBundle {
		return false
	}
	microphone := s.voiceMicrophone()
	s.mu.Lock()
	visible := !s.projectWindowHidden && !s.quitting
	sessions := s.sessions
	s.mu.Unlock()
	if sessions == nil {
		visible = false
	} else {
		_, readiness := sessions.BindSpeechSettings()
		visible = visible && readiness.Catalog != nil
	}
	s.mu.Lock()
	s.voiceDevices.rendered = visible
	s.mu.Unlock()
	microphone.SetActive(visible)
	settingsui.VoicePreparationView(c, sessions, microphone)
	return true
}

// Window hide/minimize/close and shutdown retire microphone requests as well
// as an acquired preview. No raw samples or transcription are persisted.
func (s *mainState) pauseVoiceDevices() {
	s.mu.Lock()
	seat := s.voiceDevices
	if seat != nil {
		seat.rendered = false
	}
	s.mu.Unlock()
	if seat != nil {
		seat.microphone.SetActive(false)
	}
}

// Called once after the application's UI frame. A settings-page switch must
// release preview ownership even when that page no longer renders a callback.
func (s *mainState) finishVoiceDeviceSettingsFrame() {
	s.finishVoiceDetailsNavigation()
	s.mu.Lock()
	seat := s.voiceDevices
	pause := seat != nil && !seat.rendered
	if seat != nil {
		seat.rendered = false
	}
	s.mu.Unlock()
	if pause {
		seat.microphone.SetActive(false)
	}
}

// Navigate through the existing manager even when its inventory is still cold.
func (s *mainState) openVoiceSettings() {
	s.pauseVoiceDevices()
	s.voiceMicrophone()
	controller := s.nativePluginController()
	owner := controller.Snapshot().Owner
	s.mu.Lock()
	s.settingsOpen = false
	s.settingsEmbeddedSection = ""
	s.voiceDevices.detailsPending = true
	s.voiceDevices.detailsOwner = owner
	s.mu.Unlock()
	s.selectMainPanel("plugins")
	controller.Ensure()
}

func (s *mainState) finishVoiceDetailsNavigation() {
	s.mu.Lock()
	seat := s.voiceDevices
	if seat == nil || !seat.detailsPending {
		s.mu.Unlock()
		return
	}
	visible := s.marketOpen && s.marketManager && !s.settingsOpen && !s.launcherOpen && !s.quitting
	owner := seat.detailsOwner
	if !visible {
		seat.detailsPending = false
	}
	s.mu.Unlock()
	if !visible {
		return
	}
	controller := s.nativePluginController()
	state := controller.Snapshot()
	if state.Owner != owner || state.Status != "idle" && state.Status != "loading" {
		s.mu.Lock()
		if s.voiceDevices == seat && seat.detailsOwner == owner {
			seat.detailsPending = false
		}
		s.mu.Unlock()
		if state.Owner == owner {
			controller.Actions(owner).OpenPackage(voiceInputBundle)
		}
	}
}
