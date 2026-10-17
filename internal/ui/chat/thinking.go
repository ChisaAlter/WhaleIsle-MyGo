package chat

import (
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

// ThinkingStyle colors for collapsible thinking blocks.
type ThinkingStyle struct {
	FG     ui.Color
	Muted  ui.Color
	Border ui.Color
	Field  ui.Color
	Size   float32
}

// ThinkingBlockView renders assistant thinking in a muted, expandable panel.
func ThinkingBlockView(c *ui.Context, msgID, body string, expanded *map[string]bool, st ThinkingStyle) {
	size := st.Size
	if size <= 0 {
		size = 12
	}
	open := isThinkingExpanded(expanded, msgID)
	chevron := "▸"
	if open {
		chevron = "▾"
	}
	preview := thinkingPreview(body, 120)
	ui.Column(c).Padding(8, 10, 8, 10).Gap(4).Fill().Background(st.Field).Border(1, st.Border).Radius(6).Children(func() {
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			header := ui.Button(c, chevron+" "+i18n.T(i18n.KeyChatThinking))
			if header.Clicked() {
				toggleThinkingExpanded(expanded, msgID)
			}
			if !open && preview != "" {
				ui.Text(c, preview).FontSize(size - 1).TextColor(st.Muted).Grow(1)
			}
		})
		if open && strings.TrimSpace(body) != "" {
			ui.Text(c, body).FontSize(size).TextColor(st.Muted)
		}
	})
}

func isThinkingExpanded(expanded *map[string]bool, msgID string) bool {
	if expanded == nil || msgID == "" {
		return false
	}
	return (*expanded)[msgID]
}

func toggleThinkingExpanded(expanded *map[string]bool, msgID string) {
	if expanded == nil {
		return
	}
	m := *expanded
	if m == nil {
		m = map[string]bool{}
	}
	m[msgID] = !m[msgID]
	*expanded = m
}

func thinkingPreview(body string, maxRunes int) string {
	body = strings.TrimSpace(body)
	body = strings.Join(strings.Fields(body), " ")
	if maxRunes <= 0 || utf8.RuneCountInString(body) <= maxRunes {
		return body
	}
	r := []rune(body)
	return string(r[:maxRunes]) + "…"
}
