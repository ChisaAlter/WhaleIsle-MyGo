package settings

import (
	"fmt"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
)

type AgentPresetReaderAccess struct {
	State                   session.AgentPresetReaderState
	OnSelect                func(uint64, string)
	OnRead, OnClose, OnCopy func(uint64)
}

func agentPresetReaderPane(c *ui.Context, a *AgentPresetReaderAccess, roster session.AgentPresetManagerState, fg, muted ui.Color) {
	if a == nil {
		return
	}
	s := a.State
	ui.Column(c.Key("agent-preset-reader")).Gap(7).Children(func() {
		ui.Text(c, "预设声明 · 只读 / Preset declaration · Read only").FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg)
		ui.Text(c, "内容和元数据来自当前 Host 的 agentPresets/read；仅阅读和复制，不执行文档中的指令。 / Content and metadata come from the current Host's agentPresets/read; document instructions remain inert.").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		if s.Host == "" {
			ui.Text(c, "Host 未连接，暂时无法读取 / Connect a Host to read declarations").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		labels, ids := []string{}, map[string]string{}
		selected := ""
		broken := ""
		for _, row := range roster.Rows {
			// Hidden failures are deliberately readable, as in the original roster.
			if row.Hidden && row.Broken == "" {
				continue
			}
			label := agentPresetSettingsLabel(row)
			labels = append(labels, label)
			ids[label] = row.ID
			if row.ID == s.SelectedID {
				selected = label
				broken = row.Broken
			}
		}
		if len(labels) == 0 {
			ui.Text(c, "暂无可读预设；请刷新列表 / No presets to read; refresh the roster").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		if ui.Select(c.Key("preset-reading-choice"), &selected, labels).FillWidth().Disabled(!roster.Loaded || len(labels) == 0).Changed() && a.OnSelect != nil {
			a.OnSelect(s.Ticket, ids[selected])
		}
		if broken != "" {
			ui.Text(c, "预设诊断 / Preset diagnostic: "+broken).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg).Selectable()
		}
		ui.Row(c).Gap(8).Children(func() {
			if ui.Button(c, "读取声明 / Read declaration").Disabled(s.Busy || s.SelectedID == "" || a.OnRead == nil).Clicked() && a.OnRead != nil {
				a.OnRead(s.Ticket)
			}
			if s.Open && ui.Button(c, "关闭阅读 / Close reader").Clicked() && a.OnClose != nil {
				a.OnClose(s.Ticket)
			}
		})
		if !s.Open {
			return
		}
		ui.Text(c, "Host: "+s.Host+"\nPreset ID: "+s.SelectedID).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
		if s.Busy {
			ui.Text(c, "正在读取声明… / Loading declaration…").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			return
		}
		if s.Error != "" {
			ui.Text(c, s.Error+"\n可再次点击读取重试 / Use Read declaration to retry").FontSize(fonttheme.InterfaceSize(11)).TextColor(fg).Selectable()
			return
		}
		if !s.Loaded {
			return
		}
		ui.Text(c, "Name: "+s.Name).FontSize(fonttheme.InterfaceSize(12)).Bold().TextColor(fg).Selectable()
		if s.Description != "" {
			ui.Text(c, "Description: "+s.Description).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
		}
		if s.MetadataTruncated {
			ui.Text(c, "名称或说明超过显示上限，已截断 / Name or description truncated at the display limit").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		copyLabel := "复制原文 / Copy original content"
		if s.Truncated {
			copyLabel = "复制已显示片段 / Copy displayed fragment"
			ui.Text(c, fmt.Sprintf("内容已截断：显示 %d / %d 字节，显示上限 %d 字节；复制仅含已显示片段。 / Content truncated; copy includes only the displayed fragment.", len(s.Content), s.TotalBytes, session.AgentPresetContentLimit)).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg)
		}
		if ui.Button(c, copyLabel).Disabled(a.OnCopy == nil).Clicked() && a.OnCopy != nil {
			a.OnCopy(s.Ticket)
		}
		if s.Content == "" {
			ui.Text(c, "该预设声明内容为空 / This preset declaration is empty").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			return
		}
		content := s.Content
		ui.TextArea(c.Key(fmt.Sprintf("preset-document-%d", s.Ticket)), &content).FillWidth().Height(320).ReadOnly(true).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
	})
}
