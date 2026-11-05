package chat

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets/markdown"
)

// ThinkingStyle colors for collapsible thinking blocks.
type ThinkingStyle struct {
	Body        func(*ui.Context)
	HidePreview bool
	Streaming   bool
	FG          ui.Color
	Muted       ui.Color
	Border      ui.Color
	Field       ui.Color
	Size        float32
	// Duration, when known, renders as the row's trailing label ("思考了 12 秒").
	Duration time.Duration
	// Accent and Hover complete the disclosure palette.
	Accent ui.Color
	Hover  ui.Color
}

// ThinkingBlockView renders assistant reasoning as the Original ReasoningRow:
// one 24px disclosure line (think glyph, localized title, separator, streaming
// preview, duration) and an indented Markdown body once opened.
func ThinkingBlockView(c *ui.Context, msgID, body string, expanded *map[string]bool, st ThinkingStyle) {
	size := st.Size
	if size <= 0 {
		size = secondaryChatSize()
	}
	open := isThinkingExpanded(expanded, msgID)
	preview := ""
	if !st.HidePreview {
		preview = reasoningSummary(body, st.Streaming)
	}
	trailing := ""
	if st.Duration > 0 {
		trailing = fmt.Sprintf(i18n.T(i18n.KeyChatThinkingFor), formatDuration(st.Duration))
	}
	titleColor := st.Muted
	if titleColor.A == 0 {
		titleColor = st.FG
	}
	root := ui.Column(c).FillWidth().MinWidth(0).Gap(0)
	root.Children(func() {
		paintDisclosureRow(c, disclosureRow{
			Key:        "thinking:" + msgID,
			Icon:       disclosureIconThink,
			Title:      i18n.T(i18n.KeyChatThinking),
			TitleColor: titleColor,
			Summary:    preview,
			Trailing:   trailing,
			Status:     statusNone,
			Running:    st.Streaming,
			Open:       open,
			Expandable: strings.TrimSpace(body) != "",
			OnToggle:   func() { toggleThinkingExpanded(expanded, msgID) },
		}, disclosureStyle{FG: st.FG, Muted: st.Muted, Hover: st.Hover, Accent: st.Accent, Size: size})
		if !open || strings.TrimSpace(body) == "" {
			return
		}
		bodyIndent := max(float32(22), 22+theme.CurrentPreferences().Chat.Size-14)
		ui.Column(c).FillWidth().MinWidth(0).Padding(4, 0, 4, bodyIndent).Children(func() {
			if st.Body != nil {
				st.Body(c)
			} else {
				markdown.View(c, body, transcriptMarkdownStyle(st.Muted, st.Muted, st.Field, st.FG, size, darkSurface(st.Muted)))
			}
		})
	})
}

// formatDuration renders a short elapsed label ("12 秒", "3 分 05 秒").
func formatDuration(d time.Duration) string {
	seconds := int(d.Seconds())
	if seconds < 60 {
		return fmt.Sprintf(i18n.T(i18n.KeyChatDurationSec), seconds)
	}
	minutes := seconds / 60
	rest := seconds % 60
	if rest == 0 {
		return fmt.Sprintf(i18n.T(i18n.KeyChatDurationMin), minutes)
	}
	return fmt.Sprintf(i18n.T(i18n.KeyChatDurationMS), minutes, rest)
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
