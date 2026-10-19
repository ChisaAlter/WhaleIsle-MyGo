package composer

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/desktop/voiceinput"
	"whaleisle/internal/store/session"
)

func TestVoiceEscapeOwnsCancellation(t *testing.T) {
	for _, phase := range []string{"requesting", "recording", "transcribing"} {
		draft := NewDraftState("unchanged")
		canceled, stopped := 0, 0
		m := Model{Draft: &draft.Text, State: draft, Voice: voiceinput.Status{Phase: phase}, Snapshot: session.ViewState{ClientReady: true, SelectedID: "A", StopSequenceMS: 500, Items: []session.Item{{SessionID: "A", Running: true}}}, OnCancelVoice: func() { canceled++ }, OnStop: func() { stopped++ }}
		key := ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEscape}
		for i := 0; i < 2; i++ {
			if !draftInputEvent(draft.Client, m, key) {
				t.Fatal("voice Escape unhandled")
			}
		}
		if canceled != 2 || stopped != 0 || draft.Text != "unchanged" {
			t.Fatalf("%s canceled=%d stopped=%d", phase, canceled, stopped)
		}
		for _, barrier := range []string{"ime", "modal", "suggestion", "modified", "repeat", "model"} {
			m.StopBlocked = barrier == "modal"
			m.stopSuggestionsOpen = barrier == "suggestion"
			if barrier == "ime" {
				draft.Client.marked = &ui.TextInputRange{}
			}
			if barrier == "model" {
				draft.ModelCommand = ModelCommandState{Open: true, SessionID: "A"}
			}
			ev := key
			if barrier == "modified" {
				ev.Mods = ui.Shift
			}
			if barrier == "repeat" {
				ev.Repeat = true
			}
			draftInputEvent(draft.Client, m, ev)
			if canceled != 2 || stopped != 0 {
				t.Fatalf("%s bypassed %s", phase, barrier)
			}
			draft.Client.marked = nil
			draft.ModelCommand.Open = false
		}
	}
}

func TestUnavailableVoiceOpensSettings(t *testing.T) {
	for _, mode := range []string{"missing", "offline", "unselected", "unprepared"} {
		started, settings := 0, 0
		m := Model{Snapshot: session.ViewState{ClientReady: true, SpeechSelection: "provider/en"}, OnStartVoice: func() { started++ }, OnVoiceSettings: func() { settings++ }}
		switch mode {
		case "missing":
			m.OnStartVoice = nil
		case "offline":
			m.Snapshot.ClientReady = false
		case "unselected":
			m.Snapshot.SpeechSelection = ""
		case "unprepared":
			m.Snapshot.SpeechReadinessKnown, m.Snapshot.SpeechNeedsInstallation = true, true
		}
		d := NewDraftState("keep draft")
		selected := 0
		m.State, m.Draft, m.TextInput = d, &d.Text, d.Client
		m.UseTextInputClient = true
		m.SuggestList = &ui.ListState{Selected: &selected}
		tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 400, 500)
		if err := tt.Click(composerText("语音设置", "Voice settings")); err != nil {
			t.Fatalf("%s: %v", mode, err)
		}
		if settings != 0 || started != 0 || !d.VoiceSetupOpen {
			t.Fatalf("%s microphone bypassed setup guide", mode)
		}
		if err := tt.Click(composerText("打开插件详情", "Open plugin details")); err != nil {
			t.Fatalf("%s: %v", mode, err)
		}
		if settings != 1 || started != 0 || d.Text != "keep draft" || d.VoiceSetupOpen {
			t.Fatalf("%s settings=%d microphone=%d", mode, settings, started)
		}
	}
}

func TestVoiceFeedbackRetriesOnlyWhileRecognitionRemainsUsable(t *testing.T) {
	for _, mode := range []string{"ready", "offline", "unselected", "unprepared"} {
		t.Run(mode, func(t *testing.T) {
			d := NewDraftState("keep draft")
			started, settings := 0, 0
			m := Model{State: d, Draft: &d.Text, Snapshot: session.ViewState{ClientReady: true, SpeechSelection: "provider/en", SpeechReadinessKnown: true, SpeechReady: true}, Voice: voiceinput.Status{Phase: "feedback", Message: "Transcription failed"}, OnStartVoice: func() { started++ }, OnVoiceSettings: func() { settings++ }}
			switch mode {
			case "offline":
				m.Snapshot.ClientReady = false
			case "unselected":
				m.Snapshot.SpeechSelection = ""
			case "unprepared":
				m.Snapshot.SpeechReady = false
			}
			tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 400, 500)
			if err := tt.Click(composerText("重试录音", "Retry recording")); err != nil {
				t.Fatal(err)
			}
			want := 0
			if mode == "ready" {
				want = 1
			}
			if started != want || settings != 0 || d.Text != "keep draft" {
				t.Fatalf("%s retry=%d settings=%d draft=%q", mode, started, settings, d.Text)
			}
		})
	}
}
