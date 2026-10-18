package composer

import "strings"

// AppendTranscript appends transcribed speech text into the draft.
func AppendTranscript(draft *string, text string) {
	if draft == nil {
		return
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	cur := strings.TrimRight(*draft, " \t")
	if cur == "" {
		*draft = text
		return
	}
	*draft = cur + " " + text
}
