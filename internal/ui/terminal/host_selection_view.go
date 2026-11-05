package terminal

import (
	"whaleisle/internal/ui/widgets"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/hostterminal"
)

func (h *HostState) selectionActions(c *ui.Context, store *hostterminal.Store, v hostterminal.View, session string, actions HostActions) {
	text, selected := h.terminal.SelectionText()
	if !selected || text == "" {
		return
	}
	term, attachment := h.terminal, h.attachment
	valid := func() bool {
		if h.terminal != term || h.attachment != attachment || !store.CanCopySelection(v.Owner, session, v.ActiveID, attachment) {
			return false
		}
		current, ok := term.SelectionText()
		return ok && current == text
	}
	run := func(action func(string) error) {
		if err := ApplyHostSelection(text, valid, action); err != nil {
			h.selectionNotice = err.Error()
		} else {
			h.selectionNotice = terminalLabel("选区操作已提交", "Selection action requested")
		}
	}
	ui.Row(c).Gap(4).Children(func() {
		if ui.Button(c, terminalLabel("加入对话", "Add to chat")).Key("host-quote:" + v.ActiveID + text).FontSize(10).Disabled(!actions.CanAddToChat || actions.AddToChat == nil).Clicked() {
			run(func(text string) error { return actions.AddToChat(FormatSelection(text)) })
		}
		if target, ok := FirstSelectionTarget(text); ok {
			label := terminalLabel("打开路径", "Open path")
			if target.URL {
				label = terminalLabel("打开链接", "Open link")
			}
			if ui.Button(c, label).Key("host-target:" + v.ActiveID + text).FontSize(10).Disabled(actions.OpenTarget == nil).Clicked() {
				run(func(text string) error { return actions.OpenTarget(text, v.Info.Cwd) })
			}
		}
	})
	widgets.Notice(c, "host_selection_view:43", "", h.selectionNotice != "", func() string { return h.selectionNotice })
}
