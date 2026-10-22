package pluginmanager

import (
	"context"
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	manager "whaleisle/internal/pluginmanager"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

// The plugin manager's cards follow the reference Modal: a 560 DIP card on the
// shared surface, 24 DIP of air around it, a 22/14/12/24 header holding a
// 16/500 title and a 28 DIP close box, and 24 DIP body padding.
const (
	pluginDialogWidth        = 560
	pluginDialogHeaderHeight = 58
)

// pluginDialogCloseIcon is the reference Modal's 14 DIP cross.
var pluginDialogCloseIcon = ui.MustParseSVG([]byte(`<svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1" aria-hidden="true"><path d="m4 4 8 8M12 4l-8 8"/></svg>`))

// pluginDialog draws one plugin manager card: the shared surface, the reference
// header, the body and, when the card has one, its footer. A zero height sizes
// the card to its content, capped at the window's 24 DIP inset; a positive
// height fixes it. A nil footer leaves the card as the header and body alone;
// otherwise the footer is the card's last row and never scrolls, as the
// reference's `footer` prop is (Modal.tsx:88-96). It returns the panel so the
// caller can add overlays of its own.
func pluginDialog(c *ui.Context, backdrop, panel ui.Element, colors theme.Colors, title, closeLabel string, height float32, close func(), body, footer func()) ui.Element {
	w, h := c.Size()
	box := chrome.DialogSurfaceBox{
		Width:   min(float32(pluginDialogWidth), max(float32(280), w-48)),
		Height:  height,
		Padding: 0,
	}
	panel = chrome.DialogSurfaceSized(c, backdrop, panel, c.Theme(), box)
	if height <= 0 {
		panel.MaxHeight(max(0, h-48))
	}
	panel.Label(title).Gap(0).Children(func() {
		ui.Row(c).FillWidth().Height(pluginDialogHeaderHeight).Shrink(0).Padding(22, 14, 12, 24).AlignItems(ui.Start).Children(func() {
			ui.Text(c, title).Grow(1).MinWidth(0).FontSize(theme.InterfaceSize(16)).FontWeight(500).TextColor(chrome.Color(colors.FG)).SingleLine()
			closeBox := ui.Button(c, "").Label(closeLabel).Size(28, 28).Shrink(0).Padding(7).Radius(8).
				BorderWidth(0).Background(ui.Transparent).Tooltip(closeLabel)
			closeBox.Children(func() {
				ui.Icon(c, pluginDialogCloseIcon).Size(14, 14).TextColor(chrome.Color(colors.Muted))
			})
			if closeBox.Hovered() {
				closeBox.Background(c.Theme().SurfaceHover)
			}
			if closeBox.Clicked() {
				close()
			}
		})
		body()
		if footer != nil {
			footer()
		}
	})
	return panel
}

func confirmationDialog(c *ui.Context, m Model, s manager.State, a manager.Actions) {
	open := true
	w, h := c.Size()
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	ui.DialogBase(c.Key("plugin-manager-confirm"), &open, func(backdrop, panel ui.Element) {
		chrome.DialogSurface(c, backdrop, panel, colors, max(0, min(float32(560), w-32)), 20).
			Role(ui.RoleAlertDialog).Label(copyText("确认插件操作", "Confirm plugin operation")).MaxHeight(max(0, h-48)).Gap(16).Children(func() {
			ui.Text(c, copyText("确认插件操作", "Confirm plugin operation")).FontSize(theme.InterfaceSize(16)).FontWeight(500).TextColor(chrome.Color(colors.FG))
			ui.Scroll(c).FillWidth().MaxHeight(max(0, h-200)).MinHeight(0).Children(func() {
				ui.Text(c, s.Confirmation.Description).FillWidth().Selectable().TextColor(chrome.Color(colors.FG))
			})
			ui.Row(c).FillWidth().Wrap().Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c, copyText("取消", "Cancel")).Clicked() {
					open = false
				}
				if widgets.Button(c, colors, copyText("确认并执行", "Confirm and run"), widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonMedium, Disabled: s.Busy}).Clicked() {
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
	_, h := c.Size()
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	pending := i.Phase == "starting" || i.Phase == "running" || i.Phase == "applying" || i.Phase == "cancelling" || i.Phase == "unconfirmed" || i.Phase == "unknown"
	_, githubRecovery := manager.GitHubRecoveryRegistry(i)
	title := copyText("添加插件", "Add plugin")
	if githubRecovery {
		title = copyText("无法访问 GitHub", "Cannot access GitHub")
		if i.Result.PackageResult.Kind == "timeout" {
			title = copyText("连接 GitHub 超时", "GitHub connection timed out")
		}
	}
	closeLabel := copyText("关闭", "Close")
	if i.Phase == "starting" || i.Phase == "running" || i.Phase == "unconfirmed" {
		closeLabel = copyText("取消安装并关闭", "Cancel install and close")
	}
	// The card is as tall as its content, as the reference Modal is, and the
	// body scrolls inside it once the log outgrows the window's inset. The
	// spec screen's trust note and primary action sit in the card's footer
	// instead, outside the scrolling content.
	var footer func()
	if !githubRecovery && !i.Confirmed && (i.Phase == "idle" || i.Phase == "checking") {
		footer = func() { installFooter(c, i, a, colors, s.Busy) }
	}
	bodyBottom := float32(24)
	if footer != nil {
		bodyBottom = 0
	}
	height := float32(0)
	ui.DialogBase(c.Key("plugin-manager-install"), &open, func(backdrop, panel ui.Element) {
		pluginDialog(c, backdrop, panel, colors, title, closeLabel, height, func() { open = false }, func() {
			ui.Scroll(c).FillWidth().MinHeight(0).MaxHeight(max(0, h-150)).Padding(20, 24, bodyBottom, 24).Children(func() {
				ui.Column(c).FillWidth().MinWidth(0).Gap(14).Children(func() {
					if githubRecovery {
						ui.Text(c, copyText("请尝试其他安装来源。", "Try another installation source.")).FillWidth().TextColor(muted)
						var resolved *string
						if i.RegistryInfo != nil {
							resolved = i.RegistryInfo.Resolved
						}
						already := manager.RegistryKey(manager.RegistryValue(i.RegistrySelection), resolved) == manager.NPMMirrorRegistry
						label := copyText("改用国内镜像", "Use mainland China mirror")
						if already {
							label = copyText("试试其他方式", "Try another way")
						}
						if ui.PrimaryButton(c, label).Clicked() {
							a.UseGitHubMirror()
						}
						return
					}
					if i.Confirmed {
						title := i.ConfirmedTitle.Resolve(i18n.Locale())
						if title == "" {
							title = i.Title
						}
						ui.Text(c, title+" · "+i.Version).FillWidth().TextColor(fg)
						ui.Text(c, i.Spec).FillWidth().Selectable().TextColor(muted)
					} else if i.Phase == "idle" || i.Phase == "checking" {
						if !i.MirrorRecovery {
							ui.Text(c, copyText("输入完整 npm 包名、Git 仓库地址或本地目录。", "Enter a full npm package name, Git repository address, or local directory.")).FillWidth().FontSize(theme.InterfaceSize(13)).TextColor(muted)
						}
						spec := i.Spec
						if ui.TextInput(c, &spec).FillWidth().Label(copyText("插件名称或地址", "Plugin name or address")).Placeholder("dsh-plugin-whale-pet").ReadOnly(s.Busy || i.Phase == "checking").Changed() {
							a.EditInstall(spec, i.Registry)
						}
						_, registryAnchor, guideOpen, _ := installOptionsRow(c, i, a, s.Busy || i.Phase == "checking")
						installGuidePanel(c, i, a, guideOpen, colors)
						installRegistryPanel(c, registryAnchor, i, a)
					}
					if i.Inspection != nil && i.Inspection.Status == "accepted" && i.Phase != "idle" && i.Phase != "checking" {
						ui.Text(c, strings.TrimSpace(i.Inspection.Name+" "+i.Inspection.Version)).FillWidth().FontSize(theme.InterfaceSize(14)).FontWeight(500).TextColor(fg)
						ui.Text(c, i.Inspection.Description).FillWidth().TextColor(muted)
					}
					if i.Error != "" {
						ui.Text(c, i.Error).FillWidth().Role(ui.RoleStatus).TextColor(c.Theme().Danger)
					}
					// The footer owns the checking state's primary action; the
					// body keeps the way back to editing while a check runs.
					if i.Phase == "checking" && ui.Button(c, copyText("取消检查并返回编辑", "Cancel check and return to editing")).Clicked() {
						a.CancelInstall()
					}
					if pending {
						ui.Text(c, copyText("安装阶段：", "Installation phase: ")+i.Phase).Role(ui.RoleStatus).FillWidth().TextColor(muted)
						if i.Phase == "unconfirmed" {
							if ui.Button(c, copyText("核对安装结果", "Reconcile installation")).Disabled(i.Recovering).Clicked() {
								a.ReconcileInstall()
							}
							if (!i.CancellationRequested || i.WaitingForStart) && ui.Button(c, copyText("重试取消", "Retry cancellation")).Clicked() {
								a.CancelInstall()
							}
						} else if i.Phase != "unknown" && i.Phase != "applying" && i.Phase != "cancelling" && ui.Button(c, copyText("取消安装", "Cancel installation")).Clicked() {
							a.CancelInstall()
						}
					}
					if result := i.Result; result != nil {
						ui.Text(c, result.Application).FillWidth().FontSize(theme.InterfaceSize(14)).FontWeight(500).TextColor(fg)
						if result.Application == "restart-required" {
							ui.Text(c, copyText("已保存。重启运行时以生效：", "Saved. Restart the runtime to apply: ")+result.RestartReason).FillWidth().TextColor(muted)
						}
						if i.Phase == "done" && i.Enabled == nil && result.Bundle != "" && ui.PrimaryButton(c, copyText("启用插件", "Enable plugin")).Disabled(i.Enabling).Clicked() {
							a.EnableInstalled()
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
						if i.Phase == "failed" && len(result.PendingBuilds) > 0 {
							ui.Text(c, copyText("以下包的构建脚本需要明确授权：", "These packages need explicit build-script approval: ")+strings.Join(result.PendingBuilds, ", ")).FillWidth().TextColor(c.Theme().Warning)
							if ui.Button(c, copyText("批准列出的构建脚本并重试", "Approve listed build scripts and retry")).Disabled(s.Busy).Clicked() {
								a.ApproveBuilds()
							}
						} else if i.Phase == "failed" {
							if result.FailedAt == "registry" && !i.Confirmed && ui.Button(c, copyText("更换安装源", "Change registry")).Clicked() {
								a.ChangeRegistry()
							}
							if widgets.Button(c, colors, copyText("重试安装", "Retry installation"), widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonMedium, Disabled: s.Busy}).Clicked() {
								a.RetryInstall()
							}
						}
					}
					if i.Phase != "idle" && i.Phase != "checking" && !githubRecovery {
						label := copyText("显示安装详情", "Show installation details")
						if i.DetailsOpen {
							label = copyText("收起安装详情", "Hide installation details")
						}
						if ui.Button(c, label).Clicked() {
							a.ToggleInstallDetails()
						}
						if i.DetailsOpen {
							for n, run := range i.Runs {
								if n < len(i.Attempts) {
									ui.Text(c, fmt.Sprintf(copyText("第 %d 次尝试：%s", "Attempt %d: %s"), n+1, registryName(i.Attempts[n].Registry, nil))).FillWidth().TextColor(muted)
								}
								installTerminal(c.Key("install-run-"+run.JobID), run)
							}
							if len(i.Runs) == 0 {
								output := i.Log
								if i.Result != nil && i.Result.PackageResult != nil {
									output = i.Result.PackageResult.Output
								}
								if output != "" {
									installTerminal(c.Key("install-result-output"), manager.InstallRun{Output: output, Settled: !pending})
								} else {
									ui.Text(c, copyText("暂无输出", "No output")).TextColor(muted)
								}
							}
							if i.Result != nil && i.Result.PackageResult != nil && i.Result.PackageResult.Truncated {
								ui.Text(c, copyText("输出已截断。日志路径：", "Output truncated. Log path: ")+i.Result.PackageResult.LogPath).FillWidth().Selectable().TextColor(muted)
							}
						}
					}
					if i.Phase == "unknown" {
						ui.Text(c, copyText("运行时未保留安装结果。可刷新清单查看当前状态，或返回输入。", "The runtime retained no installation result. Refresh inventory to inspect the current state, or return to input.")).FillWidth().TextColor(c.Theme().Warning)
						if ui.Button(c, copyText("刷新清单", "Refresh inventory")).Disabled(s.Loading).Clicked() {
							m.Controller.Refresh(context.Background())
						}
						if ui.Button(c, copyText("返回输入", "Back to input")).Clicked() {
							a.CancelInstall()
						}
					}
					if !pending && i.Result != nil && ui.Button(c, copyText("返回编辑", "Back to editing")).Disabled(s.Busy).Clicked() {
						a.EditInstall(i.Spec, i.Registry)
					}
				})
			})
		}, footer)
	})
	if !open {
		a.CloseInstall()
	}
}
func sourcesDialog(c *ui.Context, m Model, s manager.State, a manager.Actions) {
	open := true
	_, h := c.Size()
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	ui.DialogBase(c.Key("plugin-index-sources"), &open, func(backdrop, panel ui.Element) {
		dismissed := ui.Local(panel, "source-error-dismissed", func() uint64 { return 0 })
		pluginDialog(c, backdrop, panel, colors, copyText("插件索引源", "Plugin index sources"), copyText("关闭", "Close"), 0, func() { open = false }, func() {
			ui.Scroll(c).FillWidth().MaxHeight(max(0, h-150)).MinHeight(0).Padding(20, 24, 24, 24).Children(func() {
				sourcesEditor(c, s, a, chrome.Color(colors.FG), chrome.Color(colors.Muted))
			})
			if s.Discover.SourceError != "" && *dismissed != s.Discover.SourceErrorSequence {
				ui.Overlay(c.Key("plugin-source-error-toast"), func() {
					ui.Row(c).Absolute().Bottom(24).Left(24).Right(24).MaxWidth(560).Padding(12).Gap(10).Role(ui.RoleStatus).Background(chrome.Color(colors.BG)).Border(.5, chrome.Color(colors.Border)).Radius(8).Children(func() {
						ui.Text(c, s.Discover.SourceError).Grow(1).MinWidth(0).TextColor(c.Theme().Danger)
						if ui.Button(c, copyText("关闭", "Close")).Clicked() {
							*dismissed = s.Discover.SourceErrorSequence
						}
					})
				})
			}
		}, nil)
	})
	if !open {
		a.ToggleSources()
	}
}
func marketChoiceDialog(c *ui.Context, m Model, s manager.State, a manager.Actions) {
	choice := s.Discover.Choice
	if choice == nil {
		return
	}
	open := true
	_, h := c.Size()
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	ui.DialogBase(c.Key("plugin-market-choice"), &open, func(backdrop, panel ui.Element) {
		pluginDialog(c, backdrop, panel, colors, copyText("确认安装插件", "Confirm plugin installation"), copyText("关闭", "Close"), 0, func() { open = false }, func() {
			ui.Column(c).FillWidth().MinWidth(0).Gap(16).Padding(20, 24, 24, 24).Children(func() {
				ui.Text(c, choice.Entry.Label(i18n.Locale())+" · "+choice.Version.Version).FillWidth().FontSize(theme.InterfaceSize(16)).FontWeight(500).TextColor(fg)
				ui.Scroll(c).FillWidth().MaxHeight(max(0, h-260)).MinHeight(0).Children(func() {
					ui.Text(c, choice.Entry.Author).FillWidth().TextColor(muted)
					for _, fact := range manager.SourceFacts(choice.Entry.Name, choice.Version) {
						ui.Text(c, fact.Kind+": "+fact.Value).FillWidth().Selectable().TextColor(fg)
					}
					if !choice.Version.Compatibility.Compatible {
						ui.Text(c, copyText("此版本具有兼容性风险豁免。", "This version has a compatibility risk exemption.")).FillWidth().TextColor(c.Theme().Warning)
					}
					// This card has no reference counterpart (the market is
					// native), so its trust line borrows the spec screen's
					// `.installSafety` panel rather than a second, weaker
					// sentence: both install cards then warn in one voice.
					installSafetyNote(c)
				})
				ui.Row(c).FillWidth().Wrap().Justify(ui.End).Gap(8).Children(func() {
					if ui.Button(c, copyText("取消", "Cancel")).Clicked() {
						open = false
					}
					if ui.PrimaryButton(c, copyText("安装", "Install")).Clicked() {
						a.ConfirmMarketChoice()
					}
				})
			})
		}, nil)
	})
	if !open {
		a.CancelMarketChoice()
	}
}
