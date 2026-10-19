package launcher

// 功能优先；全部功能完成前不运行测试、不做独立审查、不新增防御性扩展

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/components"
	"whaleisle/internal/harness"
	"whaleisle/internal/ui/theme"
)

func hostBundleCard(c *ui.Context, m Model, row components.Row, compact bool) {
	pending := m.ComponentPending[row.ID]
	busy := m.ComponentBusy || m.ImportBusy || m.StartBusy || pending != "" || m.HostRuntimeState == harness.Starting
	ui.Column(c).FillWidth().Padding(12).Gap(8).Background(colSidebar).Border(1, colBorderL2).Radius(12).Children(func() {
		title := releaseText("Host 离线运行时", "Host offline runtime")
		if row.InstalledVersion != "" {
			title += " · v" + row.InstalledVersion
		}
		ui.Text(c, title).FontSize(theme.InterfaceSize(15)).FontWeight(600).TextColor(colPrimary)
		if pending != "" {
			ui.Text(c, componentPendingLabel(pending)).FontSize(theme.InterfaceSize(12)).TextColor(colSecondary)
		}
		ui.Text(c, releaseText("先安装离线包，再选择版本。Host 和 Node 一起切换；启动使用现有内核生命周期。", "Install the offline bundle, then select a version. Host and Node switch together and use the existing runtime lifecycle.")).FontSize(theme.InterfaceSize(12)).TextColor(colSecondary)
		if row.Message != "" {
			ui.Text(c, row.Message).FontSize(theme.InterfaceSize(12)).TextColor(colSecondary)
		}
		ui.Row(c).Wrap().Gap(8).Children(func() {
			if !compact {
				if ghostSmall(c, releaseText("安装离线包", "Install offline bundle")).Disabled(busy).Clicked() && m.OnComponentAction != nil {
					m.OnComponentAction(row.ID, "install")
				}
				if ghostSmall(c, releaseText("选择已安装版本", "Select installed version")).Disabled(busy).Clicked() && m.OnComponentAction != nil {
					m.OnComponentAction(row.ID, "select")
				}
				if ghostSmall(c, releaseText("打开版本目录", "Open versions folder")).Disabled(busy).Clicked() && m.OnComponentAction != nil {
					m.OnComponentAction(row.ID, "open")
				}
			}
			if m.HostRuntimeState == harness.Ready || m.HostRuntimeState == harness.Starting {
				if ghostSmall(c, releaseText("停止 Host", "Stop Host")).Disabled(m.ComponentBusy || m.ImportBusy || pending != "").Clicked() && m.OnStop != nil {
					m.OnStop()
				}
			} else if row.InstalledVersion != "" {
				if primaryBtn(c, releaseText("启动 Host", "Start Host")).Disabled(busy).Clicked() && m.OnStart != nil {
					m.OnStart()
				}
			}
		})
	})
}
