package chat

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
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
		t := design.For(c)
		style := markdown.Style{
			FontSize:         14,
			FG:               t.CodeForeground,
			Muted:            t.MutedForeground,
			CodeBG:           t.Code,
			Dark:             t.IsDark(),
			ToolCodeLanguage: t.MutedForeground,
			ToolCodeHover:    t.Accent,
			CopyLabel:        i18n.T(i18n.KeyChatToolCopy),
			CopiedLabel:      i18n.T(i18n.KeyMarkdownCopied),
			WrapLabel:        i18n.T(i18n.KeyMarkdownWrap),
			NoWrapLabel:      i18n.T(i18n.KeyMarkdownNoWrap),
			CodeLabel:        i18n.T(i18n.KeyMarkdownCode),
		}
		seat := ui.Column(c.Key("tool-code:" + key)).MinWidth(0).Shrink(0)
		position := ui.Local(seat, "position", func() ui.ScrollState { return ui.ScrollState{} })
		seat.Children(func() {
			coss.ScrollArea(c, coss.ScrollAreaOptions{State: position}, func() {
				ui.Column(c).MinWidth(0).Shrink(0).Margin(design.Sp(1), 0, design.Sp(1), design.Sp(1)).Children(func() {
					markdown.ToolCodeBlock(c, "typescript", code, style, position.Y).Label(i18n.T(i18n.KeyToolCodeCard))
				})
			}).MinWidth(0).Shrink(0).MaxHeight(260).Label(i18n.T(i18n.KeyToolCodeProgram))
		})
	}
	// The code variant owns its input card; only result text enters generic IO.
	info.Arguments = ""
	toolIORows(c, info, st, key+":result")
}
