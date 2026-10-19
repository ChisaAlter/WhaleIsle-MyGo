package composer

import "github.com/egoist/mygo/ui"

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
		title := composerText("语音识别暂不可用", "Speech recognition unavailable")
		body := composerText("请在语音输入插件详情中查看识别服务状态和设置。", "Check recognition status and settings in the Voice Input plugin details.")
		if m.Snapshot.SpeechNeedsInstallation {
			title = composerText("准备语音输入", "Set up voice input")
			body = composerText("所选本地识别服务首次使用前需要准备模型。请打开插件详情开始准备。", "The selected local recognizer needs its models prepared before first use. Open plugin details to prepare it.")
		}
		panel.Width(440).MaxWidthPercent(92).Padding(20).Background(c.Theme().Background).Children(func() {
			ui.Column(c).FillWidth().Gap(16).Children(func() {
				ui.Text(c, title).FontSize(17).Bold()
				ui.Text(c, body).FontSize(13)
				ui.Row(c).FillWidth().Gap(8).Justify(ui.End).Children(func() {
					if ui.Button(c, composerText("稍后", "Later")).Clicked() {
						m.State.VoiceSetupOpen = false
					}
					if ui.Button(c, composerText("打开插件详情", "Open plugin details")).Disabled(m.OnVoiceSettings == nil).Clicked() {
						m.State.VoiceSetupOpen = false
						m.OnVoiceSettings()
					}
				})
			})
		})
	})
}
