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
	request := m.Store.Snapshot().ArchiveConfirmation
	if request == nil {
		return
	}
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	open := true
	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, 0.4))
		panel.Width(480).MaxWidthPercent(92).Padding(20).Background(chrome.Color(colors.BG)).Radius(14).Children(func() {
			ui.Column(c).Gap(12).Children(func() {
				ui.Text(c, archiveText("停止工作并归档会话", "Stop work and archive session")).FontSize(theme.InterfaceSize(16)).Bold().TextColor(chrome.Color(colors.FG))
				ui.Text(c, request.Title).FontSize(theme.InterfaceSize(13)).TextColor(chrome.Color(colors.FG))
				ui.Text(c, archiveText("Host 报告以下工作仍在运行。确认后会请求停止这些工作，再归档会话。", "The Host reports this work is still active. Confirming requests that it stop before archiving the session.")).FontSize(theme.InterfaceSize(12)).TextColor(chrome.Color(colors.Muted))
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
					if ui.Button(c, archiveText("停止工作并归档", "Stop work and archive")).Disabled(request.Busy).Clicked() {
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
		return archiveText("• 正在运行的对话轮次", "• Running turn")
	case "subagent":
		kind = archiveText("子代理", "Subagents")
	case "job":
		kind = archiveText("后台任务", "Background jobs")
	case "schedule":
		kind = archiveText("计划任务", "Active schedules")
	}
	return fmt.Sprintf("• %s (%d): %s", kind, len(names), strings.Join(names, ", "))
}
