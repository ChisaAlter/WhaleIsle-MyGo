package pluginmanager

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	manager "whaleisle/internal/pluginmanager"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/pagekit"
)

// terminalLines caches one run's parsed output: the ANSI replay runs when
// the output or the tokens change, not every frame.
type terminalLines struct {
	output  string
	tokens  *design.Tokens
	lines   [][]ui.Span
	visible bool
}

func (l *terminalLines) of(raw string, t *design.Tokens) *terminalLines {
	if l.tokens == t && l.output == raw && l.lines != nil {
		return l
	}
	l.output, l.tokens = raw, t
	l.lines = installANSI(raw, t)
	l.visible = false
	for _, line := range l.lines {
		for _, span := range line {
			if strings.TrimSpace(span.Text) != "" {
				l.visible = true
			}
		}
	}
	return l
}

// installTerminal is one run's output card. Folding changes only the
// projection: the controller keeps the full bounded output, and Copy always
// copies that output, hidden lines included.
func installTerminal(c *ui.Context, run manager.InstallRun) {
	t := design.For(c)
	card := ui.Column(c).FillWidth().MinWidth(0).Gap(design.Sp(2)).Padding(design.Sp(3)).Radius(design.RadiusLG).Border(design.BorderWidth, t.Border).Background(t.Muted)
	expanded := ui.Local(card, "expanded", func() bool { return false })
	copied := ui.Local(card, "copied-until", func() time.Time { return time.Time{} })
	cache := ui.Local(card, "lines", func() *terminalLines { return &terminalLines{} })
	copying := time.Now().Before(*copied)
	if copying {
		c.After(time.Until(*copied))
	}
	card.Children(func() {
		ui.Row(c).FillWidth().Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
			pagekit.Code(c, run.Command).Grow(1).MinWidth(0).Selectable().TextColor(t.Foreground)
			label, icon := copyText("复制", "Copy"), coss.IconCopy
			if copying {
				label, icon = copyText("已复制", "Copied"), coss.IconCheck
			}
			if run.Output != "" && coss.Button(c, label, coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonXS, Icon: icon, Disabled: copying}).Clicked() {
				c.WriteClipboard(run.Output)
				*copied = time.Now().Add(time.Second)
			}
		})
		if run.CWD != "" {
			pagekit.Code(c, run.CWD).FillWidth().Selectable()
		}
		l := (*cache).of(run.Output, t)
		if !l.visible {
			if run.Settled {
				pagekit.Caption(c, copyText("暂无输出", "No output"))
			}
		} else {
			spans := []ui.Span{}
			for n, line := range l.lines {
				if len(l.lines) > 12 && !*expanded && n >= 6 && n < len(l.lines)-6 {
					if n == 6 {
						spans = append(spans, ui.Span{Text: "…\n"})
					}
					continue
				}
				spans = append(spans, line...)
				if n < len(l.lines)-1 {
					spans = append(spans, ui.Span{Text: "\n"})
				}
			}
			ui.ScrollHorizontal(c).FillWidth().MinWidth(0).Children(func() {
				ui.RichText(c, spans...).NoWrap().Shrink(0).Selectable().Font("monospace").FontSize(design.TextXS.Size).FixedLineHeight(design.TextXS.LineHeight).TextColor(t.MutedForeground)
			})
		}
		if l.visible && len(l.lines) > 12 {
			label := fmt.Sprintf(copyText("展开隐藏的 %d 行", "Expand %d hidden lines"), len(l.lines)-12)
			if *expanded {
				label = copyText("收起输出", "Collapse output")
			}
			if coss.Button(c, label, coss.ButtonOptions{Variant: coss.ButtonLink, Size: coss.ButtonXS}).AlignSelf(ui.Start).Clicked() {
				*expanded = !*expanded
			}
		}
		if run.Truncated {
			pagekit.Caption(c, copyText("输出已截断", "Output truncated"))
		}
		status, variant := copyText("运行中", "Running"), coss.BadgeInfo
		if run.Settled {
			status, variant = copyText("已结束（未提供退出代码）", "Ended without an exit code"), coss.BadgeSecondary
			if run.ExitCode != nil {
				status = fmt.Sprintf("Exit: %d", *run.ExitCode)
				variant = coss.BadgeSuccess
				if *run.ExitCode != 0 {
					variant = coss.BadgeError
				}
			}
		}
		coss.Badge(c, status, coss.BadgeOptions{Variant: variant}).Role(ui.RoleStatus)
	})
}
