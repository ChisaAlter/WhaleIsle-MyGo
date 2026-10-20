package composer

import (
	"github.com/egoist/mygo/ui"
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
	ui.DialogBase(c, &m.State.VoiceSetupOpen, func(backdrop, panel ui.Element) {
		title := i18n.T(i18n.KeyComposerVoiceUnavailable)
		body := i18n.T(i18n.KeyComposerVoiceUnavailableBody)
		if m.Snapshot.SpeechNeedsInstallation {
			title = i18n.T(i18n.KeyComposerVoicePrepare)
			body = i18n.T(i18n.KeyComposerVoicePrepareBody)
		}
		panel.Width(440).MaxWidthPercent(92).Padding(20).Background(c.Theme().Background).Children(func() {
			ui.Column(c).FillWidth().Gap(16).Children(func() {
				ui.Text(c, title).FontSize(17).Bold()
				ui.Text(c, body).FontSize(13)
				ui.Row(c).FillWidth().Gap(8).Justify(ui.End).Children(func() {
					if ui.Button(c, i18n.T(i18n.KeyComposerVoiceLater)).Clicked() {
						m.State.VoiceSetupOpen = false
					}
					if ui.Button(c, i18n.T(i18n.KeyComposerVoiceDetails)).Disabled(m.OnVoiceSettings == nil).Clicked() {
						m.State.VoiceSetupOpen = false
						m.OnVoiceSettings()
					}
				})
			})
		})
	})
}
