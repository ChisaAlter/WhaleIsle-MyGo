package frame

import (
	"context"
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func archiveText(zh, en string) string {
	if i18n.Locale() == i18n.LocaleZH {
		return zh
	}
	return en
}

func archiveConfirmation(c *ui.Context, m Model) {
	archiveUndoToast(c, m)
	request := m.Store.ArchiveConfirmation()
	if request == nil {
		return
	}
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	open := true
	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		chrome.DialogSurface(c, backdrop, panel, colors, 480, 24).Children(func() {
			ui.Column(c).Gap(12).Children(func() {
				ui.Text(c, archiveText("停止并归档此会话？", "Stop and archive this session?")).FontSize(theme.InterfaceSize(16)).Bold().TextColor(chrome.Color(colors.FG))
				ui.Text(c, fmt.Sprintf(archiveText("“%s”仍有正在进行的工作。归档会先停止这些工作；之后可在侧栏筛选“全部对话（显示已归档）”中恢复会话，被停止的工作不会自动继续。", "“%s” still has work in progress. Archiving first stops that work; the session can later be restored from the “All conversations (show archived)” filter in the sidebar, but stopped work does not resume automatically."), request.Title)).FontSize(theme.InterfaceSize(12)).TextColor(chrome.Color(colors.Muted))
				ui.Text(c, archiveText("将被停止的工作", "Work that will be stopped")).FontSize(theme.InterfaceSize(12)).TextColor(chrome.Color(colors.FG))
				for _, activity := range request.Activity {
					ui.Text(c, archiveActivityLine(activity)).FontSize(theme.InterfaceSize(12)).TextColor(chrome.Color(colors.FG))
				}
				if request.Error != "" {
					ui.Text(c, request.Error).FontSize(theme.InterfaceSize(12)).TextColor(chrome.Color(colors.FG))
				}
				ui.Row(c).Gap(8).Children(func() {
					if ui.Button(c, archiveText("取消", "Cancel")).Disabled(request.Busy).Clicked() {
						m.Store.CancelArchive(request.Token)
					}
					confirmLabel := archiveText("停止并归档", "Stop and archive")
					if request.Busy {
						confirmLabel = archiveText("正在停止并归档…", "Stopping and archiving…")
					}
					if ui.Button(c, confirmLabel).Disabled(request.Busy).Clicked() {
						m.Store.ConfirmArchive(context.Background(), request.Token)
					}
				})
			})
		})
	})
	if !open && !request.Busy {
		m.Store.CancelArchive(request.Token)
	}
}

func archiveActivityLine(activity session.ArchiveActivity) string {
	names := make([]string, 0, len(activity.Items))
	for _, item := range activity.Items {
		name := item.Label
		if name == "" {
			name = item.ID
		}
		names = append(names, name)
	}
	kind := activity.Kind
	switch kind {
	case "turn":
		return archiveText("进行中的回合", "Turn in progress")
	case "subagent":
		return fmt.Sprintf(archiveText("%d 个运行中的子智能体：%s", "%d running subagents: %s"), len(names), strings.Join(names, ", "))
	case "job":
		return fmt.Sprintf(archiveText("%d 个后台任务：%s", "%d background jobs: %s"), len(names), strings.Join(names, ", "))
	case "schedule":
		return fmt.Sprintf(archiveText("%d 条定时提醒：%s", "%d scheduled reminders: %s"), len(names), strings.Join(names, ", "))
	}
	return fmt.Sprintf(archiveText("%d 项其他工作（%s）", "%d other work items (%s)"), len(names), kind)
}
