package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/config"
	"whaleisle/internal/desktop/voicecapture"
	"whaleisle/internal/desktop/voiceinput"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/errcopy"
	"whaleisle/internal/harness"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
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

	probeOwner   *dsh.Client
	probeGen     uint64
	probeRunning bool
	probeDone    bool
	probeErr     error
	probeUnavail bool
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
		value := s.effectiveConfigFor(cfg)
		selected = voicecapture.Device{ID: value.GoVoiceMicrophoneID, Label: value.GoVoiceMicrophoneLabel}
	}
	microphone := voiceinput.NewMicrophone(selected, func(device voicecapture.Device) error {
		if cfg == nil {
			return fmt.Errorf("%s", i18n.T(i18n.KeyCmdMicrophoneSettingsUnavailable))
		}
		_ = s.flushExistingConfigWriter(cfg)
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

func (s *mainState) probeSpeechServiceAsync(client *dsh.Client, force bool) {
	if client == nil {
		return
	}
	s.mu.Lock()
	seat := s.voiceDevices
	if seat == nil {
		s.mu.Unlock()
		return
	}
	if seat.probeOwner != client {
		seat.probeOwner = client
		seat.probeGen++
		seat.probeRunning = false
		seat.probeDone = false
		seat.probeErr = nil
		seat.probeUnavail = false
	}
	if !force && (seat.probeRunning || seat.probeDone) {
		s.mu.Unlock()
		return
	}
	seat.probeGen++
	gen := seat.probeGen
	seat.probeRunning = true
	seat.probeDone = false
	seat.probeErr = nil
	seat.probeUnavail = false
	win := s.win
	s.mu.Unlock()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		defer cancel()
		var cat api.SpeechCatalog
		err := client.Call(ctx, "speech/catalog", map[string]any{}, &cat)
		s.mu.Lock()
		if s.voiceDevices == seat && seat.probeOwner == client && seat.probeGen == gen {
			seat.probeRunning = false
			seat.probeDone = true
			seat.probeErr = err
			seat.probeUnavail = errors.Is(err, dsh.ErrEndpointNotReady)
		}
		s.mu.Unlock()
		if win != nil {
			win.Invalidate()
		}
	}()
}

// This is the exact plugins.bundle.config key registered by the optional
// original VoiceInput mount. The manager calls it only for that live page.
func (s *mainState) renderVoiceDeviceSettings(c *ui.Context, key string) bool {
	if key != voiceInputBundle {
		return false
	}
	s.renderVoiceInputSection(c)
	return true
}

// renderVoiceInputSection paints the real voice page — device, provider,
// language and Host readiness — that the settings rail opens directly. The
// microphone preview is owned only while that page paints.
func (s *mainState) renderVoiceInputSection(c *ui.Context) {
	microphone := s.voiceMicrophone()
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	s.mu.Lock()
	visible := !s.projectWindowHidden && !s.quitting
	sessions := s.sessions
	bootState := s.bootStatus.State
	seat := s.voiceDevices
	if seat.probeOwner != client {
		seat.probeOwner = client
		seat.probeGen++
		seat.probeRunning = false
		seat.probeDone = false
		seat.probeErr = nil
		seat.probeUnavail = false
	}
	probeDone := seat.probeDone
	probeUnavail := seat.probeUnavail
	probeErr := seat.probeErr
	s.mu.Unlock()

	if sessions == nil || client == nil || (bootState != harness.Ready && bootState != harness.Stopped) {
		s.mu.Lock()
		s.voiceDevices.rendered = false
		s.mu.Unlock()
		microphone.SetActive(false)
		s.renderVoiceServiceFallback(c, i18n.Pick(
			"语音服务离线：Host 尚未就绪，请等待 Host 连接后再试。",
			"Speech service is offline: wait for the Host to connect, then try again.",
		), nil)
		return
	}

	owner, readiness := sessions.BindSpeechSettings()
	if readiness.Catalog == nil {
		s.mu.Lock()
		s.voiceDevices.rendered = false
		s.mu.Unlock()
		microphone.SetActive(false)
		if readiness.ConnectionError == "" {
			if !probeDone {
				s.probeSpeechServiceAsync(client, false)
			} else if probeUnavail {
				s.renderVoiceServiceFallback(c, i18n.Pick(
					"当前运行时未启用语音识别服务。可在插件管理中安装或启用语音输入扩展。",
					"Speech recognition service is not enabled in the current runtime. Install or enable the voice-input extension in Plugins.",
				), func() {
					s.probeSpeechServiceAsync(client, true)
					owner.Retry()
				})
				return
			} else if probeErr != nil {
				s.renderVoiceServiceFallback(c, errcopy.Under(
					i18n.Pick("读取语音服务", "Loading speech service"),
					probeErr,
				), func() {
					s.probeSpeechServiceAsync(client, true)
					owner.Retry()
				})
				return
			}
		}
		settingsui.VoicePreparationView(c, sessions, microphone)
		return
	}

	s.mu.Lock()
	s.voiceDevices.rendered = visible
	s.mu.Unlock()
	microphone.SetActive(visible)
	settingsui.VoicePreparationView(c, sessions, microphone)
}

func (s *mainState) renderVoiceServiceFallback(c *ui.Context, message string, onRetry func()) {
	t := design.For(c)
	ui.Column(c.Key("voice-service-fallback")).FillWidth().Gap(design.Sp(3)).Children(func() {
		coss.Alert(c, "", message, coss.AlertOptions{Variant: coss.AlertInfo, Key: "voice-service-status"})
		if onRetry != nil {
			if coss.Button(c, i18n.Pick("重新检测语音服务", "Retry speech service"), coss.ButtonOptions{
				Variant: coss.ButtonOutline,
				Size:    coss.ButtonSM,
			}).AlignSelf(ui.Start).Clicked() {
				onRetry()
			}
		}
		ui.Text(c, i18n.Pick(
			"语音服务就绪后，可在此配置麦克风、识别服务与离线语音模型。",
			"Once the speech service is ready, you can configure microphones, providers, and offline speech models here.",
		)).FontSize(design.TextXS.Size).TextColor(t.MutedForeground)
	})
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
	owner := controller.Snapshot().Value.Owner
	s.mu.Lock()
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
	visible := s.marketOpen && s.marketManager && !s.settingsOpen && !s.maintenanceOpen && !s.quitting
	owner := seat.detailsOwner
	if !visible {
		seat.detailsPending = false
	}
	s.mu.Unlock()
	if !visible {
		return
	}
	controller := s.nativePluginController()
	state := controller.Snapshot().Value
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
