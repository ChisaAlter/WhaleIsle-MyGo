package chat

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// ContextInjectionView shows a read-only producer notice and its resolved body.
func ContextInjectionView(c *ui.Context, msg session.Message, scope string, palette theme.Colors) {
	key := fmt.Sprintf("context:%s:%s:%g", scope, msg.ID, msg.Seq)
	box := ui.Column(c.Key(key)).FillWidth().MinWidth(0)
	open := ui.Local(box, "open", func() bool { return false })
	box.Children(func() {
		if paintToolContext(c, msg, box, open, palette) {
			return
		}
		source, _ := contextMessageParts(msg)
		producer, recall := contextProducerLabel(source)
		title, icon := i18n.T(i18n.KeyChatContextTitle), contextInjectionIcon
		if recall {
			title, icon = i18n.T(i18n.KeyChatContextRecall), contextRecallIcon
		}
		body := resolveContextBody(msg)
		summary := body.Summary
		if producer == "" {
			summary = ""
		}
		paintContextDisclosure(c, box, open, contextDisclosure{Icon: icon, Title: title, Producer: producer, Summary: summary,
			Body: func() { body.paint(c, contextBodyPalette(palette)) }}, palette)
	})
}

func contextProducerLabel(source map[string]json.RawMessage) (string, bool) {
	read := func(key string) string {
		var value string
		_ = json.Unmarshal(source[key], &value)
		return value
	}
	kind := read("kind")
	collect := func(key, field string) string {
		var items []json.RawMessage
		_ = json.Unmarshal(source[key], &items)
		seen := map[string]bool{}
		var names []string
		for _, item := range items {
			var entry map[string]json.RawMessage
			if json.Unmarshal(item, &entry) != nil {
				continue
			}
			var value string
			_ = json.Unmarshal(entry[field], &value)
			if value != "" && !seen[value] {
				seen[value] = true
				names = append(names, value)
			}
		}
		return strings.Join(names, ", ")
	}
	label := ""
	switch kind {
	case "session-reference":
		label = collect("references", "label")
	case "agent-instructions":
		label = collect("changes", "path")
	case "skill-invocation":
		label = read("name")
	}
	if label == "" {
		label = kind
	}
	return label, kind == "session-reference"
}

var contextRecallIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" stroke-width="1"><path d="M5 6.75H11M5 9H8" stroke="currentColor"/><path d="M2.37067 11.2497C1.5872 9.89252 1.32042 8.29798 1.61945 6.7597C1.91847 5.22141 2.76317 3.84293 3.99801 2.87809C5.23285 1.91325 6.7747 1.427 8.33964 1.50888C9.90458 1.59076 11.3873 2.23526 12.5147 3.32369C13.6422 4.41232 14.3384 5.8717 14.4751 7.43304C14.6118 8.99438 14.1797 10.5525 13.2585 11.8205C12.3372 13.0885 10.9889 13.9809 9.4617 14.3334C8.18666 14.6277 6.8587 14.529 5.64964 14.0601C5.17095 13.8745 4.76937 13.4929 4.26509 13.3963C3.67389 13.2832 2.95232 13.5595 2.0377 14.3334" stroke="currentColor"/></svg>`))
