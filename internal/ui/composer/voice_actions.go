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
