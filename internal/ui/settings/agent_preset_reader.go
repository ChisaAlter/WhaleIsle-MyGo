package settings

import (
	"fmt"
	"unicode/utf8"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
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
	button := coss.Button(c, "", coss.ButtonOptions{
		Variant:  coss.ButtonGhost,
		Size:     coss.ButtonIconSM,
		Label:    providerCopy("查看配置", "View configuration") + " · " + label,
		Disabled: disabled,
		Content: func() {
			ui.Icon(c, presetDocIcon).Size(design.IconSM, design.IconSM).Shrink(0).TextColor(tk(c).Foreground).Opacity(design.OpacityIcon).PassThrough()
		},
	}).Tooltip(providerCopy("查看配置", "View configuration"))
	handle := ui.Local(button, "view-trigger", func() ui.Handle { return ui.Handle{} })
	button.Bind(handle)
	if button.Clicked() {
		if a.focus != nil {
			a.focus.Trigger = handle
		}
		return true
	}
	return false
}

func agentPresetReaderPane(c *ui.Context, a *AgentPresetReaderAccess, roster session.AgentPresetManagerState, fg, muted ui.Color) {
	_ = fg
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
	bodyDialogFooterWidth(c.Key("agent-preset-reader"), &open, providerCopy("查看配置", "View configuration")+" · "+selected, 720, func() {
		if s.Busy {
			ui.Text(c, providerCopy("正在读取声明…", "Loading declaration…")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			return
		}
		if s.Error != "" {
			settingsFailureLine(c, "agent-preset-reader", providerCopy("没能读取 Agent 预设声明，请重试。", "The agent preset declaration could not be read. Try again."), s.Error)
			if kButtonDisabled(c, providerCopy("重试", "Retry"), a.OnRead == nil).Clicked() && a.OnRead != nil {
				a.OnRead(s.Ticket)
			}
			return
		}
		if !s.Loaded {
			return
		}
		if s.Content == "" {
			ui.Text(c, providerCopy("该预设声明内容为空", "This preset declaration is empty")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			return
		}
		viewer := ui.Column(c.Key(fmt.Sprintf("preset-document-%d", s.Ticket))).Gap(6)
		pages := ui.Local(viewer, "pages", func() presetDocumentPages { return presetDocumentPages{} })
		viewer.Children(func() {
			content, next := agentPresetDocumentPage(s.Content, pages.Start)
			if len(s.Content) > session.AgentPresetContentLimit {
				ui.Text(c, fmt.Sprintf(providerCopy("显示字节 %d–%d（共 %d）；可翻页阅读，复制包含全文", "Showing bytes %d–%d of %d; browse every page, copy includes the full declaration"), pages.Start+1, next, s.TotalBytes)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				ui.Row(c).Gap(8).Children(func() {
					if kButtonDisabled(c, providerCopy("上一页", "Previous page"), len(pages.Previous) == 0).Clicked() {
						last := len(pages.Previous) - 1
						pages.Start = pages.Previous[last]
						pages.Previous = pages.Previous[:last]
					}
					if kButtonDisabled(c, providerCopy("下一页", "Next page"), next == len(s.Content)).Clicked() {
						pages.Previous = append(pages.Previous, pages.Start)
						pages.Start = next
					}
				})
			}
			kTextarea(c.Key(pages.Start), &content, coss.TextareaOptions{
				Label:    providerCopy("预设配置只读内容", "Preset configuration source"),
				ReadOnly: true,
			}).Height(320).FontSize(fonttheme.InterfaceSize(12))
		})
	}, func() {
		if kButton(c, providerCopy("关闭", "Close")).Clicked() {
			open = false
		}
		if kPrimary(c, providerCopy("复制全文", "Copy full declaration"), !s.Loaded || s.Busy || a.OnCopy == nil).Clicked() && a.OnCopy != nil {
			a.OnCopy(s.Ticket)
		}
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
