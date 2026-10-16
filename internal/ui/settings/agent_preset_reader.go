package settings

import (
	"fmt"
	"unicode/utf8"
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
		selected := ""
		broken := ""
		for _, row := range roster.Rows {
			if row.ID == s.SelectedID {
				selected = agentPresetSettingsLabel(row)
				broken = row.Broken
			}
		}
		if selected == "" {
			ui.Text(c, "从上方目录选择要阅读的预设 / Select a preset to read from the roster above").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		} else {
			ui.Text(c, "阅读选择 / Reading choice: "+selected).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
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
		if ui.Button(c, "复制全文 / Copy full declaration").Disabled(a.OnCopy == nil).Clicked() && a.OnCopy != nil {
			a.OnCopy(s.Ticket)
		}
		if s.Content == "" {
			ui.Text(c, "该预设声明内容为空 / This preset declaration is empty").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			return
		}
		viewer := ui.Column(c.Key(fmt.Sprintf("preset-document-%d", s.Ticket))).Gap(6)
		pages := ui.Local(viewer, "pages", func() presetDocumentPages { return presetDocumentPages{} })
		viewer.Children(func() {
			content, next := agentPresetDocumentPage(s.Content, pages.Start)
			if len(s.Content) > session.AgentPresetContentLimit {
				ui.Text(c, fmt.Sprintf("显示字节 / Showing bytes %d–%d / %d；可翻页阅读，复制包含全文 / Browse every page; copy includes the full declaration", pages.Start+1, next, s.TotalBytes)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				ui.Row(c).Gap(8).Children(func() {
					if ui.Button(c, "上一页 / Previous page").Disabled(len(pages.Previous) == 0).Clicked() {
						last := len(pages.Previous) - 1
						pages.Start = pages.Previous[last]
						pages.Previous = pages.Previous[:last]
					}
					if ui.Button(c, "下一页 / Next page").Disabled(next == len(s.Content)).Clicked() {
						pages.Previous = append(pages.Previous, pages.Start)
						pages.Start = next
					}
				})
			}
			ui.TextArea(c.Key(pages.Start), &content).FillWidth().Height(320).ReadOnly(true).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
		})
	})
}

type presetDocumentPages struct {
	Start    int
	Previous []int
}

func agentPresetDocumentPage(content string, start int) (string, int) {
	end := min(start+session.AgentPresetContentLimit, len(content))
	if end < len(content) {
		for end > start && !utf8.RuneStart(content[end]) {
			end--
		}
	}
	return content[start:end], end
}
