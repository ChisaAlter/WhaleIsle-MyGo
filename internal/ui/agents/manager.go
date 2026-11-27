package agents

import (
	"context"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/errcopy"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	fonttheme "whaleisle/internal/ui/theme"
)

func presetManagerView(c *ui.Context, manager *session.AgentPresetManager, fg, muted ui.Color) {
	s := manager.Snapshot()
	presetManagerStateView(c, s, func() {
		go manager.Refresh(context.Background())
	}, func(id string) {
		go manager.SaveDefault(context.Background(), id)
	}, fg, muted)
}

func presetManagerStateView(c *ui.Context, s session.AgentPresetManagerState, onRefresh func(), onSaveDefault func(string), fg, muted ui.Color) {
	border := c.Theme().Border
	danger := design.For(c).DestructiveForeground
	ui.Column(c).FillWidth().Padding(10, 12, 10, 12).Gap(6).Children(func() {
		ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, agentText("Agent 预设管理", "Agent preset manager")).FontSize(fonttheme.InterfaceSize(13)).FontWeight(600).TextColor(fg).Grow(1)
			if !s.Busy && onRefresh != nil && coss.Button(c, agentText("刷新预设", "Refresh presets"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonXS}).Clicked() {
				onRefresh()
			}
		})
		ui.Text(c, agentText("默认预设用于后续新任务；当前会话保持原配置。", "The default preset applies to new tasks; the current session keeps its existing setup.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		if s.Err != "" {
			ui.Text(c, errcopy.SentenceText(s.Err)).FontSize(fonttheme.InterfaceSize(11)).TextColor(danger)
		}
		if s.Status != "" {
			ui.Text(c, s.Status).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg)
		}
		if !s.Loaded {
			ui.Text(c, agentText("刷新以加载 Host 的预设和保存权限。", "Refresh to load Host presets and save permissions.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			return
		}
		if !s.Writable {
			ui.Text(c, agentText("预设默认配置只读或不可用", "Default preset configuration is read-only or unavailable.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		if len(s.Rows) == 0 {
			ui.Text(c, agentText("暂无 Agent 预设", "No Agent presets available.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		for _, row := range s.Rows {
			ui.Column(c).FillWidth().Gap(4).Radius(10).Border(0.5, border).Padding(10, 12).Children(func() {
				ui.Row(c).FillWidth().Gap(6).AlignItems(ui.Center).Children(func() {
					label := row.Name + " (" + row.ID + ")"
					if row.ID == s.DefaultID {
						label += agentText(" · 默认", " · Default")
					}
					if row.Hidden {
						label += agentText(" · 隐藏", " · Hidden")
					}
					ui.Text(c, label).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Grow(1)
					if s.Writable && !s.Busy && row.Broken == "" && !row.Hidden && row.ID != s.DefaultID && onSaveDefault != nil {
						if coss.Button(c, agentText("设为默认", "Set as default"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonXS}).Clicked() {
							onSaveDefault(row.ID)
						}
					}
				})
				if row.Description != "" {
					ui.Text(c, row.Description).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				}
				if row.Broken != "" {
					ui.Text(c, agentText("配置不可用：", "Configuration unavailable: ")+errcopy.SentenceText(row.Broken)).FontSize(fonttheme.InterfaceSize(11)).TextColor(danger)
				}
			})
		}
	})
}
