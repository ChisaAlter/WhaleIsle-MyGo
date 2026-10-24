package chat

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/widgets/markdown"
)

// toolCodeProgram matches the code variant's formatToolBody: empty raw input
// mounts no program; invalid JSON remains literal; only a nonempty string code
// field replaces the JSON envelope. Whitespace is authored program text.
func toolCodeProgram(arguments string) (string, bool) {
	return formatToolCodeProgram(arguments)
}

func toolCodeBody(c *ui.Context, info ToolCardInfo, st ToolCardStyle, key string) {
	if code, present := toolCodeProgram(info.Arguments); present {
		background, secondary := st.Field, st.Muted
		if st.Colors.CodeBlock != "" {
			background = chrome.Color(st.Colors.CodeBlock)
			secondary = chrome.Color(st.Colors.Secondary)
		}
		style := markdown.Style{FontSize: 14, FG: st.FG, Muted: secondary, CodeBG: background, Dark: darkSurface(st.FG),
			ToolCodeLanguage: st.Muted, ToolCodeHover: chrome.Color(st.Colors.Hover),
			CopyLabel: i18n.T(i18n.KeyChatToolCopy), CopiedLabel: i18n.T(i18n.KeyMarkdownCopied),
			WrapLabel: i18n.T(i18n.KeyMarkdownWrap), NoWrapLabel: i18n.T(i18n.KeyMarkdownNoWrap), CodeLabel: i18n.T(i18n.KeyMarkdownCode)}
		scroll := ui.Scroll(c.Key("tool-code:" + key)).MinWidth(0).Shrink(0).MaxHeight(260).Label(i18n.T(i18n.KeyToolCodeProgram))
		position := ui.Local(scroll, "position", func() ui.ScrollState { return ui.ScrollState{} })
		scroll.TrackScroll(position).Children(func() {
			ui.Column(c).MinWidth(0).Shrink(0).Margin(4, 0, 4, 4).Children(func() {
				markdown.ToolCodeBlock(c, "typescript", code, style, position.Y).Label(i18n.T(i18n.KeyToolCodeCard))
			})
		})
	}
	// The code variant owns its input card; only result text enters generic IO.
	info.Arguments = ""
	toolIORows(c, info, st, key+":result")
}
