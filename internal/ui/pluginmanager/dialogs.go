package pluginmanager

import (
	"context"
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	manager "whaleisle/internal/pluginmanager"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/pagekit"
)

// pluginDialogWidth is the plugin manager dialogs' maximum width.
const pluginDialogWidth = 560

// dialogScroll holds a dialog body that may outgrow the window: it scrolls
// inside the panel, which keeps its header and footer. reserve is the room
// the dialog's own chrome (header, body padding, footer, panel border and
// the backdrop's inset) takes from the window height.
func dialogScroll(c *ui.Context, reserve float32, fn func()) {
	_, h := c.Size()
	ui.Column(c).FillWidth().MaxHeight(max(design.Sp(30), h-reserve)).Children(func() {
		coss.ScrollArea(c, coss.ScrollAreaOptions{}, func() {
			ui.Column(c).FillWidth().MinWidth(0).Gap(design.Sp(4)).Children(fn)
		})
	})
}

// confirmationText is the confirmation's question in the active locale.
func confirmationText(cf *manager.Confirmation) string {
	switch cf.Kind {
	case "remove":
		return fmt.Sprintf(copyText("卸载插件 %s？", "Remove plugin %s?"), cf.Target)
	case "exemption":
		return fmt.Sprintf(copyText("接受兼容性风险并为 %s 设置豁免？", "Accept the compatibility risk and exempt %s?"), cf.Target)
	}
	return cf.Description
}

func confirmationDialog(c *ui.Context, p *page) {
	s, a := p.s, p.a
	open := true
	cf := s.Confirmation
	confirm := coss.ButtonOptions{Disabled: s.Busy, Loading: s.Busy}
	label := copyText("确认并执行", "Confirm and run")
	if cf.Kind == "remove" {
		confirm.Variant = coss.ButtonDestructive
	}
	coss.AlertDialog(c, &open, coss.DialogOptions{Key: "plugin-manager-confirm", Title: copyText("确认插件操作", "Confirm plugin operation"),
		Description: confirmationText(cf), MaxWidth: pluginDialogWidth, Footer: func() {
			if coss.Button(c, copyText("取消", "Cancel"), coss.ButtonOptions{Variant: coss.ButtonOutline}).Clicked() {
				open = false
			}
			if coss.Button(c, label, confirm).Clicked() {
				a.Confirm()
			}
		}}, nil).Label(copyText("确认插件操作", "Confirm plugin operation"))
	if !open {
		a.CancelConfirmation()
	}
}

func installDialog(c *ui.Context, p *page) {
	s, a, m := p.s, p.a, p.m
	i := s.Install
	open := true
	pending := i.Phase == "starting" || i.Phase == "running" || i.Phase == "applying" || i.Phase == "cancelling" || i.Phase == "unconfirmed" || i.Phase == "unknown"
	_, githubRecovery := manager.GitHubRecoveryRegistry(i)
	title := copyText("添加插件", "Add plugin")
	description := ""
	if githubRecovery {
		title = copyText("无法访问 GitHub", "Cannot access GitHub")
		if i.Result.PackageResult.Kind == "timeout" {
			title = copyText("连接 GitHub 超时", "GitHub connection timed out")
		}
	} else if !i.Confirmed && (i.Phase == "idle" || i.Phase == "checking") && !i.MirrorRecovery {
		description = copyText("输入完整 npm 包名、Git 仓库地址或本地目录。", "Enter a full npm package name, Git repository address, or local directory.")
	}
	var footer func()
	if !githubRecovery && !i.Confirmed && (i.Phase == "idle" || i.Phase == "checking") {
		footer = func() { installFooter(c, i, a, s.Busy) }
	}
	coss.Dialog(c, &open, coss.DialogOptions{Key: "plugin-manager-install", Title: title, Description: description, MaxWidth: pluginDialogWidth,
		FooterVariant: coss.FooterBare, Footer: footer}, func() {
		// The spec screen's footer is the dialog's largest chrome: the warn
		// note and the 40 DIP action in their 16/24 padding, under the
		// title+description header.
		dialogScroll(c, design.Sp(88), func() {
			if githubRecovery {
				pagekit.Muted(c, copyText("请尝试其他安装来源。", "Try another installation source.")).FillWidth()
				var resolved *string
				if i.RegistryInfo != nil {
					resolved = i.RegistryInfo.Resolved
				}
				already := manager.RegistryKey(manager.RegistryValue(i.RegistrySelection), resolved) == manager.NPMMirrorRegistry
				label := copyText("改用国内镜像", "Use mainland China mirror")
				if already {
					label = copyText("试试其他方式", "Try another way")
				}
				if coss.Button(c, label, coss.ButtonOptions{}).AlignSelf(ui.Start).Clicked() {
					a.UseGitHubMirror()
				}
				return
			}
			if i.Confirmed {
				title := i.ConfirmedTitle.Resolve(i18n.Locale())
				if title == "" {
					title = i.Title
				}
				ui.Column(c).FillWidth().Gap(design.Sp(0.5)).Children(func() {
					pagekit.Strong(c, title+" · "+i.Version).FillWidth()
					pagekit.Code(c, i.Spec).FillWidth().Selectable()
				})
			} else if i.Phase == "idle" || i.Phase == "checking" {
				spec := i.Spec
				coss.Field(c, coss.FieldOptions{Label: copyText("包名或地址", "Plugin name or address")}, func() {
					if coss.Input(c, &spec, coss.InputOptions{Placeholder: "dsh-plugin-whale-pet", ReadOnly: s.Busy || i.Phase == "checking"}).AutoFocus().Changed() {
						a.EditInstall(spec, i.Registry)
					}
				})
				_, registryAnchor, guideOpen, _ := installOptionsRow(c, i, a, s.Busy || i.Phase == "checking")
				installGuidePanel(c, i, a, guideOpen)
				installRegistryPanel(c, registryAnchor, i, a)
			}
			if i.Inspection != nil && i.Inspection.Status == "accepted" && i.Phase != "idle" && i.Phase != "checking" {
				ui.Column(c).FillWidth().Gap(design.Sp(0.5)).Children(func() {
					pagekit.Strong(c, strings.TrimSpace(i.Inspection.Name+" "+i.Inspection.Version)).FillWidth()
					pagekit.Muted(c, i.Inspection.Description).FillWidth()
				})
			}
			if i.Error != "" {
				pagekit.FaultSentence(c, copyText("无法安装", "Cannot install"), installErrorText(i), "", nil)
			}
			if i.Phase == "checking" && coss.Button(c, copyText("取消安装并返回编辑", "Cancel check and return to editing"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM}).AlignSelf(ui.Start).Clicked() {
				a.CancelInstall()
			}
			if pending {
				installProgress(c, i, a)
			}
			if result := i.Result; result != nil {
				installResult(c, p, i, result)
			}
			if i.Phase != "idle" && i.Phase != "checking" && !githubRecovery {
				installDetails(c, i, a, pending)
			}
			if i.Phase == "unknown" {
				pagekit.Note(c, coss.AlertWarning, copyText("安装结果未知", "The installation result is unknown"),
					copyText("运行时未保留安装结果。可刷新清单查看当前状态，或返回输入。", "The runtime retained no installation result. Refresh inventory to inspect the current state, or return to input."), nil)
				ui.Row(c).Gap(design.Sp(2)).Children(func() {
					if coss.Button(c, copyText("刷新清单", "Refresh inventory"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Icon: coss.IconRefresh, Disabled: s.Loading}).Clicked() {
						m.Controller.Refresh(context.Background())
					}
					if coss.Button(c, copyText("返回输入", "Back to input"), coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonSM}).Clicked() {
						a.CancelInstall()
					}
				})
			}
			if !pending && i.Result != nil && coss.Button(c, copyText("返回编辑", "Back to editing"), coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonSM, Icon: coss.IconArrowLeft, Disabled: s.Busy}).AlignSelf(ui.Start).Clicked() {
				a.EditInstall(i.Spec, i.Registry)
			}
		})
	}).Label(title)
	if !open {
		a.CloseInstall()
	}
}

// installErrorText is the install failure in the active locale: the
// controller's authored refusals in their own words, raw failures as their
// sentence.
func installErrorText(i manager.InstallState) string {
	raw := i.Error
	switch {
	case strings.HasPrefix(raw, "Plugin is already installed: "):
		return copyText("插件已安装：", "Plugin is already installed: ") + strings.TrimPrefix(raw, "Plugin is already installed: ")
	case strings.HasPrefix(raw, "This plugin is supplied by the runtime: "):
		return copyText("此插件由运行时提供：", "This plugin is supplied by the runtime: ") + strings.TrimPrefix(raw, "This plugin is supplied by the runtime: ")
	case raw == "The package does not declare a plugin bundle.":
		return copyText("此包没有声明插件包。", raw)
	case raw == "Installation interrupted. Confirm the Host result before retrying.":
		return copyText("安装已中断。请先确认 Host 的结果再重试。", raw)
	case raw == "No active request was confirmed; reconcile the result before retrying.":
		return copyText("没有确认到进行中的请求；请先核对结果再重试。", raw)
	case raw == "The Host has no retained installation result.":
		return copyText("Host 没有保留安装结果。", raw)
	case i.Guidance != "" && raw == i.Guidance:
		return raw
	case i.Inspection != nil && i.Inspection.Capabilities != nil && raw == i.Inspection.Capabilities.Guidance:
		return raw
	}
	return errcopyText(raw)
}

// installProgress is the running phase with its cancel and recovery actions.
func installProgress(c *ui.Context, i manager.InstallState, a manager.Actions) {
	ui.Row(c).FillWidth().Gap(design.Sp(2)).AlignItems(ui.Center).Role(ui.RoleStatus).Children(func() {
		if i.Phase != "unconfirmed" && i.Phase != "unknown" {
			coss.Spinner(c, coss.SpinnerOptions{})
		}
		pagekit.Muted(c, copyText("安装阶段：", "Installation phase: ")+i.Phase).Grow(1)
	})
	ui.Row(c).FillWidth().Wrap().Gap(design.Sp(2)).Children(func() {
		if i.Phase == "unconfirmed" {
			if coss.Button(c, copyText("核对安装状态", "Reconcile installation"), coss.ButtonOptions{Size: coss.ButtonSM, Disabled: i.Recovering}).Clicked() {
				a.ReconcileInstall()
			}
			if (!i.CancellationRequested || i.WaitingForStart) && coss.Button(c, copyText("重试取消", "Retry cancellation"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM}).Clicked() {
				a.CancelInstall()
			}
			if i.NativeCanFinish && coss.Button(c, copyText("保留当前结果并结束", "Keep current results and finish"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Disabled: i.Recovering}).Clicked() {
				a.FinishNativeReview()
			}
		} else if i.Phase != "unknown" && i.Phase != "applying" && i.Phase != "cancelling" && coss.Button(c, copyText("取消安装", "Cancel installation"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM}).Clicked() {
			a.CancelInstall()
		}
	})
}

// installResult is the Host's answer: applied, restart required or failed,
// with the follow-up actions it allows.
func installResult(c *ui.Context, p *page, i manager.InstallState, result *manager.Change) {
	s, a := p.s, p.a
	switch {
	case result.Error != nil:
		pagekit.FaultSentence(c, copyText("安装失败", "Installation failed"), errcopySentenceFor(result.Error.Code, result.Error.Diagnostic), "", nil)
	case result.Application == "restart-required":
		pagekit.Note(c, coss.AlertInfo, copyText("已保存，重启运行时后生效", "Saved. Restart the runtime to apply it"), result.RestartReason, nil)
	case result.Application == "applied":
		pagekit.Note(c, coss.AlertSuccess, copyText("已安装", "Installed"), "", nil)
	case result.Application != "":
		pagekit.Strong(c, result.Application).FillWidth()
	}
	for _, warning := range result.Warnings {
		pagekit.Caption(c, warning).FillWidth()
	}
	if i.Phase == "done" && i.Enabled == nil && result.Bundle != "" && coss.Button(c, copyText("启用插件", "Enable plugin"), coss.ButtonOptions{Disabled: i.Enabling, Loading: i.Enabling}).AlignSelf(ui.Start).Clicked() {
		a.EnableInstalled()
	}
	if result.Error != nil {
		for _, item := range result.Error.Incompatible {
			it := item
			pagekit.Note(c, coss.AlertWarning, it.Name+"@"+it.Version, fmt.Sprintf(copyText("要求的运行时：%s", "Requires runtime %s"), it.RuntimeVersion), func() {
				if coss.Button(c, copyText("接受兼容性风险", "Accept compatibility risk"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Disabled: s.Busy}).Clicked() {
					a.RequestExemption(it.Name+"@"+it.Version, it.RuntimeVersion)
				}
			})
		}
	}
	if i.Phase == "failed" && len(result.PendingBuilds) > 0 {
		pagekit.Note(c, coss.AlertWarning, copyText("以下包的构建脚本需要明确授权", "These packages need explicit build-script approval"), strings.Join(result.PendingBuilds, ", "), nil)
		if coss.Button(c, copyText("批准列出的构建脚本并重试", "Approve listed build scripts and retry"), coss.ButtonOptions{Disabled: s.Busy}).AlignSelf(ui.Start).Clicked() {
			a.ApproveBuilds()
		}
	} else if i.Phase == "failed" {
		ui.Row(c).Wrap().Gap(design.Sp(2)).Children(func() {
			if result.FailedAt == "registry" && !i.Confirmed && coss.Button(c, copyText("更换安装源", "Change registry"), coss.ButtonOptions{Variant: coss.ButtonOutline}).Clicked() {
				a.ChangeRegistry()
			}
			unsafeRetry := result.Error != nil && (result.Error.Code == "native-install/partial" || result.Error.Code == "native-install/unconfirmed")
			if !unsafeRetry && coss.Button(c, copyText("重试安装", "Retry installation"), coss.ButtonOptions{Icon: coss.IconRefresh, Disabled: s.Busy}).Clicked() {
				a.RetryInstall()
			}
		})
	}
}

// installDetails is the disclosure holding each attempt's terminal.
func installDetails(c *ui.Context, i manager.InstallState, a manager.Actions, pending bool) {
	label := copyText("显示安装详情", "Show installation details")
	icon := coss.IconChevronDown
	if i.DetailsOpen {
		label, icon = copyText("收起安装详情", "Hide installation details"), coss.IconChevronUp
	}
	if coss.Button(c, label, coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonSM, TrailingIcon: icon}).AlignSelf(ui.Start).Margin(0, 0, 0, -design.Sp(2)).Clicked() {
		a.ToggleInstallDetails()
	}
	if !i.DetailsOpen {
		return
	}
	for n, run := range i.Runs {
		if n < len(i.Attempts) {
			pagekit.Caption(c, fmt.Sprintf(copyText("第 %d 次尝试：%s", "Attempt %d: %s"), n+1, registryName(i.Attempts[n].Registry, nil))).FillWidth()
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
			pagekit.Muted(c, copyText("暂无输出", "No output"))
		}
	}
	if i.Result != nil && i.Result.PackageResult != nil && i.Result.PackageResult.Truncated {
		pagekit.Code(c, copyText("输出已截断。日志路径：", "Output truncated. Log path: ")+i.Result.PackageResult.LogPath).FillWidth().Selectable()
	}
}

func sourcesDialog(c *ui.Context, p *page) {
	open := true
	coss.Dialog(c, &open, coss.DialogOptions{Key: "plugin-index-sources", Title: copyText("插件索引源", "Plugin index sources"), MaxWidth: pluginDialogWidth}, func() {
		dialogScroll(c, design.Sp(36), func() { sourcesEditor(c, p) })
	}).Label(copyText("插件索引源", "Plugin index sources"))
	if !open {
		p.a.ToggleSources()
	}
}

func marketChoiceDialog(c *ui.Context, p *page) {
	a := p.a
	choice := p.s.Discover.Choice
	if choice == nil {
		return
	}
	open := true
	coss.Dialog(c, &open, coss.DialogOptions{Key: "plugin-market-choice", Title: copyText("确认安装插件", "Confirm plugin installation"), FooterVariant: coss.FooterBare,
		Description: choice.Entry.Label(i18n.Locale()) + " · " + choice.Version.Version, MaxWidth: pluginDialogWidth, Footer: func() {
			if coss.Button(c, copyText("取消", "Cancel"), coss.ButtonOptions{Variant: coss.ButtonOutline}).Clicked() {
				open = false
			}
			if coss.Button(c, copyText("安装", "Install"), coss.ButtonOptions{Icon: coss.IconDownload}).Clicked() {
				a.ConfirmMarketChoice()
			}
		}}, func() {
		dialogScroll(c, design.Sp(52), func() {
			if choice.Entry.Author != "" {
				pagekit.Muted(c, choice.Entry.Author).FillWidth()
			}
			if facts := manager.SourceFacts(choice.Entry.Name, choice.Version); len(facts) > 0 {
				rows(c, func() {
					for n, fact := range facts {
						if n > 0 {
							rowSeparator(c)
						}
						rowFrame(ui.Row(c)).Children(func() {
							pagekit.Caption(c, fact.Kind).Width(design.Sp(24)).Shrink(0)
							pagekit.Code(c, fact.Value).Grow(1).MinWidth(0).Selectable()
						})
					}
				})
			}
			if !choice.Version.Compatibility.Compatible {
				pagekit.Note(c, coss.AlertWarning, copyText("此版本具有兼容性风险豁免。", "This version has a compatibility risk exemption."), "", nil)
			}
			installSafetyNote(c)
		})
	}).Label(copyText("确认安装插件", "Confirm plugin installation"))
	if !open {
		a.CancelMarketChoice()
	}
}
