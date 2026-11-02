package sidebar

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

// Batch confirmation belongs to the root, independent of the folded tree or
// narrow rail. Targets are the Store's frozen request, never current choices.
func batchDialog(c *ui.Context, m Model) {
	b := m.Batch
	if !b.Confirming && !b.Busy {
		return
	}
	open := true
	cancel := func() {
		if !b.Busy && m.OnBatchCancel != nil {
			m.OnBatchCancel(b.Token)
		}
	}
	ui.DialogBase(c.Key(fmt.Sprintf("session-batch-dialog-%d", b.Token)), &open, func(backdrop, panel ui.Element) {
		warning := workspaceText("归档这些会话将停止其中正在进行的工作。之后可从归档列表恢复。", "Archiving these sessions stops their active work. You can restore them from the archive list.")
		title := fmt.Sprintf(workspaceText("归档 %d 个会话", "Archive %d sessions"), len(b.Targets))
		if b.Kind == "delete" {
			title = fmt.Sprintf(workspaceText("删除 %d 个会话", "Delete %d sessions"), len(b.Targets))
			warning = workspaceText("将永久删除所选会话及其关联子会话的对话记录，无法恢复。工作区文件不受影响。运行中的回合须先停止；无法删除的会话会保留，并在结果中列出原因。", "This permanently deletes the selected conversation logs and their linked subagent sessions. This cannot be undone. Workspace files are unchanged. Running turns must be stopped first. Sessions that cannot be deleted will be kept with their reasons listed in the results.")
		}
		chrome.DialogSurfaceSized(c, backdrop, panel, c.Theme(), chrome.DialogSurfaceBox{Width: 480, Radius: 14, Padding: 24}).Label(title).Gap(16).Children(func() {
			ui.Text(c, title).FillWidth().FontSize(theme.InterfaceSize(16)).FontWeight(500).FixedLineHeight(24).TextColor(c.Theme().Text)
			ui.Text(c, warning).FillWidth().FontSize(theme.InterfaceSize(14)).FixedLineHeight(22).TextColor(c.Theme().Text)
			ui.Scroll(c).FillWidth().Height(168).Children(func() {
				for _, target := range b.Targets {
					ui.Text(c, target.Title+" · "+target.ID).FillWidth().FontSize(theme.InterfaceSize(12)).FixedLineHeight(18).TextColor(c.Theme().TextMuted)
				}
			})
			if b.Busy {
				ui.Text(c, fmt.Sprintf(workspaceText("处理中 %d / %d", "Processing %d / %d"), b.Attempted, len(b.Targets))).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(c.Theme().TextMuted)
			}
			ui.Row(c).FillWidth().Justify(ui.End).Gap(8).Children(func() {
				close := ui.Button(c.Key("batch-dialog-cancel"), workspaceText("取消", "Cancel batch")).MinWidth(72).Height(32).Disabled(b.Busy)
				first := ui.Local(close, "initial-focus", func() bool { return true })
				if *first && !b.Busy {
					close.Focus()
					*first = false
				}
				if close.Clicked() {
					cancel()
				}
				if ui.Button(c.Key("batch-dialog-confirm"), workspaceText("确认批量操作", "Confirm batch")).MinWidth(88).Height(32).Disabled(b.Busy || !b.Confirming || m.OnBatchConfirm == nil).Clicked() && !b.Busy && b.Confirming && m.OnBatchConfirm != nil {
					m.OnBatchConfirm(b.Token)
				}
			})
		})
	})
	if !open {
		cancel()
	}
}
