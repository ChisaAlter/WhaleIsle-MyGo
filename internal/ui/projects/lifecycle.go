package projects

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	fonttheme "whaleisle/internal/ui/theme"
)

func archiveConfirmation(c *ui.Context, m Model) {
	p := m.ArchiveCandidate
	ui.Text(c, tr("停止后台工作并归档 Project", "Stop work and archive Project")).FontSize(fonttheme.InterfaceSize(13))
	ui.Text(c, p.Title).FontSize(fonttheme.InterfaceSize(13))
	ui.Text(c, tr("后台工作会停止，工作目录、对话和成果将保留。这是归档整个Project，不是仅归档一张要求卡片。", "Background work will stop. Working directory, conversation and results are retained. This archives the whole Project, rather than one request card.")).FontSize(fonttheme.InterfaceSize(12))
	if p.Activity.State == "" {
		ui.Text(c, tr("Host未提供完整活动信息；确认归档将尝试停止该Project的工作。", "Host activity details are incomplete; confirming archive will attempt to stop this Project’s work.")).FontSize(fonttheme.InterfaceSize(11))
	} else {
		ui.Text(c, fmt.Sprintf(tr("Lead运行：%t；成员运行：%d；排队：%d；准备中：%d；状态：%s", "Lead running: %t; members running: %d; queued: %d; preparing: %d; state: %s"), p.Activity.CoordinatorRunning, p.Activity.Running, p.Activity.Queued, p.Activity.Provisioning, p.Activity.State)).FontSize(fonttheme.InterfaceSize(11))
	}
	ui.Row(c).Gap(4).Children(func() {
		if ui.Button(c, tr("取消", "Cancel")).Disabled(m.Busy).Clicked() && m.OnCancelArchive != nil {
			m.OnCancelArchive()
		}
		if ui.Button(c, tr("停止工作并归档", "Stop work and archive")).Disabled(m.Busy || !m.Listing.Available).Clicked() && m.OnConfirmArchive != nil {
			m.OnConfirmArchive()
		}
	})
}
