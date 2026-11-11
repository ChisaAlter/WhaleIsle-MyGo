package assistant

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/proactive"
	"whaleisle/internal/ui/theme"
)

func proactiveTime(value string) string {
	if at, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return at.Local().Format("01-02 15:04")
	}
	return value
}

func proactiveSources(c *ui.Context, p ProactiveModel, fg, muted, border, accent ui.Color) {
	if p.OnSourceAdd == nil && len(p.State.Sources) == 0 {
		return
	}
	group := ui.Column(c.Key("proactive-sources")).FillWidth().MinWidth(0).Gap(8)
	open := ui.Local(group, "open", func() bool { return false })
	group.Children(func() {
		ui.Row(c).Gap(8).Wrap().Children(func() {
			label := whaleTaskCopy("GitHub 来源", "GitHub sources")
			if len(p.State.Sources) > 0 {
				label += fmt.Sprintf(" (%d)", len(p.State.Sources))
			}
			if ui.Button(c, label).Clicked() {
				*open = !*open
			}
			for _, source := range p.State.Sources {
				if source.Error != "" {
					ui.Text(c, whaleTaskCopy("来源检查失败", "Source check failed")).FontSize(theme.InterfaceSize(12)).TextColor(accent)
					break
				}
			}
		})
		if !*open {
			return
		}
		ui.Text(c, whaleTaskCopy("每 15 分钟检查选定仓库的议题和 PR。新事项进入收件箱，选择执行时才会调用模型。", "Checks issues and PRs in selected repositories every 15 minutes. New items enter the inbox; model calls start only when you run a task.")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		for _, source := range p.State.Sources {
			ui.Column(c.Key(source.ID)).FillWidth().MinWidth(0).Gap(6).Padding(8, 0).Children(func() {
				ui.Text(c, source.Repository).FontSize(theme.InterfaceSize(14)).FontWeight(600).TextColor(fg)
				status := whaleTaskCopy("已暂停", "Paused")
				if source.Enabled {
					status = whaleTaskCopy("已启用", "Enabled")
				}
				if source.LastSuccess != "" {
					status += whaleTaskCopy(" · 上次成功 ", " · Last success ") + proactiveTime(source.LastSuccess)
				}
				ui.Text(c, status).FontSize(theme.InterfaceSize(12)).TextColor(muted)
				if source.Error != "" {
					ui.Text(c, source.Error).TextColor(accent).Selectable()
				}
				ui.Row(c).Gap(8).Wrap().Children(func() {
					label := whaleTaskCopy("暂停来源", "Pause source")
					if !source.Enabled {
						label = whaleTaskCopy("恢复来源", "Resume source")
					}
					if ui.Button(c, label).Disabled(p.SourceBusy || p.OnSourceToggle == nil).Clicked() {
						p.OnSourceToggle(source.Repository, !source.Enabled)
					}
					if source.Enabled && ui.Button(c, whaleTaskCopy("立即检查", "Check now")).Disabled(p.SourceBusy || p.OnSourceRefresh == nil).Clicked() {
						p.OnSourceRefresh(source.ID)
					}
				})
				ui.Box(c).FillWidth().Height(.5).Background(border)
			})
		}
		ui.Text(c, whaleTaskCopy("添加仓库（owner/name）", "Add repository (owner/name)")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		repository := p.SourceRepository
		ui.TextInput(c.Key("proactive-source-repo"), &repository).Label(whaleTaskCopy("GitHub 仓库", "GitHub repository")).FillWidth().Height(36).Radius(8).Border(.5, border).Disabled(p.SourceBusy).OnChange(func() {
			if p.OnSourceEdit != nil {
				p.OnSourceEdit(repository)
			}
		})
		ui.Text(c, whaleTaskCopy("使用本机 GitHub CLI 的登录。可填写完整 github.com 仓库地址。", "Uses your GitHub CLI sign-in. A full github.com repository URL also works.")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		if p.SourceError != "" {
			ui.Text(c, p.SourceError).TextColor(accent).Selectable()
		}
		ui.Row(c).Children(func() {
			if ui.Button(c, whaleTaskCopy("添加来源", "Add source")).Disabled(p.SourceBusy || p.OnSourceAdd == nil || strings.TrimSpace(repository) == "").Clicked() {
				p.OnSourceAdd()
			}
		})
	})
}

func proactiveUnread(notices []proactive.Notice) int {
	count := 0
	for _, n := range notices {
		if n.ReadAt == "" {
			count++
		}
	}
	return count
}

func proactiveNotices(c *ui.Context, p ProactiveModel, fg, muted, border, accent ui.Color) {
	if len(p.State.Notices) == 0 {
		return
	}
	group := ui.Column(c.Key("proactive-notices")).FillWidth().MinWidth(0).Gap(8)
	history := ui.Local(group, "history", func() bool { return false })
	expanded := ui.Local(group, "expanded", func() string { return "" })
	group.Children(func() {
		ui.Row(c).Gap(8).Wrap().Children(func() {
			ui.Text(c, fmt.Sprintf(whaleTaskCopy("动态 · %d 条未读", "Updates · %d unread"), proactiveUnread(p.State.Notices))).FontSize(theme.InterfaceSize(15)).FontWeight(600).TextColor(fg)
			label := whaleTaskCopy("查看历史", "Show history")
			if *history {
				label = whaleTaskCopy("仅看未读", "Unread only")
			}
			if ui.Button(c, label).Clicked() {
				*history = !*history
			}
		})
		for i := len(p.State.Notices) - 1; i >= 0; i-- {
			n := p.State.Notices[i]
			if n.ReadAt != "" && !*history && *expanded != n.ID {
				continue
			}
			ui.Column(c.Key(n.ID)).FillWidth().MinWidth(0).Gap(6).Padding(8, 0).Children(func() {
				title := n.Title
				if n.Kind == "discovery" {
					title += fmt.Sprintf(whaleTaskCopy(" · %s 项新事项", " · %s new items"), n.Body)
				} else {
					title += whaleTaskCopy(" · 结果待核对", " · Result ready")
				}
				ui.Text(c, title).FontSize(theme.InterfaceSize(14)).FontWeight(600).TextColor(fg)
				ui.Text(c, proactiveTime(n.CreatedAt)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
				ui.Row(c).Gap(8).Wrap().Children(func() {
					if ui.Button(c, whaleTaskCopy("查看动态", "View update")).Clicked() {
						*expanded = n.ID
						if p.OnNoticeOpen != nil {
							p.OnNoticeOpen(n)
						}
					}
					if n.ReadAt == "" && ui.Button(c, whaleTaskCopy("标为已读", "Mark read")).Disabled(p.SourceBusy || p.OnNoticeRead == nil).Clicked() {
						p.OnNoticeRead(n.ID)
					}
				})
				if *expanded == n.ID {
					if n.Kind == "result" {
						ui.Text(c, n.Body).FontSize(theme.InterfaceSize(13)).TextColor(fg).Selectable()
					}
					status := map[string]string{"pending": "等待桌面提醒", "sending": "正在提交桌面提醒", "shown": "已提交桌面提醒", "in-app": "已送达应用内", "unavailable": "桌面提醒不可用，已保留应用内记录", "failed": "桌面提醒失败", "unknown": "上次桌面提醒结果未知"}[n.DesktopStatus]
					if status != "" {
						ui.Text(c, whaleTaskCopy(status, n.DesktopStatus)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
					}
					if n.DesktopError != "" {
						ui.Text(c, n.DesktopError).TextColor(accent).Selectable()
					}
					if n.DesktopStatus == "failed" || n.DesktopStatus == "unknown" || n.DesktopStatus == "unavailable" {
						ui.Row(c).Children(func() {
							if ui.Button(c, whaleTaskCopy("重试桌面提醒", "Retry desktop notice")).Disabled(p.SourceBusy || p.OnNoticeRetry == nil).Clicked() {
								p.OnNoticeRetry(n.ID)
							}
						})
					}
				}
				ui.Box(c).FillWidth().Height(.5).Background(border)
			})
		}
	})
}
