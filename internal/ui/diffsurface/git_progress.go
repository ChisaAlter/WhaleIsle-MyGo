package diffsurface

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/desktop/git"
	"whaleisle/internal/desktop/webview"
	fonttheme "whaleisle/internal/ui/theme"
)

// DrawGitProgress shows bounded, operation-owned progress; dismissal never stops
// a mutating Git command. Only message preparation has a cancellation action.
func DrawGitProgress(c *ui.Context, p *git.Progress, fg, muted ui.Color, onDismiss func()) {
	if p == nil {
		return
	}
	s := p.Snapshot()
	column := ui.Column(c).Key(p).FillWidth().Padding(4, 12, 4, 12).Gap(2)
	column.Children(func() {
		ui.Row(c).FillWidth().Gap(6).Children(func() {
			ui.Text(c, s.Phase).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg).Grow(1).MaxLines(2)
			if s.Done && onDismiss != nil {
				if ui.Button(c, word("关闭进度", "Dismiss progress")).FontSize(fonttheme.InterfaceSize(10)).Clicked() {
					onDismiss()
				}
			}
		})
		ui.Text(c, s.Root).FontSize(fonttheme.InterfaceSize(10)).TextColor(muted).MaxLines(1)
		lines := s.Lines
		if len(lines) > 5 {
			lines = lines[len(lines)-5:]
		}
		details := s.Details
		if details == "" {
			details = strings.Join(s.Lines, "\n")
		}
		expanded := ui.Local(column, "git-progress-details", func() bool { return false })
		ui.Row(c).Gap(6).FillWidth().Children(func() {
			if details != "" {
				caption := word("展开详情", "Show details")
				if *expanded {
					caption = word("收起详情", "Hide details")
				}
				if ui.Button(c, caption).FontSize(fonttheme.InterfaceSize(12)).Clicked() {
					*expanded = !*expanded
				}
				if s.Failed && ui.Button(c, word("复制完整错误", "Copy full error")).FontSize(fonttheme.InterfaceSize(12)).Clicked() {
					mygo.Clipboard.WriteText(details)
				}
			}
			if s.Done && !s.Failed && s.ActionURL != "" && ui.Button(c, word("打开仓库", "Open repository")).FontSize(fonttheme.InterfaceSize(12)).Clicked() {
				if target, err := git.RepositoryWebURL(s.ActionURL); err == nil {
					if err = webview.OpenExternal(target); err != nil {
						p.SetDetails(details + "\n" + err.Error())
					}
				}
			}
		})
		if *expanded && details != "" {
			ui.ScrollBoth(c).FillWidth().Height(180).Children(func() { ui.Text(c, details).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable() })
			return
		}
		for _, line := range lines {
			ui.Text(c, line).FontSize(fonttheme.InterfaceSize(10)).TextColor(muted).MaxLines(2)
		}
	})
}
