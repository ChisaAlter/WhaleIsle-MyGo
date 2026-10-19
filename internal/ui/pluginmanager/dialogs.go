package pluginmanager

import (
	"context"
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	manager "whaleisle/internal/pluginmanager"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

func confirmationDialog(c *ui.Context, m Model, s manager.State, a manager.Actions) {
	open := true
	w, h := c.Size()
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	ui.DialogBase(c.Key("plugin-manager-confirm"), &open, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, .35)).Padding(16)
		panel.Role(ui.RoleAlertDialog).Label(copyText("确认插件操作", "Confirm plugin operation")).Width(max(0, min(float32(560), w-32))).MaxHeight(max(0, h-48)).Background(chrome.Color(colors.BG)).Border(.5, chrome.Color(colors.Border)).Radius(12).Padding(20).Gap(16).Children(func() {
			ui.Text(c, copyText("确认插件操作", "Confirm plugin operation")).FontSize(theme.InterfaceSize(16)).FontWeight(500).TextColor(chrome.Color(colors.FG))
			ui.Scroll(c).FillWidth().MaxHeight(max(0, h-200)).MinHeight(0).Children(func() {
				ui.Text(c, s.Confirmation.Description).FillWidth().Selectable().TextColor(chrome.Color(colors.FG))
			})
			ui.Row(c).FillWidth().Wrap().Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c, copyText("取消", "Cancel")).Clicked() {
					open = false
				}
				if ui.PrimaryButton(c, copyText("确认并执行", "Confirm and run")).Disabled(s.Busy).Clicked() {
					a.Confirm()
				}
			})
		})
	})
	if !open {
		a.CancelConfirmation()
	}
}
func installDialog(c *ui.Context, m Model, s manager.State, a manager.Actions) {
	i := s.Install
	open := true
	w, h := c.Size()
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	pending := i.Phase == "starting" || i.Phase == "running" || i.Phase == "applying" || i.Phase == "cancelling" || i.Phase == "unconfirmed" || i.Phase == "unknown"
	ui.DialogBase(c.Key("plugin-manager-install"), &open, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, .35)).Padding(16)
		panel.Label(copyText("添加插件", "Add plugin")).Width(max(0, min(float32(640), w-32))).Height(max(0, min(float32(720), h-48))).Background(chrome.Color(colors.BG)).Border(.5, chrome.Color(colors.Border)).Radius(12).Clip().Children(func() {
			ui.Row(c).FillWidth().Padding(20, 20, 12, 20).Gap(8).Children(func() {
				ui.Text(c, copyText("添加插件", "Add plugin")).Grow(1).MinWidth(0).FontSize(theme.InterfaceSize(18)).FontWeight(500).TextColor(fg)
				if ui.Button(c, copyText("关闭", "Close")).Clicked() {
					open = false
				}
			})
			ui.Scroll(c).FillWidth().Grow(1).MinHeight(0).Padding(0, 20, 20, 20).Children(func() {
				ui.Column(c).FillWidth().MinWidth(0).Gap(14).Children(func() {
					ui.Text(c, copyText("输入完整的 npm 包名、Git 仓库地址或本地目录。", "Enter a full npm package name, Git repository address, or local directory.")).FillWidth().FontSize(theme.InterfaceSize(13)).TextColor(muted)
					spec := i.Spec
					if ui.TextInput(c, &spec).FillWidth().Label(copyText("插件名或地址", "Plugin name or address")).Placeholder("dsh-plugin-whale-pet").ReadOnly(pending || s.Busy).Changed() {
						a.EditInstall(spec, i.Registry)
					}
					ui.Text(c, copyText("安装源", "Installation registry")).FontSize(theme.InterfaceSize(13)).TextColor(fg)
					registry := i.Registry
					if i.RegistryInfo != nil {
						labels := []string{copyText("配置的默认安装源", "Configured default registry")}
						ids := []string{""}
						if i.RegistryInfo.Registry != nil {
							labels = append(labels, *i.RegistryInfo.Registry)
							ids = append(ids, *i.RegistryInfo.Registry)
						}
						for _, url := range i.RegistryInfo.FallbackRegistries {
							found := false
							for _, existing := range ids {
								if existing == url {
									found = true
								}
							}
							if !found {
								labels = append(labels, url)
								ids = append(ids, url)
							}
						}
						selected := labels[0]
						for n, id := range ids {
							if id == registry {
								selected = labels[n]
							}
						}
						if ui.Select(c, &selected, labels).FillWidth().Disabled(pending || s.Busy).Changed() {
							for n, label := range labels {
								if selected == label {
									a.EditInstall(i.Spec, ids[n])
								}
							}
						}
					}
					if ui.TextInput(c, &registry).FillWidth().Label(copyText("自定义安装源 URL", "Custom registry URL")).Placeholder("https://registry.npmjs.org/").ReadOnly(pending || s.Busy).Changed() {
						a.EditInstall(i.Spec, registry)
					}
					ui.Text(c, copyText("请确认来源可信。插件以当前 profile 的权限运行。", "Confirm the source is trusted. Plugins run with this profile's permissions.")).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
					if i.Phase == "checking" {
						ui.Text(c, copyText("正在检查…", "Checking…")).TextColor(muted)
					}
					if i.Error != "" {
						ui.Text(c, i.Error).FillWidth().TextColor(c.Theme().Danger)
					}
					if i.Inspection != nil && i.Inspection.Status == "accepted" && (i.Inspection.Bundle == nil || *i.Inspection.Bundle) {
						ui.Text(c, strings.TrimSpace(i.Inspection.Name+" "+i.Inspection.Version)).FillWidth().FontSize(theme.InterfaceSize(14)).FontWeight(500).TextColor(fg)
						ui.Text(c, i.Inspection.Description).FillWidth().TextColor(muted)
						if !pending && i.Result == nil && ui.PrimaryButton(c, copyText("安装", "Install")).Disabled(s.Busy).Clicked() {
							a.RequestInstall()
						}
					} else if !pending && ui.PrimaryButton(c, copyText("检查插件", "Check plugin")).Disabled(s.Busy || i.Phase == "checking" || strings.TrimSpace(i.Spec) == "").Clicked() {
						a.Inspect()
					}
					if pending {
						ui.Text(c, copyText("安装阶段：", "Installation phase: ")+i.Phase).Role(ui.RoleStatus).FillWidth().TextColor(muted)
						if i.Phase == "unconfirmed" {
							if ui.Button(c, copyText("核对安装结果", "Reconcile installation")).Clicked() {
								a.ReconcileInstall()
							}
						} else if i.Phase != "unknown" && i.Phase != "applying" && i.Phase != "cancelling" && ui.Button(c, copyText("取消安装", "Cancel installation")).Clicked() {
							a.CancelInstall()
						}
					}
					if i.Result != nil {
						result := i.Result
						ui.Text(c, result.Application).FillWidth().FontSize(theme.InterfaceSize(14)).FontWeight(500).TextColor(fg)
						if result.Application == "restart-required" {
							ui.Text(c, copyText("已保存。重启运行时后生效：", "Saved. Restart the runtime to apply: ")+result.RestartReason).FillWidth().TextColor(muted)
						}
						if result.Error != nil {
							ui.Text(c, result.Error.Code+": "+result.Error.Diagnostic).FillWidth().TextColor(c.Theme().Danger)
							for _, item := range result.Error.Incompatible {
								it := item
								ui.Text(c, it.Name+"@"+it.Version+" · runtime "+it.RuntimeVersion).FillWidth().TextColor(c.Theme().Warning)
								if ui.Button(c, copyText("接受兼容性风险", "Accept compatibility risk")).Disabled(s.Busy).Clicked() {
									a.RequestExemption(it.Name+"@"+it.Version, it.RuntimeVersion)
								}
							}
						}
						if len(result.PendingBuilds) > 0 {
							ui.Text(c, copyText("以下包的构建脚本需要明确授权：", "These packages need explicit build-script approval: ")+strings.Join(result.PendingBuilds, ", ")).FillWidth().TextColor(c.Theme().Warning)
							if ui.Button(c, copyText("批准列出的构建脚本", "Approve listed build scripts")).Disabled(s.Busy).Clicked() {
								a.ApproveBuilds()
							}
						}
						if result.PackageResult != nil {
							ui.Text(c, result.PackageResult.Output).FillWidth().Selectable().FontSize(theme.InterfaceSize(12)).TextColor(muted)
							if result.PackageResult.Truncated {
								ui.Text(c, copyText("输出已截断。日志路径：", "Output truncated. Log path: ")+result.PackageResult.LogPath).FillWidth().TextColor(muted)
							}
							if result.PackageResult.ExitCode != nil {
								ui.Text(c, fmt.Sprintf("Exit: %d", *result.PackageResult.ExitCode)).TextColor(muted)
							}
						}
					}
					for _, run := range i.Runs {
						ui.Column(c).FillWidth().MinWidth(0).Gap(4).Padding(10).Border(.5, chrome.Color(colors.Border)).Radius(6).Children(func() {
							ui.Text(c, run.Command).FillWidth().MinWidth(0).Selectable().FontSize(theme.InterfaceSize(12)).FontWeight(500).TextColor(fg)
							ui.Text(c, run.CWD).FillWidth().MinWidth(0).Selectable().FontSize(theme.InterfaceSize(11)).TextColor(muted)
							for _, chunk := range run.Chunks {
								color := muted
								if chunk.Stream == "stderr" {
									color = c.Theme().Warning
								}
								ui.Text(c, "["+chunk.Stream+"] "+chunk.Text).FillWidth().MinWidth(0).Selectable().FontSize(theme.InterfaceSize(12)).TextColor(color)
							}
							if run.Truncated {
								ui.Text(c, copyText("输出已截断", "Output truncated")).TextColor(muted)
							}
							if run.Settled {
								status := copyText("已结束，未提供退出代码", "Ended without an exit code")
								if run.ExitCode != nil {
									status = fmt.Sprintf("Exit: %d", *run.ExitCode)
								}
								ui.Text(c, status).TextColor(muted)
							}
						})
					}
					if len(i.Runs) == 0 && i.Log != "" {
						ui.Text(c, i.Log).FillWidth().MinWidth(0).Selectable().FontSize(theme.InterfaceSize(12)).TextColor(muted)
					}
					if i.Phase == "unknown" {
						ui.Text(c, copyText("请先刷新清单核对结果，再决定是否重新安装。", "Refresh inventory and reconcile the outcome before deciding whether to install again.")).FillWidth().TextColor(c.Theme().Warning)
						if ui.Button(c, copyText("刷新清单", "Refresh inventory")).Disabled(s.Loading).Clicked() {
							m.Controller.Refresh(context.Background())
						}
					}
					if !pending && i.Result != nil && ui.Button(c, copyText("返回编辑", "Back to editing")).Disabled(s.Busy).Clicked() {
						a.EditInstall(i.Spec, i.Registry)
					}
				})
			})
		})
	})
	if !open {
		a.CloseInstall()
	}
}
