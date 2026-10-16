package settings

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	fonttheme "whaleisle/internal/ui/theme"
)

type CreatorTaskAccess struct {
	State                                session.CreatorTaskState
	OnPrepare                            func()
	OnCancel, OnCreate, OnLocate, OnOpen func(uint64)
}

func creatorTaskPane(c *ui.Context, a *CreatorTaskAccess, roster session.AgentPresetManagerState, fg, muted ui.Color) {
	if a == nil {
		return
	}
	s := a.State
	available := false
	for _, row := range roster.Rows {
		if row.ID == "cordis" && !row.Hidden && row.Broken == "" {
			available = true
		}
	}
	ui.Column(c.Key("creator-task-entry")).Gap(7).Children(func() {
		ui.Text(c, "Creator 新任务 / Creator task").FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg)
		ui.Text(c, "创建独立任务并为该任务选择 cordis，随后返回输入区；不会发送模型提示词，也不会改变全局默认预设。 / Creates a new cordis task and opens its composer; no prompt or global default change.").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		if !roster.Loaded {
			ui.Text(c, "请先刷新预设列表 / Refresh presets first").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		} else if !available {
			ui.Text(c, "当前 Host 未提供可用 cordis；无法创建 Creator 任务 / cordis is missing, hidden or broken").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		if s.Error != "" {
			ui.Text(c, s.Error).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg).Selectable()
		}
		if s.Review || s.Attempted {
			ui.Text(c, fmt.Sprintf("Host: %s\nWorkspace: %s\nSession: %s\nPreset: cordis · %s", s.Host, s.Cwd, s.SessionID, s.Phase)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
		}
		if s.Review {
			ui.Text(c, "确认后仅对上面的新任务创建并应用 cordis；离开本页或切换 Host/会话会使确认失效。 / Confirm this exact new task. Leaving this page or changing Host/session invalidates the review.").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			ui.Row(c).Gap(8).Children(func() {
				if ui.Button(c, "取消 / Cancel Creator").Disabled(s.Busy || a.OnCancel == nil).Clicked() && a.OnCancel != nil {
					a.OnCancel(s.Ticket)
				}
				if ui.Button(c, "确认创建 Creator 任务 / Create Creator task").Disabled(s.Busy || !available || a.OnCreate == nil).Clicked() && a.OnCreate != nil {
					a.OnCreate(s.Ticket)
				}
			})
			return
		}
		if s.Attempted && !s.Opened {
			ui.Row(c).Gap(8).Children(func() {
				if !s.Created && ui.Button(c, "定位已提交目标 / Locate attempted task").Disabled(s.Busy || a.OnLocate == nil).Clicked() && a.OnLocate != nil {
					a.OnLocate(s.Ticket)
				}
				if s.Created && ui.Button(c, "打开已有任务 / Open existing task").Disabled(s.Busy || a.OnOpen == nil).Clicked() && a.OnOpen != nil {
					a.OnOpen(s.Ticket)
				}
			})
			return
		}
		if ui.Button(c, "新建 Creator 任务… / New Creator task…").Disabled(!roster.Loaded || !available || roster.Busy || s.Busy || a.OnPrepare == nil).Clicked() && a.OnPrepare != nil {
			a.OnPrepare()
		}
	})
}
