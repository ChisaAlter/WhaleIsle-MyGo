package composer

import "github.com/egoist/mygo/ui"

// Local IME/suggestions/dialogs own Escape before a voice operation. An active
// voice operation must never arm the model-response double-Escape sequence.
func cancelVoiceOnEscape(client *DraftClient, m Model, ev ui.InputEvent) bool {
	if !m.Voice.Busy() {
		return false
	}
	client.stop.reset()
	if client.marked != nil || m.StopBlocked || m.stopSuggestionsOpen || ev.Mods != 0 || ev.Repeat {
		return false
	}
	if m.OnCancelVoice != nil {
		m.OnCancelVoice()
	}
	return true
}

func voiceStartButton(c *ui.Context, m Model, locked bool, height float32) {
	setup := m.OnStartVoice == nil || !m.Snapshot.ClientReady || m.Snapshot.SpeechSelection == "" || (m.Voice.Phase == "feedback" && m.Voice.Pending == "" && m.Voice.Message != "")
	if setup && m.OnVoiceSettings != nil {
		if ui.Button(c, "语音设置 / Voice settings").Height(height).Disabled(locked).Tooltip("打开设置，检查语音输入插件、服务商准备状态和麦克风权限 / Open settings to configure speech input").Clicked() {
			m.OnVoiceSettings()
		}
		return
	}
	if m.OnStartVoice != nil && ui.Button(c, "录音 / Record").Height(height).Disabled(locked).Tooltip("点击使用系统默认麦克风；停止或到时交当前语音提供方转写，只填草稿，不发送消息。"+m.Snapshot.SpeechSelection).Clicked() {
		m.OnStartVoice()
	}
}
