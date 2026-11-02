package agents

import (
	"context"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	fonttheme "whaleisle/internal/ui/theme"
)

func presetManagerView(c *ui.Context, manager *session.AgentPresetManager, fg, muted ui.Color) {
	s := manager.Snapshot()
	border := c.Theme().Border
	ui.Column(c).FillWidth().Padding(10, 12, 10, 12).Gap(6).Children(func() {
		ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Agent 预设管理").FontSize(fonttheme.InterfaceSize(13)).FontWeight(600).TextColor(fg).Grow(1)
			if !s.Busy && ui.Button(c, "刷新预设").Height(24).Padding(0, 8).Radius(6).Border(0.5, border).FontSize(fonttheme.InterfaceSize(11)).Clicked() {
				go manager.Refresh(context.Background())
			}
		})
		ui.Text(c, "默认预设用于后续新任务；当前会话保持原配置。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		if s.Busy {
			ui.Text(c, "正在处理…").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		if s.Err != "" {
			ui.Text(c, s.Err).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg)
		}
		if s.Status != "" {
			ui.Text(c, s.Status).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg)
		}
		if !s.Loaded {
			ui.Text(c, "刷新以加载 Host 的预设和保存权限。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			return
		}
		if !s.Writable {
			ui.Text(c, "预设默认配置只读或不可用").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		if len(s.Rows) == 0 {
			ui.Text(c, "暂无 Agent 预设").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		for _, row := range s.Rows {
			ui.Column(c).FillWidth().Gap(4).Radius(10).Border(0.5, border).Padding(10, 12).Children(func() {
				ui.Row(c).FillWidth().Gap(6).AlignItems(ui.Center).Children(func() {
					label := row.Name + " (" + row.ID + ")"
					if row.ID == s.DefaultID {
						label += " · 默认"
					}
					if row.Hidden {
						label += " · 隐藏"
					}
					ui.Text(c, label).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Grow(1)
					if s.Writable && !s.Busy && row.Broken == "" && !row.Hidden && row.ID != s.DefaultID {
						if ui.Button(c, "设为默认").Height(24).Padding(0, 8).Radius(6).Border(0.5, border).FontSize(fonttheme.InterfaceSize(11)).Clicked() {
							id := row.ID
							go manager.SaveDefault(context.Background(), id)
						}
					}
				})
				if row.Description != "" {
					ui.Text(c, row.Description).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				}
				if row.Broken != "" {
					ui.Text(c, "配置不可用："+row.Broken).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg)
				}
			})
		}
	})
}
