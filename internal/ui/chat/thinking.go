package chat

import (
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets/markdown"
)

// ThinkingStyle colors for collapsible thinking blocks.
type ThinkingStyle struct {
	HidePreview bool
	Streaming   bool
	FG          ui.Color
	Muted       ui.Color
	Border      ui.Color
	Field       ui.Color
	Size        float32
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
	preview := reasoningSummary(body, st.Streaming)
	ui.Column(c).Padding(8, 10, 8, 10).Gap(4).Fill().Background(st.Field).Border(1, st.Border).Radius(6).Children(func() {
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			header := ui.Button(c, chevron+" "+i18n.T(i18n.KeyChatThinking))
			if header.Clicked() {
				toggleThinkingExpanded(expanded, msgID)
			}
			if strings.TrimSpace(body) != "" && ui.Button(c, i18n.T(i18n.KeyChatCopy)).Label("Copy reasoning "+msgID).Clicked() {
				c.WriteClipboard(body)
			}
			if !open && !st.HidePreview && preview != "" {
				ui.Text(c, preview).FontSize(size).TextColor(st.Muted).Grow(1).MinWidth(0).SingleLine()
			}
		})
		if open && strings.TrimSpace(body) != "" {
			markdown.View(c, body, markdown.Style{FontSize: size, FG: st.Muted, Muted: st.Muted, CodeBG: st.Field, Accent: st.FG})
		}
	})
}

// A live reasoning preview advances only when a paragraph's first line is
// complete. Settled reasoning uses its first line. Partial words from the
// streaming tail must not replace the last completed summary on every chunk.
func reasoningSummary(body string, streaming bool) string {
	if !streaming {
		line, _, _ := strings.Cut(body, "\n")
		return strings.ReplaceAll(strings.TrimSuffix(line, "\r"), "**", "")
	}
	lines := strings.Split(body, "\n")
	summary := ""
	paragraphStart := true
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			paragraphStart = true
			continue
		}
		if paragraphStart && i < len(lines)-1 {
			summary = line
		}
		paragraphStart = false
	}
	return strings.ReplaceAll(summary, "**", "")
}

func secondaryChatSize() float32 {
	size := theme.CurrentPreferences().Chat.Size
	if size > 14 {
		return size - 2
	}
	return size - 1
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
