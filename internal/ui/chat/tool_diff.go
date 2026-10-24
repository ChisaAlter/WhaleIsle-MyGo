package chat

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/widgets/markdown"
)

func toolDiffCard(c *ui.Context, m *toolDiffModel, st ToolCardStyle, key string) {
	rows := toolDiffRows(m.Diffs)
	if len(rows) == 0 {
		return
	}
	background, secondary := st.Field, st.Muted
	if st.Colors.CodeBlock != "" {
		background = chrome.Color(st.Colors.CodeBlock)
		secondary = chrome.Color(st.Colors.Secondary)
	}
	style := markdown.Style{FontSize: 14, FG: st.FG, Muted: secondary, CodeBG: background, Dark: darkSurface(st.FG), ToolCodeLanguage: st.Muted, ToolCodeHover: chrome.Color(st.Colors.Hover),
		CopyLabel: i18n.T(i18n.KeyChatToolCopy), CopiedLabel: i18n.T(i18n.KeyToolReadCopied), WrapLabel: i18n.T(i18n.KeyMarkdownWrap), NoWrapLabel: i18n.T(i18n.KeyToolReadUnwrap), CodeLabel: i18n.T(i18n.KeyToolReadCodeLabel)}
	palette := markdown.DiffPalette{Added: ui.Hex("#22c55e"), Deleted: ui.Hex("#ec1313"), AddedBG: ui.Hex("#22c55e").Alpha(20.0 / 255), DeletedBG: ui.Hex("#ec1313").Alpha(20.0 / 255)}
	if style.Dark {
		palette.Deleted = ui.Hex("#f25a5a")
		palette.AddedBG = ui.Hex("#22c55e").Alpha(31.0 / 255)
		palette.DeletedBG = ui.Hex("#f25a5a").Alpha(31.0 / 255)
	}
	if st.Colors.Danger != "" {
		palette.Deleted = chrome.Color(st.Colors.Danger)
	}
	labels := markdown.DiffLabels{Expand: func(n int) string { return fmt.Sprintf(i18n.T(i18n.KeyToolDiffExpand), n) }, ExpandAria: func(n int) string { return fmt.Sprintf(i18n.T(i18n.KeyToolDiffExpandAria), n) }, Collapse: i18n.T(i18n.KeyChatRowCollapse), CollapseAria: i18n.T(i18n.KeyToolDiffCollapseAria)}
	lines := make([]markdown.DiffLine, len(rows))
	for i, row := range rows {
		lines[i] = markdown.DiffLine{Kind: row.Kind, Text: row.Text}
	}
	paths := make([]string, len(m.Diffs))
	for i, diff := range m.Diffs {
		paths[i] = diff.Path
	}
	markdown.ToolDiffBlock(c.Key("tool-diff:"+key), markdown.ToolDiffLanguage(paths), lines, toolDiffCopyText(rows), 9, style, palette, labels).Margin(4, 0, 4, 4).Label(i18n.T(i18n.KeyToolDiffCard))
}
