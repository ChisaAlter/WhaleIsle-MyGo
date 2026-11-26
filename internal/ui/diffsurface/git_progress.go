package diffsurface

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/desktop/git"
	"whaleisle/internal/desktop/webview"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	fonttheme "whaleisle/internal/ui/theme"
)

// DrawGitProgress shows bounded, operation-owned progress; dismissal never stops
// a mutating Git command. Only message preparation has a cancellation action.
func DrawGitProgress(c *ui.Context, p *git.Progress, fg, muted ui.Color, onDismiss func()) {
	drawGitProgress(c, p, fg, muted, onDismiss)
}

func drawGitProgress(c *ui.Context, p *git.Progress, fg, muted ui.Color, onDismiss func()) bool {
	if p == nil {
		return false
	}
	s := p.Snapshot()
	leftPad, rightPad := chrome.TitleBarSafeHorizontalPadding(c.TitleBar(), design.Sp(3))
	column := ui.Column(c).Key(p).FillWidth().MinWidth(0).Shrink(0).Padding(design.Sp(1), rightPad, design.Sp(1), leftPad).Gap(design.Sp(0.5))
	column.Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Gap(design.Sp(1.5)).Children(func() {
			ui.Text(c, s.Phase).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg).Grow(1).MinWidth(0).MaxLines(2)
			if s.Done && onDismiss != nil {
				if coss.Button(c, word("关闭进度", "Dismiss progress"), coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonXS, Key: "git-progress-dismiss"}).Shrink(0).Clicked() {
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
		ui.Row(c).Gap(design.Sp(1.5)).FillWidth().MinWidth(0).Wrap().Children(func() {
			if details != "" {
				caption := word("展开详情", "Show details")
				if *expanded {
					caption = word("收起详情", "Hide details")
				}
				if coss.Button(c, caption, coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonXS, Key: "git-progress-toggle"}).Shrink(0).Clicked() {
					*expanded = !*expanded
				}
				if s.Failed && coss.Button(c, word("复制完整错误", "Copy full error"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonXS, Key: "git-progress-copy"}).Shrink(0).Clicked() {
					mygo.Clipboard.WriteText(details)
				}
			}
			if s.Done && !s.Failed && s.ActionURL != "" && coss.Button(c, word("打开仓库", "Open repository"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonXS, Key: "git-progress-open-repo"}).Shrink(0).Clicked() {
				if target, err := git.RepositoryWebURL(s.ActionURL); err == nil {
					if err = webview.OpenExternal(target); err != nil {
						p.SetDetails(details + "\n" + err.Error()) // verbatim-diagnostics: this panel exists to show the failure word for word
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
	return s.Done
}
