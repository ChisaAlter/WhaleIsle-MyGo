package projects

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

func taskDeliveryChangeActions(c *ui.Context, m Model, card projectflow.RequestCard) {
	if card.Delivery == nil || card.Status != "completed" && card.Delivery.Outcome != "partial" {
		return
	}
	box := ui.Column(c.Key("delivery-changes/" + card.ID)).FillWidth().MinWidth(0).Gap(6)
	type confirmation struct {
		Open     bool
		Revision int
	}
	ask := ui.Local(box, "revert-confirmation", func() confirmation { return confirmation{} })
	if card.Changes != nil && card.Changes.RevertedAt != nil {
		ask.Open = false
	}
	box.Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).Children(func() {
			if card.Changes != nil && card.Changes.IsGit {
				ui.Text(c, fmt.Sprintf(tr("%d 个文件", "%d files"), card.Changes.FileCount)).FontSize(theme.InterfaceSize(12))
				if card.Changes.Added > 0 || card.Changes.Removed > 0 {
					ui.Text(c, fmt.Sprintf("+%d −%d", card.Changes.Added, card.Changes.Removed)).FontSize(theme.InterfaceSize(12))
				}
				if ui.Button(c, tr("查看改动", "View changes")).Height(28).Disabled(m.Busy || !m.Listing.Available || m.OnRequestDiff == nil).Clicked() {
					m.OnRequestDiff(card.ID)
				}
			} else {
				for _, work := range card.Work {
					if work.MemberRole != "" && work.MemberRole != "readonly" {
						if ui.Button(c, tr("查看改动", "View changes")).Height(28).Disabled(m.Busy || !taskCardCurrent(m, card) || m.OnWorkDiff == nil).Clicked() {
							m.OnWorkDiff(work.WorkstreamID)
						}
						break
					}
				}
			}
			if card.Changes != nil && card.Changes.Revertible && card.Changes.RevertedAt == nil && m.RequestMutable {
				if ui.Button(c, tr("撤销改动", "Revert changes")).Height(28).Disabled(m.Busy || !taskCardCurrent(m, card) || m.OnRevertRequest == nil).Clicked() {
					ask.Open, ask.Revision = true, card.Revision
				}
			}
		})
	})
	if !ask.Open {
		return
	}
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	conflict := strings.Contains(m.RequestRevertError, "pluginrpc: revert_conflict:")
	ui.DialogBase(c.Key("revert/"+card.ID), &ask.Open, func(back, panel ui.Element) {
		back.Background(ui.RGBA(0, 0, 0, .4))
		panel.Width(440).MaxWidthPercent(94).Padding(20).Gap(14).Radius(16).Background(chrome.Color(colors.Field)).Children(func() {
			title := tr("撤销此要求的改动？", "Revert this requirement’s changes?")
			body := tr("此要求改动的文件将恢复到开始开发前的状态，其他文件不受影响。如文件在交付后又被修改，将不执行撤销。", "Files changed by this requirement return to their state before development started; other files are untouched. If they changed again after delivery, nothing is reverted.")
			if conflict {
				title = tr("无法自动撤销", "Could not revert automatically")
				body = tr("文件在交付后又被修改，自动撤销会覆盖较新的编辑，因此没有改动任何文件。", "These files changed after delivery. Reverting automatically would overwrite newer edits, so nothing was changed.")
			}
			ui.Text(c, title).FontSize(theme.InterfaceSize(16)).FontWeight(600)
			ui.Text(c, body).FillWidth().FontSize(theme.InterfaceSize(13))
			if m.RequestRevertError != "" {
				ui.Text(c, m.RequestRevertError).FillWidth().FontSize(theme.InterfaceSize(12)).Selectable()
			}
			stale := ask.Revision != card.Revision || !taskCardCurrent(m, card)
			if stale {
				ui.Text(c, tr("要求已变化，请刷新后重新确认。", "The requirement changed; refresh and confirm again.")).FontSize(theme.InterfaceSize(12))
			}
			ui.Row(c).FillWidth().Gap(8).Children(func() {
				ui.Box(c).Grow(1)
				if ui.Button(c, tr("取消", "Cancel")).Disabled(m.Busy).Clicked() {
					ask.Open = false
				}
				if conflict {
					if ui.PrimaryButton(c, tr("交给协调者处理", "Ask the coordinator")).Disabled(m.Busy || stale || !m.RequestMutable || m.OnRevertAskDraft == nil).Clicked() {
						ask.Open = false
						m.OnRevertAskDraft()
					}
				} else if ui.PrimaryButton(c, tr("撤销改动", "Revert changes")).Disabled(m.Busy || stale || !m.RequestMutable || m.OnRevertRequest == nil).Clicked() {
					m.OnRevertRequest(card.ID, ask.Revision)
				}
			})
		})
	})
}

func requestChangesView(c *ui.Context, m Model, value projectflow.RequestChanges) {
	text := tr("自开始开发以来此要求产生的改动；尚未交付。", "Changes made by this requirement since development started; not delivered yet.")
	if value.Final {
		text = tr("此要求从开始开发到交付所产生的改动。", "Changes made by this requirement from the start of development to delivery.")
	}
	ui.Text(c, text).FontSize(theme.InterfaceSize(13))
	if value.RevertedAt != nil {
		ui.Text(c, tr("这些改动已被撤销。", "These changes have been reverted.")).FontSize(theme.InterfaceSize(13))
	}
	if value.Isolated {
		ui.Text(c, tr("部分工作在独立 worktree 中进行；其改动不在此列，也不能从这里撤销。", "Part of this work ran in an isolated worktree; those changes are not listed here and cannot be reverted from here.")).FontSize(theme.InterfaceSize(13))
	}
	scopes := value.Workspaces
	if scopes == nil {
		scopes = []projectflow.RequestChangeScope{value.RequestChangeScope}
	}
	for _, scope := range scopes {
		if value.Truncated {
			scope.Truncated = true
		}
		renderRequestChangeScope(c, m, scope)
	}
}
