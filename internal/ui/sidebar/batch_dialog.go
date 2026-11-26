package sidebar

import (
	"fmt"
	"strconv"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
)

// Batch confirmation belongs to the root, independent of the folded tree or
// narrow rail. Targets are the Store's frozen request, never current choices.
func batchDialog(c *ui.Context, m Model) {
	batchDialogPtr(c, &m)
}

func batchDialogPtr(c *ui.Context, m *Model) {
	b := m.Batch
	if !b.Confirming && !b.Busy {
		return
	}
	onCancel := m.OnBatchCancel
	onConfirm := m.OnBatchConfirm
	t := tok(c)
	open := true
	cancel := func() {
		if !b.Busy && onCancel != nil {
			onCancel(b.Token)
		}
	}
	countStr := strconv.Itoa(len(b.Targets))
	warning := i18n.T(i18n.KeyWSSelectionConfirmArchiveDesc)
	title := i18n.WSf(i18n.KeyWSSelectionConfirmArchiveTitle, "n", countStr)
	actionTip := i18n.T(i18n.KeyWSSelectionConfirmArchiveAction)
	confirmVariant := coss.ButtonDefault
	if b.Kind == "delete" {
		title = i18n.WSf(i18n.KeyWSSelectionConfirmDeleteTitle, "n", countStr)
		warning = i18n.T(i18n.KeyWSSelectionConfirmDeleteDesc)
		actionTip = i18n.T(i18n.KeyWSSelectionConfirmDeleteAction)
		confirmVariant = coss.ButtonDestructive
	}
	coss.AlertDialog(c, &open, coss.DialogOptions{
		Title:           title,
		Description:     warning,
		MaxWidth:        480,
		DismissDisabled: b.Busy,
		Key:             fmt.Sprintf("session-batch-dialog-%d", b.Token),
		Footer: func() {
			closeBtn := coss.Button(c, workspaceText("取消", "Cancel batch"), coss.ButtonOptions{
				Variant:  coss.ButtonOutline,
				Size:     coss.ButtonMD,
				Disabled: b.Busy,
				Key:      "batch-dialog-cancel",
			}).MinWidth(72)
			first := ui.Local(closeBtn, "initial-focus", func() bool { return true })
			if *first && !b.Busy {
				closeBtn.Focus()
				*first = false
			}
			if closeBtn.Clicked() {
				cancel()
			}
			if coss.Button(c, workspaceText("确认批量操作", "Confirm batch"), coss.ButtonOptions{
				Variant:  confirmVariant,
				Size:     coss.ButtonMD,
				Disabled: b.Busy || !b.Confirming || onConfirm == nil,
				Loading:  b.Busy,
				Key:      "batch-dialog-confirm",
			}).Tooltip(actionTip).MinWidth(88).Clicked() && !b.Busy && b.Confirming && onConfirm != nil {
				onConfirm(b.Token)
			}
		},
	}, func() {
		coss.ScrollArea(c, coss.ScrollAreaOptions{Key: "batch-dialog-targets"}, func() {
			ui.Column(c).FillWidth().Gap(design.Sp(0.5)).Children(func() {
				for _, target := range b.Targets {
					ui.Text(c, target.Title+" · "+target.ID).FillWidth().FontSize(fs(12)).FixedLineHeight(18).TextColor(t.MutedForeground)
				}
			})
		}).Label(i18n.T(i18n.KeyWSSelectionConfirmTargets)).FillWidth().Height(168)
		if b.Busy {
			ui.Text(c, fmt.Sprintf(workspaceText("处理中 %d / %d", "Processing %d / %d"), b.Attempted, len(b.Targets))).FillWidth().FontSize(fs(12)).TextColor(t.MutedForeground)
		}
	})
	if !open {
		cancel()
	}
}
