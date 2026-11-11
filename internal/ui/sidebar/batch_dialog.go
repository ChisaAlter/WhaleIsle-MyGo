package sidebar

import (
	"fmt"
	"strconv"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
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
		countStr := strconv.Itoa(len(b.Targets))
		warning := i18n.T(i18n.KeyWSSelectionConfirmArchiveDesc)
		title := i18n.WSf(i18n.KeyWSSelectionConfirmArchiveTitle, "n", countStr)
		actionTip := i18n.T(i18n.KeyWSSelectionConfirmArchiveAction)
		if b.Kind == "delete" {
			title = i18n.WSf(i18n.KeyWSSelectionConfirmDeleteTitle, "n", countStr)
			warning = i18n.T(i18n.KeyWSSelectionConfirmDeleteDesc)
			actionTip = i18n.T(i18n.KeyWSSelectionConfirmDeleteAction)
		}
		chrome.DialogSurfaceSized(c, backdrop, panel, c.Theme(), chrome.DialogSurfaceBox{Width: 480, Radius: 14, Padding: 24}).Label(title).Gap(16).Children(func() {
			ui.Text(c, title).FillWidth().FontSize(theme.InterfaceSize(16)).FontWeight(500).FixedLineHeight(24).TextColor(c.Theme().Text)
			ui.Text(c, warning).FillWidth().FontSize(theme.InterfaceSize(14)).FixedLineHeight(22).TextColor(c.Theme().Text)
			ui.Scroll(c).Label(i18n.T(i18n.KeyWSSelectionConfirmTargets)).FillWidth().Height(168).Children(func() {
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
				if ui.Button(c.Key("batch-dialog-confirm"), workspaceText("确认批量操作", "Confirm batch")).Tooltip(actionTip).MinWidth(88).Height(32).Disabled(b.Busy || !b.Confirming || m.OnBatchConfirm == nil).Clicked() && !b.Busy && b.Confirming && m.OnBatchConfirm != nil {
					m.OnBatchConfirm(b.Token)
				}
			})
		})
	})
	if !open {
		cancel()
	}
}
