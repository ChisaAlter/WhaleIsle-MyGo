package settings

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"runtime"
	"whaleisle/internal/shortcuts"
	fonttheme "whaleisle/internal/ui/theme"
)

func shortcutsPane(c *ui.Context, e *shortcuts.Editor, fg, muted ui.Color) {
	if e == nil {
		return
	}
	ui.Column(c).Key("native-page-shortcuts").Gap(8).Children(func() {
		ui.Text(c, providerCopy("应用与主页快捷键", "Application and page shortcuts")).FontSize(fonttheme.InterfaceSize(14)).Bold().TextColor(fg)
		ui.Text(c, providerCopy("Mod 为 Windows/Linux 的 Ctrl、macOS 的 Command。会话和面板命令在主页执行；工作区、应用导航、缩放与运行命令在各原生页面生效。输入区域、终端、弹窗保留按键所有权。", "Mod is Ctrl on Windows/Linux and Command on macOS. Session and pane commands run on the main page; workspace, application navigation, zoom and run commands work across native pages. Inputs, terminals and popups retain key ownership.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		ui.Row(c).Gap(8).Children(func() {
			ui.Text(c, fmt.Sprintf(providerCopy("应用缩放：%.0f%%", "Application zoom: %.0f%%"), c.ZoomFactor()*100)).TextColor(muted)
			if ui.Button(c, providerCopy("缩小", "Zoom out")).Clicked() {
				c.SetZoomFactor(c.ZoomFactor() - 0.1)
			}
			if ui.Button(c, providerCopy("实际大小", "Actual size")).Clicked() {
				c.SetZoomFactor(1)
			}
			if ui.Button(c, providerCopy("放大", "Zoom in")).Clicked() {
				c.SetZoomFactor(c.ZoomFactor() + 0.1)
			}
		})
		if runtime.GOOS == "darwin" {
			ui.Text(c, providerCopy("macOS 的 Command+` 保留给窗口切换，终端默认未绑定；可在下方设置其他组合。", "On macOS, Command+` belongs to window switching. Terminal is unbound by default; choose another combination below.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		ui.Text(c, providerCopy("支持 Mod + 字母、数字、标点、Enter 或 F1–F12，可加 Shift/Alt。编辑、文件保存与系统保留按键不可改绑。草稿仅在保存成功后生效。", "Use Mod plus a letter, digit, punctuation, Enter or F1–F12, optionally Shift/Alt. Editing, file-save and system-reserved keys are excluded. Drafts take effect only after a successful save.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		if e.ReadError != "" {
			ui.Text(c, e.ReadError).TextColor(fg)
		}
		if e.Err != "" {
			ui.Text(c, e.Err).TextColor(fg)
		}
		if e.Status != "" {
			ui.Text(c, e.Status).TextColor(muted)
		}
		ui.TextInput(c, &e.Query).Label("shortcut-search").FillWidth().Placeholder(providerCopy("搜索命令或快捷键", "Search commands or shortcuts"))
		issues := e.Issues()
		if len(issues) > 0 {
			ui.Text(c, fmt.Sprintf(providerCopy("%d 条绑定无效或冲突；请修复后保存（可清空搜索查看全部）。", "%d invalid/conflicting bindings; fix them before saving (clear search to show all)."), len(issues))).TextColor(fg)
		}
		rows := e.Search(e.Query)
		if len(rows) == 0 {
			ui.Text(c, providerCopy("没有匹配的快捷键", "No matching shortcuts")).TextColor(muted)
		}
		saved := e.Active()
		for _, command := range rows {
			command := command
			ui.Column(c).Key(command.ID).Gap(4).Padding(6, 0, 6, 0).Children(func() {
				ui.Text(c, providerCopy(command.ZH, command.EN)).Bold().TextColor(fg)
				display := saved[command.ID]
				if display == "" {
					display = providerCopy("未绑定", "Unbound")
				}
				defaultDisplay := e.Default(command.ID)
				if defaultDisplay == "" {
					defaultDisplay = providerCopy("未绑定", "Unbound")
				}
				ui.Text(c, providerCopy("已保存：", "Saved: ")+display+providerCopy("；默认：", "; default: ")+defaultDisplay).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				ui.TextInput(c, e.Draft(command.ID)).Label("shortcut-" + command.ID).FillWidth().Disabled(e.ReadError != "").Placeholder(e.Default(command.ID))
				shortcutRecorder(c, e, command.ID)
				ui.Row(c).Gap(8).Children(func() {
					if ui.Button(c, providerCopy("解绑", "Unbind")).Label("unbind-" + command.ID).Disabled(e.ReadError != "").Clicked() {
						*e.Draft(command.ID) = ""
						e.Status = ""
					}
					if ui.Button(c, providerCopy("使用默认值", "Use default")).Label("default-" + command.ID).Disabled(e.ReadError != "").Clicked() {
						e.ResetDraft(command.ID)
						e.Status = ""
					}
				})
				if issue := issues[command.ID]; issue != "" {
					ui.Text(c, issue).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg)
				}
			})
		}
		ui.Row(c).Gap(8).Children(func() {
			if ui.Button(c, providerCopy("放弃草稿并刷新", "Discard drafts and reload")).Label("reload-shortcuts").Clicked() {
				e.Discard()
			}
			if ui.Button(c, providerCopy("保存快捷键", "Save shortcuts")).Label("save-shortcuts").Disabled(e.ReadError != "" || len(issues) > 0 || e.ResetPending || e.Recording != "").Clicked() {
				e.Save()
			}
			if ui.Button(c, providerCopy("恢复全部默认值…", "Reset all shortcuts…")).Label("reset-shortcuts").Disabled(e.ReadError != "" || e.ResetPending).Clicked() {
				e.RequestReset()
			}
		})
		if e.ResetPending {
			ui.Text(c, providerCopy("确认恢复全部应用和主页绑定并立即保存？将替换自定义绑定与未保存草稿。", "Restore and immediately save all application and page defaults? This replaces custom bindings and unsaved drafts.")).TextColor(fg)
			ui.Row(c).Gap(8).Children(func() {
				if ui.Button(c, providerCopy("确认恢复并保存", "Restore and save")).Label("confirm-reset-shortcuts").Clicked() {
					e.ConfirmReset()
				}
				if ui.Button(c, providerCopy("取消", "Cancel")).Label("cancel-reset-shortcuts").Clicked() {
					e.CancelReset()
				}
			})
		}
	})
}
