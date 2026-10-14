package composer

import (
	"github.com/egoist/mygo/ui"
	"strings"
)

// SpeechInsertion captures the original native UTF-16 selection, not whichever
// session/caret happens to be active when the asynchronous transcript arrives.
type SpeechInsertion struct {
	text    string
	target  ui.TextInputRange
	editSeq float64
}

func CaptureSpeechInsertion(d *DraftState) (SpeechInsertion, bool) {
	if d == nil || d.Client == nil || d.Sending {
		return SpeechInsertion{}, false
	}
	if _, marked := d.Client.MarkedRange(); marked {
		return SpeechInsertion{}, false
	}
	return SpeechInsertion{text: d.Text, target: d.Client.Selection().Range, editSeq: d.EditSeq}, true
}
func (s SpeechInsertion) Insert(d *DraftState, text string) bool {
	text = strings.TrimSpace(text)
	if d == nil || d.Client == nil || d.Sending || d.Text != s.text || d.EditSeq != s.editSeq || text == "" {
		return false
	}
	if _, marked := d.Client.MarkedRange(); marked {
		return false
	}
	d.Client.ReplaceText(&s.target, text)
	d.Error = ""
	return true
}
func InsertPendingSpeech(d *DraftState, text string) bool {
	captured, ok := CaptureSpeechInsertion(d)
	return ok && captured.Insert(d, text)
}
