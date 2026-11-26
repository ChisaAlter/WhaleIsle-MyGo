package launcher

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/components"
	"whaleisle/internal/harness"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

func hostBundleCard(c *ui.Context, pal launcherPalette, m Model, row components.Row, compact bool) {
	pending := m.ComponentPending[row.ID]
	busy := m.ComponentBusy || m.ImportBusy || m.StartBusy || pending != "" || m.HostRuntimeState == harness.Starting
	chrome.FrostSurface(ui.Column(c).FillWidth().Padding(12).Gap(8).Border(1, pal.borderL2), pal.colors, m.SystemDark, pal.sidebar, 12).Children(func() {
		title := releaseText("Host 离线运行时", "Host offline runtime")
		if row.InstalledVersion != "" {
			title += " · v" + row.InstalledVersion
		}
		ui.Text(c, title).FontSize(theme.InterfaceSize(15)).FontWeight(600).TextColor(pal.primary)
		if pending != "" {
			ui.Text(c, componentPendingLabel(pending)).FontSize(theme.InterfaceSize(12)).TextColor(pal.secondary)
		}
		ui.Text(c, releaseText("先安装离线包，再选择版本。Host 和 Node 一起切换；启动使用现有内核生命周期。", "Install the offline bundle, then select a version. Host and Node switch together and use the existing runtime lifecycle.")).FontSize(theme.InterfaceSize(12)).TextColor(pal.secondary)
		if row.Message != "" {
			ui.Text(c, row.Message).FontSize(theme.InterfaceSize(12)).TextColor(pal.secondary)
		}
		ui.Row(c).Wrap().Gap(8).Children(func() {
			if !compact {
				if ghostSmall(c, pal, releaseText("安装离线包", "Install offline bundle")).Disabled(busy).Clicked() && m.OnComponentAction != nil {
					m.OnComponentAction(row.ID, "install")
				}
				if ghostSmall(c, pal, releaseText("选择已安装版本", "Select installed version")).Disabled(busy).Clicked() && m.OnComponentAction != nil {
					m.OnComponentAction(row.ID, "select")
				}
				if ghostSmall(c, pal, releaseText("打开版本目录", "Open versions folder")).Disabled(busy).Clicked() && m.OnComponentAction != nil {
					m.OnComponentAction(row.ID, "open")
				}
			}
			if m.HostRuntimeState == harness.Ready || m.HostRuntimeState == harness.Starting {
				if ghostSmall(c, pal, releaseText("停止 Host", "Stop Host")).Disabled(m.ComponentBusy || m.ImportBusy || pending != "").Clicked() && m.OnStop != nil {
					m.OnStop()
				}
			} else if row.InstalledVersion != "" {
				if primaryBtn(c, pal, releaseText("启动 Host", "Start Host")).Disabled(busy).Clicked() && m.OnStart != nil {
					m.OnStart()
				}
			}
		})
	})
}
