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
	focus                   *presetReaderFocus
	OnSelect                func(uint64, string)
	OnRead, OnClose, OnCopy func(uint64)
}

type presetReaderFocus struct {
	Trigger *ui.Handle
	Open    bool
}

func agentPresetViewAction(c *ui.Context, a *AgentPresetReaderAccess, label string, disabled bool) bool {
	button := ui.Button(c, "").Label("查看配置 / View configuration · "+label).Size(28, 28).Padding(6).Radius(6).BorderWidth(0).Background(ui.Transparent).Disabled(disabled).Tooltip(providerCopy("查看配置", "View configuration"))
	handle := ui.Local(button, "view-trigger", func() ui.Handle { return ui.Handle{} })
	button.Bind(handle).Children(func() { ui.Icon(c, bodyIcon("document")).Size(16, 16) })
	if button.Clicked() {
		if a.focus != nil {
			a.focus.Trigger = handle
		}
		return true
	}
	return false
}

func agentPresetReaderPane(c *ui.Context, a *AgentPresetReaderAccess, roster session.AgentPresetManagerState, fg, muted ui.Color) {
	if a == nil {
		return
	}
	s := a.State
	if !s.Open {
		if a.focus != nil && a.focus.Open && a.focus.Trigger != nil {
			a.focus.Trigger.RequestFocus(c)
		}
		if a.focus != nil {
			a.focus.Open = false
		}
		return
	}
	if a.focus != nil {
		a.focus.Open = true
	}
	selected := s.SelectedID
	for _, row := range roster.Rows {
		if row.ID == s.SelectedID {
			selected = agentPresetSettingsLabel(row)
		}
	}
	open := true
	bodyDialog(c.Key("agent-preset-reader"), &open, providerCopy("查看配置", "View configuration")+" · "+selected, func() {
		if s.Busy {
			ui.Text(c, "正在读取声明… / Loading declaration…").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			return
		}
		if s.Error != "" {
			ui.Text(c, s.Error).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg).Selectable()
			if ui.Button(c, providerCopy("重试", "Retry")).Disabled(a.OnRead == nil).Clicked() && a.OnRead != nil {
				a.OnRead(s.Ticket)
			}
			return
		}
		if !s.Loaded {
			return
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
	if !open && a.OnClose != nil {
		a.OnClose(s.Ticket)
		if a.focus != nil && a.focus.Trigger != nil {
			a.focus.Trigger.RequestFocus(c)
			a.focus.Open = false
		}
	}
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
