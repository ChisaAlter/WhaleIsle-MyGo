package pluginmanager

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	manager "whaleisle/internal/pluginmanager"
	"whaleisle/internal/ui/theme"
)

// Folding changes only the projection. The controller retains the full bounded
// run output, and copying always copies that output, including hidden lines.
func installTerminal(c *ui.Context, run manager.InstallRun) {
	card := ui.Column(c).FillWidth().MinWidth(0).Gap(4).Padding(10).Border(.5, c.Theme().Border).Radius(6)
	expanded := ui.Local(card, "expanded", func() bool { return false })
	copied := ui.Local(card, "copied-until", func() time.Time { return time.Time{} })
	copying := time.Now().Before(*copied)
	if copying {
		c.After(time.Until(*copied))
	}
	card.Children(func() {
		ui.Row(c).FillWidth().Gap(8).Children(func() {
			ui.Text(c, run.Command).Grow(1).MinWidth(0).Selectable().FontSize(theme.InterfaceSize(12)).FontWeight(500).TextColor(c.Theme().Text)
			label := copyText("复制", "Copy")
			if copying {
				label = copyText("已复制", "Copied")
			}
			if run.Output != "" && ui.Button(c, label).Disabled(copying).Clicked() {
				c.WriteClipboard(run.Output)
				*copied = time.Now().Add(time.Second)
			}
		})
		if run.CWD != "" {
			ui.Text(c, run.CWD).FillWidth().Selectable().FontSize(theme.InterfaceSize(11)).TextColor(c.Theme().TextMuted)
		}
		lines := installANSI(run.Output, c.Theme())
		visible := false
		for _, line := range lines {
			for _, span := range line {
				if strings.TrimSpace(span.Text) != "" {
					visible = true
				}
			}
		}
		if !visible {
			if run.Settled {
				ui.Text(c, copyText("暂无输出", "No output")).TextColor(c.Theme().TextMuted)
			}
		} else {
			spans := []ui.Span{}
			for n, line := range lines {
				if len(lines) > 12 && !*expanded && n >= 6 && n < len(lines)-6 {
					if n == 6 {
						spans = append(spans, ui.Span{Text: "…\n"})
					}
					continue
				}
				spans = append(spans, line...)
				if n < len(lines)-1 {
					spans = append(spans, ui.Span{Text: "\n"})
				}
			}
			ui.ScrollHorizontal(c).FillWidth().MinWidth(0).Children(func() {
				ui.RichText(c, spans...).NoWrap().Shrink(0).Selectable().Font("monospace").FontSize(theme.InterfaceSize(12)).TextColor(c.Theme().TextMuted)
			})
		}
		if visible && len(lines) > 12 {
			label := fmt.Sprintf(copyText("展开隐藏的 %d 行", "Expand %d hidden lines"), len(lines)-12)
			if *expanded {
				label = copyText("收起输出", "Collapse output")
			}
			if ui.Button(c, label).Clicked() {
				*expanded = !*expanded
			}
		}
		if run.Truncated {
			ui.Text(c, copyText("输出已截断", "Output truncated")).TextColor(c.Theme().TextMuted)
		}
		status := copyText("运行中", "Running")
		if run.Settled {
			status = copyText("已结束（未提供退出代码）", "Ended without an exit code")
			if run.ExitCode != nil {
				status = fmt.Sprintf("Exit: %d", *run.ExitCode)
			}
		}
		ui.Text(c, status).Role(ui.RoleStatus).TextColor(c.Theme().TextMuted)
	})
}
