package composer

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func composerText(zh, en string) string {
	if i18n.Locale() == i18n.LocaleEN {
		return en
	}
	return zh
}

// The existing + launcher owns occasional intake and composition actions. They
// share its keyboard navigation and never replace or submit the current draft.
func composerToolItems(m Model, query string) []suggestItem {
	if m.Inert || m.State != nil && (m.State.Sending || m.State.Preparing > 0) {
		return nil
	}
	var items []suggestItem
	add := func(zh, en string, action func()) {
		if action == nil || query != "" && !strings.Contains(strings.ToLower(zh+" "+en), strings.ToLower(query)) {
			return
		}
		items = append(items, suggestItem{label: composerText(zh, en), action: action})
	}
	add("添加文件", "Add files", m.OnPickImage)
	if m.OnPasteImage != nil {
		add("粘贴图片", "Paste image", func() { m.OnPasteImage(true) })
	}
	add("引用文件夹", "Reference folder", m.OnPickDirectory)
	if !m.Voice.Busy() && m.Voice.Pending == "" {
		if voiceNeedsSetup(m) && m.OnVoiceSettings != nil {
			add("语音设置", "Voice settings", m.OnVoiceSettings)
		} else {
			add("录音", "Record", m.OnStartVoice)
		}
	}
	if !m.Voice.Busy() {
		add("转写语音文件", "Transcribe audio file", m.OnPickSpeech)
		add("切换语音服务", "Change speech provider", m.OnCycleSpeech)
	}
	if m.OnPromptModeChange != nil {
		addMode := func(zh, en string, selected bool, action func()) {
			before := len(items)
			add(zh, en, action)
			if selected && len(items) > before {
				items[len(items)-1].sub = composerText("当前设置", "Selected")
			}
		}
		mode := session.NormalizePromptMode(string(m.Snapshot.PromptMode))
		addMode("跟随发送设置", "Use send setting", !m.Snapshot.PromptModeExplicit, m.OnUseBusyEnterSetting)
		addMode("忙碌时排队", "Queue while busy", m.Snapshot.PromptModeExplicit && mode == session.PromptModeQueue, func() { m.OnPromptModeChange(session.PromptModeQueue) })
		addMode("忙碌时插话", "Steer while busy", m.Snapshot.PromptModeExplicit && mode == session.PromptModeSteer, func() { m.OnPromptModeChange(session.PromptModeSteer) })
	}
	if m.OnToggleSkills != nil {
		add("技能", "Skills", func() { m.OnToggleSkills(!m.SkillsOpen) })
	}
	return items
}

// Only an active operation occupies the toolbar; microphone setup and idle
// intake stay in +, while stop/insert/discard remain immediately reachable.
func paintVoiceControls(c *ui.Context, m Model, fg, muted, fill ui.Color, height float32) {
	locked := m.State != nil && (m.State.Sending || m.State.Preparing > 0)
	button := func(glyph, label string, disabled bool, action func()) {
		if action != nil && ui.Button(c, glyph).Label(label).Tooltip(label).
			Width(height).Height(height).Padding(0).Radius(height/2).
			Background(fill).BorderWidth(0).TextColor(fg).Disabled(disabled).Clicked() {
			action()
		}
	}
	if m.Voice.Phase == "recording" {
		button("■", composerText("停止并转写", "Stop recording"), false, m.OnStopVoice)
	}
	if m.Voice.Pending != "" {
		button("↵", composerText("插入转写", "Insert transcript"), locked, m.OnInsertVoice)
	}
	if m.Voice.Phase != "" {
		button("×", composerText("取消录音", "Discard recording"), false, m.OnCancelVoice)
	}
	if m.Voice.Message != "" {
		ui.Text(c, m.Voice.Message).MaxWidth(180).SingleLine().FontSize(11).TextColor(muted).Tooltip(m.Voice.Pending)
	}
}

func voiceNeedsSetup(m Model) bool {
	return m.OnStartVoice == nil || !m.Snapshot.ClientReady || m.Snapshot.SpeechSelection == "" ||
		(m.Voice.Phase == "feedback" && m.Voice.Pending == "" && m.Voice.Message != "")
}

func composerSendDisabled(m Model) bool {
	if m.Inert || m.StopBlocked {
		return true
	}
	if m.State != nil {
		if m.State.Sending || m.State.Preparing > 0 {
			return true
		}
		for _, attachment := range m.State.Attachments {
			if attachment.PrepareError != "" {
				return true
			}
		}
		if len(m.State.Attachments) > 0 {
			return false
		}
	}
	return composerDraftEmpty(m)
}

func composerDraftEmpty(m Model) bool {
	return (m.Draft == nil || strings.TrimSpace(*m.Draft) == "") && (m.State == nil || len(m.State.Attachments) == 0)
}
