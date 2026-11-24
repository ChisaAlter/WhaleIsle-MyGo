package chat

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/widgets/markdown"
)

func toolDiffCard(c *ui.Context, m *toolDiffModel, st ToolCardStyle, key string) {
	rows := toolDiffRows(m.Diffs)
	if len(rows) == 0 {
		return
	}
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
	bgAlpha := float32(20.0 / 255)
	if style.Dark {
		bgAlpha = 31.0 / 255
	}
	palette := markdown.DiffPalette{
		Added:     t.SuccessForeground,
		Deleted:   t.DestructiveForeground,
		AddedBG:   t.Success.Alpha(bgAlpha),
		DeletedBG: t.Destructive.Alpha(bgAlpha),
	}
	labels := markdown.DiffLabels{
		Expand:       func(n int) string { return fmt.Sprintf(i18n.T(i18n.KeyToolDiffExpand), n) },
		ExpandAria:   func(n int) string { return fmt.Sprintf(i18n.T(i18n.KeyToolDiffExpandAria), n) },
		Collapse:     i18n.T(i18n.KeyChatRowCollapse),
		CollapseAria: i18n.T(i18n.KeyToolDiffCollapseAria),
	}
	lines := make([]markdown.DiffLine, len(rows))
	for i, row := range rows {
		lines[i] = markdown.DiffLine{Kind: row.Kind, Text: row.Text}
	}
	paths := make([]string, len(m.Diffs))
	for i, diff := range m.Diffs {
		paths[i] = diff.Path
	}
	markdown.ToolDiffBlock(c.Key("tool-diff:"+key), markdown.ToolDiffLanguage(paths), lines, toolDiffCopyText(rows), 9, style, palette, labels).
		Margin(design.Sp(1), 0, design.Sp(1), design.Sp(1)).Label(i18n.T(i18n.KeyToolDiffCard))
}
