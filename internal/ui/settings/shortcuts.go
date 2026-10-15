package settings

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"runtime"
	"whaleisle/internal/shortcuts"
)

func shortcutsPane(c *ui.Context, e *shortcuts.Editor, fg, muted ui.Color) {
	if e == nil {
		return
	}
	ui.Column(c).Key("native-page-shortcuts").Gap(8).Children(func() {
		ui.Text(c, providerCopy("主页快捷键", "Main-page shortcuts")).FontSize(14).Bold().TextColor(fg)
		ui.Text(c, providerCopy("7 条绑定，6 类操作。Mod 为 Windows/Linux 的 Ctrl、macOS 的 Command。仅在主页非输入区域生效。", "Seven bindings, six actions. Mod is Ctrl on Windows/Linux and Command on macOS. Active only outside input on the main page.")).FontSize(11).TextColor(muted)
		if runtime.GOOS == "darwin" {
			ui.Text(c, providerCopy("macOS 的 Command+` 保留给窗口切换，终端默认未绑定；可在下方设置其他组合。", "On macOS, Command+` belongs to window switching. Terminal is unbound by default; choose another combination below.")).FontSize(11).TextColor(muted)
		}
		ui.Text(c, providerCopy("支持 Mod + 字母、数字或标点，可加 Shift/Alt。编辑、文件保存与系统保留按键不可改绑。草稿仅在保存成功后生效。", "Use Mod plus a letter, digit or punctuation, optionally Shift/Alt. Editing, file-save and system-reserved keys are excluded. Drafts take effect only after a successful save.")).FontSize(11).TextColor(muted)
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
				defaultDisplay := command.Default
				if defaultDisplay == "" {
					defaultDisplay = providerCopy("未绑定", "Unbound")
				}
				ui.Text(c, providerCopy("已保存：", "Saved: ")+display+providerCopy("；默认：", "; default: ")+defaultDisplay).FontSize(11).TextColor(muted)
				ui.TextInput(c, e.Draft(command.ID)).Label("shortcut-" + command.ID).FillWidth().Disabled(e.ReadError != "").Placeholder(command.Default)
				ui.Row(c).Gap(8).Children(func() {
					if ui.Button(c, providerCopy("解绑", "Unbind")).Label("unbind-" + command.ID).Disabled(e.ReadError != "").Clicked() {
						*e.Draft(command.ID) = ""
						e.Status = ""
					}
					if ui.Button(c, providerCopy("使用默认值", "Use default")).Label("default-" + command.ID).Disabled(e.ReadError != "").Clicked() {
						*e.Draft(command.ID) = command.Default
						e.Status = ""
					}
				})
				if issue := issues[command.ID]; issue != "" {
					ui.Text(c, issue).FontSize(11).TextColor(fg)
				}
			})
		}
		ui.Row(c).Gap(8).Children(func() {
			if ui.Button(c, providerCopy("保存快捷键", "Save shortcuts")).Label("save-shortcuts").Disabled(e.ReadError != "" || len(issues) > 0 || e.ResetPending).Clicked() {
				e.Save()
			}
			if ui.Button(c, providerCopy("恢复全部默认值…", "Reset all shortcuts…")).Label("reset-shortcuts").Disabled(e.ReadError != "" || e.ResetPending).Clicked() {
				e.RequestReset()
			}
		})
		if e.ResetPending {
			ui.Text(c, providerCopy("确认恢复这 7 条主页绑定并立即保存？将替换自定义绑定与未保存草稿。", "Restore and immediately save all seven main-page defaults? This replaces custom bindings and unsaved drafts.")).TextColor(fg)
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
