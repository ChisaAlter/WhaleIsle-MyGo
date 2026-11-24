package chat

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/widgets/markdown"
)

func toolReadCard(c *ui.Context, m *toolReadModel, _ ToolCardStyle, key string) {
	t := design.For(c)
	style := markdown.Style{
		FontSize:         design.TextSM.Size,
		FG:               t.CodeForeground,
		Muted:            t.MutedForeground,
		CodeBG:           t.Code,
		Dark:             t.IsDark(),
		ToolCodeLanguage: t.MutedForeground,
		ToolCodeHover:    t.Accent,
		CopyLabel:        i18n.T(i18n.KeyChatToolCopy),
		CopiedLabel:      i18n.T(i18n.KeyToolReadCopied),
		WrapLabel:        i18n.T(i18n.KeyMarkdownWrap),
		NoWrapLabel:      i18n.T(i18n.KeyToolReadUnwrap),
		CodeLabel:        i18n.T(i18n.KeyToolReadCodeLabel),
	}
	lines := make([]markdown.ReadLine, len(m.Lines))
	for i, line := range m.Lines {
		lines[i] = markdown.ReadLine{Number: line.Number, Text: line.Text}
	}
	labels := markdown.ReadLabels{
		Window:       toolReadWindowLabel,
		Expand:       toolReadExpandRestLabel,
		ExpandAria:   toolReadExpandAriaLabel,
		Collapse:     i18n.T(i18n.KeyChatRowCollapse),
		CollapseAria: toolReadCollapseAriaLabel(),
	}
	markdown.ToolReadBlock(c.Key("tool-read:"+key), m.Label, m.Lang, lines, m.TotalLines, 8, style, labels).
		Margin(design.Sp(1), 0, design.Sp(1), design.Sp(1)).Label(i18n.T(i18n.KeyToolReadCard))
}

func toolReadWindowLabel(shown, total int) string {
	return fmt.Sprintf(referenceCopy("Showing %d of %d lines", "显示 %d / %d 行"), shown, total)
}

func toolReadExpandRestLabel(n int) string {
	return fmt.Sprintf(referenceCopy("… %d more lines", "… 其余 %d 行"), n)
}

func toolReadExpandAriaLabel(n int) string {
	return fmt.Sprintf(referenceCopy("Expand %d more lines", "展开其余 %d 行"), n)
}

func toolReadCollapseAriaLabel() string {
	return referenceCopy("Collapse content", "收起内容")
}
