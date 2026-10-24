package chat

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/widgets/markdown"
)

func toolReadCard(c *ui.Context, m *toolReadModel, st ToolCardStyle, key string) {
	background, secondary := st.Field, st.Muted
	if st.Colors.CodeBlock != "" {
		background = chrome.Color(st.Colors.CodeBlock)
		secondary = chrome.Color(st.Colors.Secondary)
	}
	style := markdown.Style{FontSize: 14, FG: st.FG, Muted: secondary, CodeBG: background, Dark: darkSurface(st.FG),
		ToolCodeLanguage: st.Muted, ToolCodeHover: chrome.Color(st.Colors.Hover),
		CopyLabel: i18n.T(i18n.KeyChatToolCopy), CopiedLabel: i18n.T(i18n.KeyToolReadCopied), WrapLabel: i18n.T(i18n.KeyMarkdownWrap), NoWrapLabel: i18n.T(i18n.KeyToolReadUnwrap), CodeLabel: i18n.T(i18n.KeyToolReadCodeLabel)}
	lines := make([]markdown.ReadLine, len(m.Lines))
	for i, line := range m.Lines {
		lines[i] = markdown.ReadLine{Number: line.Number, Text: line.Text}
	}
	labels := markdown.ReadLabels{Window: func(shown, total int) string { return fmt.Sprintf(i18n.T(i18n.KeyToolReadWindow), shown, total) },
		Expand: func(n int) string { return fmt.Sprintf(i18n.T(i18n.KeyToolReadExpand), n) }, ExpandAria: func(n int) string { return fmt.Sprintf(i18n.T(i18n.KeyToolReadExpandAria), n) },
		Collapse: i18n.T(i18n.KeyChatRowCollapse), CollapseAria: i18n.T(i18n.KeyToolReadCollapseAria)}
	markdown.ToolReadBlock(c.Key("tool-read:"+key), m.Label, m.Lang, lines, m.TotalLines, 8, style, labels).Margin(4, 0, 4, 4).Label(i18n.T(i18n.KeyToolReadCard))
}
