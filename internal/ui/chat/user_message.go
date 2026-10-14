package chat

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets/markdown"
)

// userMessageParts recognizes the legacy attachment lines outside code fences.
// User-authored whitespace and fenced image-like text remain literal content.
func userMessageParts(body string) (refs []string, text string) {
	lines := strings.Split(body, "\n")
	kept := make([]string, 0, len(lines))
	seen := map[string]bool{}
	var fence byte
	fenceLength := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		marker, count := byte(0), 0
		if len(trimmed) > 0 && (trimmed[0] == '`' || trimmed[0] == '~') {
			marker = trimmed[0]
			for count < len(trimmed) && trimmed[count] == marker {
				count++
			}
		}
		if fence != 0 {
			kept = append(kept, line)
			if marker == fence && count >= fenceLength && strings.TrimSpace(trimmed[count:]) == "" {
				fence = 0
			}
			continue
		}
		if count >= 3 {
			fence, fenceLength = marker, count
			kept = append(kept, line)
			continue
		}
		for _, ref := range session.ImageRefs(line) {
			if !seen[ref] {
				refs = append(refs, ref)
				seen[ref] = true
			}
		}
		if !strings.HasPrefix(trimmed, "[image] ") {
			kept = append(kept, line)
		}
	}
	return refs, strings.Join(kept, "\n")
}

// userMessageView follows MessageItem's userRow/userStack geometry. The two
// nested caps express min(chat-width * .702, 82% of the available row).
func userMessageView(c *ui.Context, m Model, msg session.Message, colors theme.Colors, refs []string, body string, onOpenImage func(string)) {
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	axis := m.ContentWidth
	if axis <= 0 {
		axis = 748
	}
	ui.Column(c).FillWidth().Gap(6).AlignItems(ui.End).Children(func() {
		ui.Column(c).FillWidth().MaxWidthPercent(82).MinWidth(0).AlignItems(ui.End).Children(func() {
			ui.Column(c).FillWidth().MaxWidth(axis * .702).MinWidth(0).Gap(8).AlignItems(ui.End).Children(func() {
				for _, ref := range refs {
					var open func(string)
					if m.OnOpenImageLightbox != nil || m.ImageLightboxPath != nil {
						open = onOpenImage
					}
					ImageInlineView(c, ref, fg, muted, open)
				}
				if body != "" {
					ui.Column(c).Label("User message bubble").MinWidth(0).MaxWidthPercent(100).
						Background(chrome.Color(colors.Bubble)).Radius(20).Padding(10, 16, 10, 16).Children(func() {
						ui.Text(c, body).FontSize(14).FixedLineHeight(22).TextColor(fg).Selectable()
					})
				}
				// Preserve the existing URL-opening action alongside literal user text.
				if m.OnOpenCitationURL != nil {
					links := markdown.Links(body)
					if len(links) > 0 {
						ui.Row(c).FillWidth().Wrap().Gap(6).Justify(ui.End).Children(func() {
							for _, link := range links {
								label := "↗ " + link.Label
								button := ui.Button(c, "").Label(label).Tooltip(link.URL).MaxWidthPercent(100).MinWidth(0).
									FontSize(13).BorderWidth(0).Background(ui.Transparent).TextColor(chrome.Color(colors.Accent))
								button.Children(func() { ui.Text(c, label).Grow(1).MinWidth(0).SingleLine() })
								if button.Clicked() {
									m.OnOpenCitationURL(link.URL)
								}
							}
						})
					}
				}
			})
		})
		ui.Row(c).Label("User message actions").Gap(8).Height(28).AlignItems(ui.Center).Children(func() {
			if msg.EventTime > 0 {
				ui.Text(c, formatEventTime(msg.EventTime)).FontSize(13).FixedLineHeight(24).Padding(0, 12, 0, 0).TextColor(muted)
			}
			action := func(glyph, label string, run func()) {
				if ui.Button(c, glyph).Label(label).Tooltip(label).Width(28).Height(28).Padding(6).Radius(8).BorderWidth(0).Background(ui.Transparent).TextColor(muted).FontSize(15).Clicked() {
					run()
				}
			}
			if showMessageCopy(msg) {
				action("⎘", i18n.T(i18n.KeyChatCopy), func() { c.Services().WriteClipboard(msg.Body) })
			}
			if showMessageCopy(msg) && m.OnQuoteMessage != nil {
				action("↗", i18n.T(i18n.KeyChatQuote), func() { m.OnQuoteMessage(msg.Body) })
			}
			if showMessageEdit(msg) && m.OnEditMessage != nil {
				action("✎", i18n.T(i18n.KeyChatEdit), func() { m.OnEditMessage(msg.Seq, msg.Body) })
			}
		})
	})
}
