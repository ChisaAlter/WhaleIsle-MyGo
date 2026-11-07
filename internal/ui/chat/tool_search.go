package chat

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets/markdown"
)

func toolSearchCard(c *ui.Context, m *toolSearchModel, st ToolCardStyle, key string) {
	data := markdown.SearchData{Kind: m.Kind, Paths: m.Paths, Total: m.Total, Truncated: m.Truncated, Files: make([]markdown.SearchFile, len(m.Files))}
	for i, file := range m.Files {
		data.Files[i] = markdown.SearchFile{Path: file.Path, Matches: make([]markdown.SearchMatch, len(file.Matches))}
		for j, match := range file.Matches {
			data.Files[i].Matches[j] = markdown.SearchMatch{LineNumber: match.LineNumber, Line: match.Line}
		}
	}
	background, secondary := st.Field, st.Muted
	if st.Colors.CodeBlock != "" {
		background = chrome.Color(st.Colors.CodeBlock)
		secondary = chrome.Color(st.Colors.Secondary)
	}
	banner := ui.Hex("#f9fafb")
	if darkSurface(st.FG) {
		banner = ui.Hex("#2c2c2e")
	}
	style := markdown.Style{FG: st.FG, Muted: secondary, CodeBG: background, CodeBanner: banner, ToolCodeLanguage: st.Muted}
	labels := markdown.SearchLabels{
		PathsSummary:   toolSearchPathsSummary,
		MatchesSummary: toolSearchMatchesSummary,
		Copy: i18n.T(i18n.KeyChatToolCopy), Copied: i18n.T(i18n.KeyToolReadCopied), NoResults: i18n.T(i18n.KeyToolSearchEmpty), Collapse: i18n.T(i18n.KeyChatRowCollapse), CollapseAria: toolSearchCollapseAriaLabel(),
		Expand: toolSearchExpandRestLabel, ExpandAria: toolSearchExpandAriaLabel,
	}
	markdown.ToolSearchBlock(c.Key("tool-search:"+key), data, 8, style, labels).Margin(4, 0, 4, 4).Label(i18n.T(i18n.KeyToolSearchCard))
	if m.Recovery != "" {
		prefs := theme.RenderPreferences()
		ui.Text(c, m.Recovery).Font(prefs.Interface.Family).FontSize(13).FixedLineHeight(20).TextColor(st.Muted).MinWidth(0).Shrink(0).Selectable().Margin(4, 0, 4, 4)
	}
}

func toolSearchPathsSummary(shown int, total float64, truncated bool) string {
	if truncated {
		return fmt.Sprintf(referenceCopy("Showing %d of %s paths", "显示 %d / 共 %s 个路径"), shown, markdown.SearchNumber(total))
	}
	return fmt.Sprintf(referenceCopy("%d paths", "%d 个路径"), shown)
}

func toolSearchMatchesSummary(shown int, total float64, files int, truncated bool) string {
	if truncated {
		return fmt.Sprintf(referenceCopy("Showing %d of %s matches · %d files", "显示 %d / 共 %s 处匹配 · %d 个文件"), shown, markdown.SearchNumber(total), files)
	}
	return fmt.Sprintf(referenceCopy("%d matches · %d files", "%d 处匹配 · %d 个文件"), shown, files)
}

func toolSearchExpandRestLabel(n int) string {
	return fmt.Sprintf(referenceCopy("… %d more lines", "… 其余 %d 行"), n)
}

func toolSearchExpandAriaLabel(n int) string {
	return fmt.Sprintf(referenceCopy("Expand %d more result lines", "展开其余 %d 行结果"), n)
}

func toolSearchCollapseAriaLabel() string {
	return referenceCopy("Collapse results", "收起结果")
}
