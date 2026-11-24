package composer

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/i18n"
)

// The existing dialog seat only guides to plugin details. It never starts
// preparation or retries a recording as a side effect of opening settings.
func paintVoiceSetup(c *ui.Context, m Model) {
	if m.State == nil {
		return
	}
	if !voiceNeedsSetup(m) || m.Voice.Busy() || m.Inert {
		m.State.VoiceSetupOpen = false
	}
	title := i18n.T(i18n.KeyComposerVoiceUnavailable)
	body := i18n.T(i18n.KeyComposerVoiceUnavailableBody)
	if m.Snapshot.SpeechNeedsInstallation {
		title = i18n.T(i18n.KeyComposerVoicePrepare)
		body = i18n.T(i18n.KeyComposerVoicePrepareBody)
	}
	coss.Dialog(c, &m.State.VoiceSetupOpen, coss.DialogOptions{
		Title:       title,
		Description: body,
		MaxWidth:    440,
		NoClose:     true,
		Footer: func() {
			if coss.Button(c, i18n.T(i18n.KeyComposerVoiceLater), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM}).Clicked() {
				m.State.VoiceSetupOpen = false
			}
			if coss.Button(c, i18n.T(i18n.KeyComposerVoiceDetails), coss.ButtonOptions{Variant: coss.ButtonDefault, Size: coss.ButtonSM, Disabled: m.OnVoiceSettings == nil}).Clicked() {
				m.State.VoiceSetupOpen = false
				m.OnVoiceSettings()
			}
		},
	}, nil)
}
