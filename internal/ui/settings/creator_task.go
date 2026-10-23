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
		ui.Text(c, providerCopy("Creator 新任务", "Creator task")).FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg)
		ui.Text(c, providerCopy("创建独立任务并为该任务选择 cordis，随后返回输入区；不会发送模型提示词，也不会改变全局默认预设。", "Creates a new cordis task and opens its composer; no prompt or global default change.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		if !roster.Loaded {
			ui.Text(c, providerCopy("请先刷新预设列表", "Refresh presets first")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		} else if !available {
			ui.Text(c, providerCopy("当前 Host 未提供可用 cordis；无法创建 Creator 任务", "cordis is missing, hidden or broken")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		if s.Error != "" {
			ui.Text(c, s.Error).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg).Selectable()
		}
		if s.Review || s.Attempted {
			ui.Text(c, fmt.Sprintf("Host: %s\nWorkspace: %s\nSession: %s\nPreset: cordis · %s", s.Host, s.Cwd, s.SessionID, s.Phase)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
		}
		if s.Attempted && !s.Opened {
			ui.Row(c).Gap(8).Children(func() {
				if !s.Created && ui.Button(c, providerCopy("定位已提交目标", "Locate attempted task")).Disabled(s.Busy || a.OnLocate == nil).Clicked() && a.OnLocate != nil {
					a.OnLocate(s.Ticket)
				}
				if s.Created && ui.Button(c, providerCopy("打开已有任务", "Open existing task")).Disabled(s.Busy || a.OnOpen == nil).Clicked() && a.OnOpen != nil {
					a.OnOpen(s.Ticket)
				}
			})
			return
		}
		if ui.Button(c, providerCopy("新建 Creator 任务", "New Creator task")).AlignSelf(ui.Start).Height(32).Padding(0, 14).Disabled(!roster.Loaded || !available || roster.Busy || s.Busy || a.OnPrepare == nil).Clicked() && a.OnPrepare != nil {
			a.OnPrepare()
		}
	})
}
