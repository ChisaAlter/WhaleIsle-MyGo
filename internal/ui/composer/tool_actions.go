package composer

import (
	"fmt"
	"strings"
	"time"

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
	// Intake rows open the Add section; mode and utility rows open Commands.
	sectionAdd, sectionCommands := composerText("添加", "Add"), composerText("指令", "Commands")
	add := func(section, zh, en string, action func()) {
		if action == nil || query != "" && !strings.Contains(strings.ToLower(zh+" "+en), strings.ToLower(query)) {
			return
		}
		items = append(items, suggestItem{label: composerText(zh, en), section: section, action: action})
	}
	add(sectionAdd, "添加文件", "Add files", m.OnPickImage)
	if m.OnPasteImage != nil {
		add(sectionAdd, "粘贴图片", "Paste image", func() { m.OnPasteImage(true) })
	}
	add(sectionAdd, "引用文件夹", "Reference folder", m.OnPickDirectory)
	if !m.Voice.Busy() && m.Voice.Pending == "" {
		if voiceNeedsSetup(m) && m.OnVoiceSettings != nil {
			add(sectionAdd, "语音设置", "Voice settings", m.OnVoiceSettings)
		} else {
			add(sectionAdd, "录音", "Record", m.OnStartVoice)
		}
	}
	if !m.Voice.Busy() {
		add(sectionAdd, "转写语音文件", "Transcribe audio file", m.OnPickSpeech)
		add(sectionAdd, "切换语音服务", "Change speech provider", m.OnCycleSpeech)
	}
	if m.OnPromptModeChange != nil {
		addMode := func(zh, en string, selected bool, action func()) {
			before := len(items)
			add(sectionCommands, zh, en, action)
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
		add(sectionCommands, "技能", "Skills", func() { m.OnToggleSkills(!m.SkillsOpen) })
	}
	return items
}

// The optional voice contribution keeps its microphone trigger in the input
// activity seat; feedback offers Retry without forcing a trip to Settings.
func paintVoiceControls(c *ui.Context, m Model, fg, muted, fill ui.Color, height float32) {
	locked := m.State != nil && (m.State.Sending || m.State.Preparing > 0)
	button := func(glyph, label string, disabled bool, action func()) {
		if action == nil {
			return
		}
		caption := glyph
		if glyph == "microphone" {
			caption = ""
		}
		control := ui.Button(c, caption).Label(label).Tooltip(label).
			Width(height).Height(height).Padding(0).Radius(height / 2).
			Background(fill).BorderWidth(0).TextColor(fg).Disabled(disabled)
		if glyph == "microphone" {
			control.Children(func() { ui.Icon(c, voiceMicrophoneIcon).Size(18, 18).PassThrough() })
		}
		if control.Clicked() {
			action()
		}
	}
	if m.Voice.Phase == "" {
		// One stable control name: the launcher row that opens voice settings
		// keeps its own caption, so the microphone never shadows it in the tree.
		if voiceNeedsSetup(m) {
			button("microphone", i18n.T(i18n.KeyComposerVoiceInput), locked, func() {
				if m.State != nil {
					m.State.VoiceSetupOpen = true
				} else if m.OnVoiceSettings != nil {
					m.OnVoiceSettings()
				}
			})
		} else {
			button("microphone", i18n.T(i18n.KeyComposerVoiceInput), locked, m.OnStartVoice)
		}
	}
	paintVoiceSetup(c, m)
	if m.Voice.Phase == "recording" {
		button("■", i18n.T(i18n.KeyComposerVoiceStop), false, m.OnStopVoice)
		paintVoiceClock(c, m, fg, muted)
	} else if m.State != nil && !m.State.VoiceStarted.IsZero() {
		// Leaving the recording phase retires the clock; the controller owns
		// the transcript, not the elapsed time.
		m.State.VoiceStarted = time.Time{}
		m.State.VoiceTick = time.Time{}
	}
	if m.Voice.Pending != "" {
		button("↵", i18n.T(i18n.KeyComposerVoiceInsert), locked, m.OnInsertVoice)
	}
	if m.Voice.Phase == "feedback" && m.Voice.Pending == "" {
		button("microphone", i18n.T(i18n.KeyComposerVoiceRetry), locked || voiceNeedsSetup(m), m.OnStartVoice)
	}
	if m.Voice.Phase != "" {
		button("×", i18n.T(i18n.KeyComposerVoiceCancel), false, m.OnCancelVoice)
	}
	if m.Voice.Message != "" {
		// Feedback such as "microphone unavailable" stays visible and offers the
		// settings seat as its recovery, so a failed capture is never a dead end.
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Text(c, m.Voice.Message).MaxWidth(180).SingleLine().FontSize(11).TextColor(muted).Tooltip(m.Voice.Pending)
			if !m.Voice.Busy() && m.OnVoiceSettings != nil {
				link := ui.Button(c, i18n.T(i18n.KeyComposerVoiceSettings)).FontSize(11).Padding(0, 4).Height(20).
					Radius(6).BorderWidth(0).Background(ui.Transparent).TextColor(muted)
				if link.Clicked() {
					m.OnVoiceSettings()
				}
			}
		})
	}
}

// voiceTickInterval paces the recording clock and the activity bars; the strip
// only exists while a capture is running, so the repaint cost is bounded.
const voiceTickInterval = 50 * time.Millisecond

// paintVoiceClock draws a rolling history of measured capture RMS and the clock.
// Levels come from the active capture's measured RMS, never a clock animation.
func paintVoiceClock(c *ui.Context, m Model, fg, muted ui.Color) {
	if m.State == nil {
		return
	}
	now := c.Now()
	if m.State.VoiceStarted.IsZero() || now.Before(m.State.VoiceStarted) {
		// A clock that stepped backwards (test frame clocks, sleep/wake) starts
		// the strip again rather than showing a negative elapsed time.
		m.State.VoiceStarted = now
		m.State.VoiceTick = time.Time{}
	}
	elapsed := now.Sub(m.State.VoiceStarted)
	if elapsed < 0 {
		elapsed = 0
	}
	if m.State.VoiceTick.IsZero() || !now.Before(m.State.VoiceTick.Add(voiceTickInterval)) {
		m.State.VoiceTick = now
		services := c.Services()
		time.AfterFunc(voiceTickInterval, services.Invalidate)
	}
	seconds := int(elapsed / time.Second)
	clock := fmt.Sprintf("%d:%02d", seconds/60, seconds%60)
	row := ui.Row(c.Key("voice-waveform")).Gap(6).AlignItems(ui.Center).MinWidth(0).Shrink(1)
	width, _ := c.Size()
	waveWidth := max(float32(24), min(float32(240), (width-180)*.4))
	history := ui.Local(row, "rms-history", func() voiceWaveformHistory { return voiceWaveformHistory{} })
	if !history.Started.Equal(m.State.VoiceStarted) {
		*history = voiceWaveformHistory{Started: m.State.VoiceStarted}
	}
	if history.Sampled.IsZero() || !now.Before(history.Sampled.Add(voiceTickInterval)) {
		history.Sampled = now
		history.Values = append(history.Values, max(float32(0), min(float32(1), float32(m.Voice.Level))))
		if len(history.Values) > 80 {
			history.Values = history.Values[len(history.Values)-80:]
		}
	}
	values := append([]float32(nil), history.Values...)
	row.Children(func() {
		ui.Box(c).Width(waveWidth).MinWidth(24).Height(24).Shrink(1).Label(i18n.T(i18n.KeyComposerVoiceRecording)).Draw(func(p *ui.Painter, r ui.Rect) {
			const bars = 80
			step := r.W / bars
			for i := 0; i < bars; i++ {
				level := float32(0)
				if index := len(values) - bars + i; index >= 0 {
					level = values[index]
				}
				bar := max(float32(1.2), r.H*.9*level)
				p.Fill(ui.Rect{X: r.X + (float32(i)+.5)*step, Y: r.Y + (r.H-bar)/2, W: max(float32(.6), step*.38), H: bar}, muted.Alpha(.25+float32(i)/120), 1)
			}
		})
		ui.Text(c, clock).FontSize(11).TextColor(fg).FixedLineHeight(14).Shrink(0).Tooltip(i18n.T(i18n.KeyComposerVoiceRecording))
	})
}

type voiceWaveformHistory struct {
	Started, Sampled time.Time
	Values           []float32
}

// composerVoiceConfigured reports a usable speech selection, which is what
// lets the hero show the microphone instead of hiding the unconfigured seat.
func composerVoiceConfigured(m Model) bool {
	if m.OnStartVoice == nil || !m.Snapshot.ClientReady || strings.TrimSpace(m.Snapshot.SpeechSelection) == "" {
		return false
	}
	return !m.Snapshot.SpeechReadinessKnown || m.Snapshot.SpeechReady
}

func voiceNeedsSetup(m Model) bool {
	return m.OnStartVoice == nil || !m.Snapshot.ClientReady || m.Snapshot.SpeechSelection == "" || m.Snapshot.SpeechReadinessKnown && !m.Snapshot.SpeechReady
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
