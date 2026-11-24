package usage

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"

	usagestore "whaleisle/internal/store/usage"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/pagekit"
)

// embeddedView is the usage-stats settings contribution. Account and jobs retain
// their own surfaces; this body follows dsh-usage-panel's StatsSection contract.
func embeddedView(c *ui.Context, m Model) {
	pagekit.Page(c, pagekit.PageOptions{Key: "usage-stats-body", Embedded: true}, func() {
		usageHeader(c, m, usageText("用量与费用", "Usage & cost"))
		renderPreferences(c, m)
		if m.ActionError != "" {
			pagekit.FaultSentence(c, "", m.ActionError, "", nil)
		}
		if m.Store == nil {
			usageEmpty(c, "", usageText("连接 Host 后即可读取用量统计。", "Connect to a Host to read usage statistics."))
			return
		}
		statsView(c, m)
	})
}

func renderPreferences(c *ui.Context, m Model) {
	if m.Preferences == nil {
		return
	}
	t := design.For(c)
	ui.Column(c).FillWidth().MinWidth(0).Gap(0).Children(func() {
		m.Preferences(c, t.Foreground, t.MutedForeground)
	})
}

// usageSourceLine says where the totals came from in words a person can read,
// and keeps the log file's own path behind a details disclosure: the path is how
// a person points support at the record, not the answer to "what am I looking at".
func usageSourceLine(c *ui.Context, source string) {
	text := usageText("统计来源：本机网关用量记录", "Source: this machine's gateway usage log")
	if source == "" {
		text = usageText("统计来源：会话日志", "Source: session logs")
	}
	row := ui.Row(c).FillWidth().MinWidth(0).Gap(design.Sp(1.5)).AlignItems(ui.Center)
	open := ui.Local(row, "usage-source-details", func() bool { return false })
	row.Children(func() {
		pagekit.Caption(c, text).Grow(1).MinWidth(0)
		if source == "" {
			return
		}
		icon := coss.IconChevronRight
		if *open {
			icon = coss.IconChevronDown
		}
		btn := coss.Button(c, usageText("详情", "Details"), coss.ButtonOptions{
			Variant: coss.ButtonGhost,
			Size:    coss.ButtonXS,
			Icon:    icon,
		}).Role(ui.RoleDisclosure).Expanded(*open)
		if btn.Clicked() {
			*open = !*open
		}
	})
	if source != "" && *open {
		pagekit.Caption(c, usageText("用量记录文件：", "Usage log file: ")+source).FillWidth().Selectable()
	}
}

func usageSourceLabel(o *usagestore.Overview) (string, string) {
	if o.Coverage.Source == "magpie" && o.Coverage.Magpie != nil {
		m := o.Coverage.Magpie
		day := func(at *int64) string {
			if at == nil {
				return "—"
			}
			return time.UnixMilli(*at).UTC().Format("2006-01-02")
		}
		return fmt.Sprintf(usageText("总计口径：Magpie · %d 次调用 · %s → %s UTC", "Totals basis: Magpie · %d calls · %s → %s UTC"), m.Calls, day(m.From), day(m.To)), m.File
	}
	return usageText("总计口径：会话日志（输入+输出，fork 继承前缀已去重）", "Totals basis: session logs (input + output, fork-inherited prefixes de-duplicated)"), ""
}
