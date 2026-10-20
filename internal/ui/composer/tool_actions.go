package composer

import (
	"fmt"
	"math"
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

// The optional voice contribution keeps its microphone trigger in the input
// activity seat; feedback offers Retry without forcing a trip to Settings.
func paintVoiceControls(c *ui.Context, m Model, fg, muted, fill ui.Color, height float32) {
	// The hero composer mirrors the reference's control set: no microphone seat
	// until speech is configured, unless a capture, feedback or the setup guide
	// is active. The docked composer keeps its always-visible control.
	if m.Hero && !composerVoiceConfigured(m) && m.Voice.Phase == "" && m.Voice.Pending == "" &&
		m.Voice.Message == "" && !(m.State != nil && m.State.VoiceSetupOpen) {
		return
	}
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
const voiceTickInterval = 200 * time.Millisecond

// paintVoiceClock draws the live "1:05" clock and a five-bar activity strip.
// The bars are an animation of the recording state, not a level meter: the
// capture pipeline publishes no amplitudes.
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
	ui.Row(c).Gap(6).AlignItems(ui.Center).Shrink(0).Children(func() {
		t := float32(elapsed.Seconds())
		ui.Box(c).Width(26).Height(12).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
			const bars = 5
			for i := 0; i < bars; i++ {
				level := 0.35 + 0.65*float32(math.Abs(math.Sin(float64(t)*2.4+float64(i)*1.35)))
				bar := r.H * level
				step := r.W / bars
				p.Fill(ui.Rect{X: r.X + float32(i)*step, Y: r.Y + (r.H-bar)/2, W: max(1, step-2), H: bar}, muted, 1)
			}
		})
		ui.Text(c, clock).FontSize(11).TextColor(fg).FixedLineHeight(14).Shrink(0).Tooltip(i18n.T(i18n.KeyComposerVoiceRecording))
	})
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
