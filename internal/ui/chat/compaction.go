package chat

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets/markdown"
)

var (
	compactionTerminalIcon     = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke-width="1"><path d="M3 4L7 8L3 12" stroke="currentColor" /><path d="M9 12H13" stroke="currentColor" /></svg>`))
	compactionChevronRightIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke-width="1"><path d="M6 12L9.29289 8.70711C9.68342 8.31658 9.68342 7.68342 9.29289 7.29289L6 4" stroke="currentColor" /></svg>`))
	compactionChevronDownIcon  = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke-width="1"><path d="M4 6L7.29289 9.29289C7.68342 9.68342 8.31658 9.68342 8.70711 9.29289L12 6" stroke="currentColor" /></svg>`))
)

// CompactionView renders an inline 24px context compaction row matching
// CompactionItem.tsx in @deepseek-harness/ui-chat. Missing history evidence
// stays in the compact single-line row and never falls back to a raw body card.
func CompactionView(c *ui.Context, msg session.Message, fg, muted, field ui.Color) {
	root := ui.Column(c).FillWidth().MinWidth(0).Padding(2, 0).Clip()
	open := ui.Local(root, "summary-open", func() bool { return false })

	dark := darkSurface(fg)
	secondary := chrome.Color(theme.SecondaryInk(dark))
	caption := compactionCaptionColor(dark, muted)

	compact := msg.AutomaticCompaction
	summary := ""
	if compact != nil {
		summary = strings.TrimSpace(compact.Summary)
	}
	expandable := summary != ""
	isExpanded := expandable && *open

	label := i18n.T(i18n.KeyChatCompactionSummaryMissing)
	if compact != nil && compact.ShadowedItemCount != nil && compact.ShadowedTokenCount != nil {
		label = fmt.Sprintf(i18n.T(i18n.KeyChatCompactionCounts), *compact.ShadowedItemCount, *compact.ShadowedTokenCount)
	} else if expandable {
		label = i18n.T(i18n.KeyPlanBoardExpand)
	}

	root.Children(func() {
		header := ui.Row(c).Height(24).FillWidth().MinWidth(0).AlignItems(ui.Center).
			Label(i18n.T(i18n.KeyChatCompaction))
		if expandable {
			header.Cursor(ui.CursorPointer)
		} else {
			header.Cursor(ui.CursorDefault)
		}

		active := expandable && (header.Hovered() || isExpanded)
		titleColor := secondary
		if active {
			titleColor = fg
		}

		header.Children(func() {
			ui.Row(c).Width(22).Height(24).Shrink(0).AlignItems(ui.Center).Children(func() {
				icon := compactionTerminalIcon
				if active {
					icon = compactionChevronRightIcon
					if isExpanded {
						icon = compactionChevronDownIcon
					}
				}
				ui.Icon(c, icon).Size(14, 14).TextColor(titleColor).PassThrough()
			})
			ui.Row(c).Grow(1).Height(24).MinWidth(0).AlignItems(ui.Center).Children(func() {
				ui.Text(c, i18n.T(i18n.KeyChatCompaction)).FontSize(13).FixedLineHeight(24).
					TextColor(titleColor).NoWrap().Shrink(0)
				ui.Box(c).Size(2, 2).Radius(999).Background(caption).Margin(0, 8, 0, 8).Shrink(0)
				ui.Text(c, label).FontSize(13).FixedLineHeight(24).
					TextColor(caption).NoWrap().Ellipsis("…").Grow(1).MinWidth(0)
			})
		})

		if expandable && header.Clicked() {
			*open = !*open
		}
		if isExpanded {
			ui.Column(c).FillWidth().MinWidth(0).Padding(4, 0, 4, 22).Children(func() {
				markdown.View(c, compact.Summary, markdown.Style{
					FG:       secondary,
					Muted:    caption,
					CodeBG:   field,
					Accent:   c.Theme().Accent,
					FontSize: 13,
					Dark:     dark,
				})
			})
		}
	})
}

func compactionCaptionColor(dark bool, fallback ui.Color) ui.Color {
	if dark {
		return ui.RGB(129, 133, 140)
	}
	if fallback.A != 0 {
		return ui.RGB(173, 178, 184)
	}
	return ui.RGB(173, 178, 184)
}
