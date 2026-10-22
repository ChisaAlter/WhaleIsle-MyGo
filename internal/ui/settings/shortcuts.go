package settings

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"runtime"
	"whaleisle/internal/shortcuts"
	"whaleisle/internal/ui/chrome"
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
		if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
			ui.Text(c, providerCopy("按物理位置录制一至两个同时按住的键，释放后保存当前命令。可用 Mod、Control、Alt、Shift、Meta；双键文本写作 K & J。其它命令草稿保持未保存。", "Record one or two simultaneously held physical keys; release to save that command. Use Mod, Control, Alt, Shift or Meta; write paired keys as K & J. Other command drafts remain unsaved.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		if e.ReadError != "" {
			ui.Text(c, e.ReadError).TextColor(fg)
		}
		if e.Err != "" {
			ui.Text(c, e.Err).TextColor(fg)
			if ui.Button(c, providerCopy("审阅当前配置并保留草稿", "Review current settings and keep drafts")).Clicked() {
				e.Review()
			}
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
						e.SaveRow(command.ID, false)
					}
					if ui.Button(c, providerCopy("使用默认值", "Use default")).Label("default-" + command.ID).Disabled(e.ReadError != "").Clicked() {
						e.ResetDraft(command.ID)
						e.SaveRow(command.ID, true)
					}
				})
				if issue := issues[command.ID]; issue != "" {
					ui.Text(c, issue).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg)
				}
			})
		}
		ui.Row(c).Gap(8).Children(func() {
			if ui.Button(c, providerCopy("放弃草稿并刷新", "Discard drafts and reload")).Label("reload-shortcuts").Clicked() {
				e.Reload()
			}
			if ui.Button(c, providerCopy("保存快捷键", "Save shortcuts")).Label("save-shortcuts").Disabled(e.ReadError != "" || len(issues) > 0 || e.ResetPending || e.Recording != "").Clicked() {
				e.Save()
			}
			if ui.Button(c, providerCopy("恢复全部默认值…", "Reset all shortcuts…")).Label("reset-shortcuts").Disabled(e.ReadError != "" || e.ResetPending).Clicked() {
				e.RequestReset()
			}
		})
		if e.ResetPending {
			open := true
			ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
				width, _ := c.Size()
				chrome.DialogSurfaceTheme(c, backdrop, panel, c.Theme(), min(float32(480), max(float32(0), width-32)), 20).Gap(12).Children(func() {
					ui.Text(c, providerCopy("确认恢复全部应用和主页绑定并立即保存？将替换已知命令的自定义绑定与未保存草稿，未知命令保留。", "Restore and save all application and page defaults? This replaces known commands' bindings and drafts; unknown commands remain.")).TextColor(fg)
					ui.Row(c).Gap(8).Children(func() {
						if ui.Button(c, providerCopy("确认恢复并保存", "Restore and save")).Label("confirm-reset-shortcuts").Clicked() {
							e.ConfirmReset()
						}
						if ui.Button(c, providerCopy("取消", "Cancel")).Label("cancel-reset-shortcuts").Clicked() {
							e.CancelReset()
						}
					})
				})
			})
			if !open {
				e.CancelReset()
			}
		}
	})
}
