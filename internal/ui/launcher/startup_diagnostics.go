package launcher

import (
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/startuprec"
)

func startupCauseText(d startuprec.Diagnostics) string {
	switch d.Kind {
	case "oom":
		return releaseText("内存耗尽：先检查可用内存；不能据此归咎或禁用插件。", "Out of memory: check available memory; this does not identify a faulty plugin.")
	case "port-excluded":
		return releaseText("监听端口被系统策略拒绝：检查端口及安全策略。", "The listen port is denied by system policy; check the port and security policy.")
	case "port-in-use":
		return releaseText("端口已占用：更换端口或确认占用进程。", "The port is already in use; choose another port or identify the owner.")
	case "missing-node":
		return releaseText("缺少 Node 运行环境：在版本页选择含运行环境的发布。", "Node runtime is missing; choose a release with runtime on the Versions page.")
	case "session-cache":
		return releaseText("会话缓存结构不兼容：保留原文件后修复；不会自动删除会话数据。", "Session cache schema mismatch: preserve the original data before repair. No session data is deleted automatically.")
	case "runtime":
		return releaseText("证据指向内置运行时损坏；跳过用户插件无法修复。请在版本页选择匹配的运行环境。", "Evidence points to built-in runtime damage. Skipping user plugins cannot repair it; choose a matching runtime on the Versions page.")
	case "plugins":
		return releaseText("本次启动证据指向以下插件；仅已验证、可禁用的库存条目允许针对性修复。", "This startup implicates the plugins below. Only verified, mutable inventory entries can be repaired selectively.")
	default:
		return releaseText("暂无足够证据定位插件。可重试或检查当前启动日志。", "There is not enough evidence to identify a faulty plugin. Retry or inspect the current startup log.")
	}
}
func startupDiagnosticsPanel(c *ui.Context, m Model) {
	d := m.StartupDiagnostics
	ui.Column(c).FillWidth().Gap(6).Padding(10, 0, 10, 0).Children(func() {
		ui.Text(c, releaseText("启动诊断", "Startup diagnostics")).FontSize(14).FontWeight(600).TextColor(colPrimary)
		ui.Text(c, startupCauseText(d)).FontSize(12).TextColor(colSecondary)
		for _, e := range d.Evidence {
			label := e.Name
			if !e.Known {
				label += " · " + releaseText("未在当前库存确认", "not verified in inventory")
			}
			ui.Text(c, label).FontSize(12).FontWeight(600).TextColor(colPrimary)
			ui.Text(c, e.Line).FontSize(11).TextColor(colTertiary)
		}
		if !d.InventoryReady {
			ui.Text(c, releaseText("精确修复需要已就绪的 Host 插件控制器。可先启用跳过插件并启动恢复模式，再刷新库存；Host 无法启动时，请使用版本页修复运行环境。", "Selective repair needs a ready Host plugin controller. Start recovery mode with user plugins skipped, then refresh inventory. If the Host cannot start, use Versions to repair the runtime.")).FontSize(12).TextColor(colSecondary)
		}
		ui.Row(c).Gap(8).Children(func() {
			if ghostSmall(c, releaseText("刷新诊断与库存", "Refresh diagnostics and inventory")).Disabled(m.StartupRepairBusy).Clicked() && m.OnRefreshStartupDiagnostics != nil {
				m.OnRefreshStartupDiagnostics()
			}
			if primaryBtn(c, releaseText("禁用可疑插件并重试", "Disable suspects and retry")).Disabled(!d.CanRepair || m.StartupRepairBusy).Clicked() && m.OnRepairStartupSuspects != nil {
				m.OnRepairStartupSuspects()
			}
			if (d.Kind == "runtime" || d.Kind == "missing-node") && ghostSmall(c, releaseText("选择修复版本", "Choose repair release")).Clicked() && m.OnTab != nil {
				m.OnTab(TabVersions)
			}
		})
		if len(d.RepairNames) > 0 {
			ui.Text(c, releaseText("仅处理：", "Targets only: ")+strings.Join(d.RepairNames, ", ")).FontSize(11).TextColor(colTertiary)
		}
		for _, p := range d.Plugins {
			label := p.Name + " · " + releaseText("已禁用", "disabled")
			if p.Enabled {
				label = p.Name + " · " + releaseText("已启用", "enabled")
			}
			if p.Suspect {
				label += " · " + releaseText("可疑", "suspect")
			}
			if p.Restriction != "" {
				label += " · " + p.Restriction
			}
			ui.Text(c, label).FontSize(12).TextColor(colSecondary)
		}
	})
}
