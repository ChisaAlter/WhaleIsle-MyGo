package projects

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

// RequestRevertState is scoped to one controller-admitted card revision, so
// another expanded card cannot replace its authorization or conflict result.
type RequestRevertState struct {
	Known, Allowed bool
	Error          string
	Confirm, Ask   func()
}

func taskDeliveryChangeActions(c *ui.Context, m Model, card projectflow.RequestCard) {
	actions := requestCardActions(m, card)
	if card.Delivery == nil || card.Status != "completed" && card.Delivery.Outcome != "partial" {
		return
	}
	box := ui.Column(c.Key("delivery-changes/" + card.ID)).FillWidth().MinWidth(0).Gap(6)
	type confirmation struct {
		Open     bool
		Revision int
	}
	ask := ui.Local(box, "revert-confirmation", func() confirmation { return confirmation{} })
	revert := RequestRevertState{}
	if m.RequestRevert != nil {
		revert = m.RequestRevert(card.ID, card.Revision)
	}
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
						if ui.Button(c, tr("查看改动", "View changes")).Height(28).Disabled(m.Busy || !actions.Known || actions.WorkDiff == nil).Clicked() {
							actions.WorkDiff(work.WorkstreamID)
						}
						break
					}
				}
			}
			if card.Changes != nil && card.Changes.Revertible && card.Changes.RevertedAt == nil && m.RequestMutable {
				if ui.Button(c, tr("撤销改动", "Revert changes")).Height(28).Disabled(m.Busy || !revert.Allowed || revert.Confirm == nil).Clicked() {
					ask.Open, ask.Revision = true, card.Revision
				}
			}
		})
	})
	if !ask.Open {
		return
	}
	// Keep the revision the user actually confirmed, even if a later card read
	// changes the displayed detail while this dialog is open.
	if m.RequestRevert != nil {
		revert = m.RequestRevert(card.ID, ask.Revision)
	}
	resolved := theme.Resolve(m.ThemeID)
	colors := resolved.Active(m.SystemDark)
	dark := resolved.Scheme == theme.SchemeDark || resolved.Scheme == theme.SchemeSystem && m.SystemDark
	mask, surface := ui.RGBA(0, 0, 0, .24), ui.RGBA(255, 255, 255, .8)
	secondary, stroke := ui.RGB(97, 102, 107), ui.RGBA(0, 0, 0, .12)
	elevationStroke := ui.RGBA(0, 0, 0, .16)
	primary, primaryHover, primaryText := ui.RGB(15, 17, 21), ui.RGB(67, 69, 74), ui.RGB(255, 255, 255)
	if dark {
		mask, surface = ui.RGBA(0, 0, 0, .5), ui.RGBA(44, 44, 46, .8)
		secondary, stroke = ui.RGB(207, 211, 214), ui.RGBA(255, 255, 255, .16)
		elevationStroke = ui.RGBA(255, 255, 255, .2)
		primary, primaryHover, primaryText = ui.RGB(249, 250, 251), ui.RGB(235, 238, 242), ui.RGB(15, 17, 21)
	}
	conflict := strings.Contains(revert.Error, "pluginrpc: revert_conflict:")
	close := func() {
		if !m.Busy {
			ask.Open = false
		}
	}
	// A temporary open value lets Escape/mask dismissal use the same busy guard.
	// DialogBase retains its modal scope and returns focus to the original trigger.
	open := true
	viewportWidth, _ := c.Size()
	ui.DialogBase(c.Key("revert/"+card.ID), &open, func(back, panel ui.Element) {
		back.Background(mask).Padding(24)
		// Modal.module.css: a 380px card, 28px panel corners, and a separate
		// padded content region and footer separated by 20px.
		panel.Width(380).MaxWidth(max(1, viewportWidth-48)).Padding(0, 0, 24, 0).Gap(20).Radius(28).Clip().Background(surface).Shadow(0, 0, 0, .5, elevationStroke).Shadow(0, 3, 8, 0, ui.RGBA(0, 0, 0, .04)).Shadow(0, 0, 20, 0, ui.RGBA(0, 0, 0, .05)).Children(func() {
			title := tr("撤销此要求的改动？", "Revert this requirement’s changes?")
			body := tr("此要求改动的文件将恢复到开始开发前的状态，其他文件不受影响。如文件在交付后又被修改，将不执行撤销。", "Files changed by this requirement return to their state before development started; other files are untouched. If they changed again after delivery, nothing is reverted.")
			if conflict {
				title = tr("无法自动撤销", "Could not revert automatically")
				body = tr("文件在交付后又被修改，自动撤销会覆盖较新的编辑，因此没有改动任何文件。", "These files changed after delivery. Reverting automatically would overwrite newer edits, so nothing was changed.")
			}
			panel.Label(title)
			ui.Column(c).FillWidth().MinWidth(0).Children(func() {
				ui.Row(c).FillWidth().MinWidth(0).Padding(22, 14, 12, 24).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, title).FontSize(16).FixedLineHeight(24).FontWeight(500).TextColor(chrome.Color(colors.FG)).Grow(1).MinWidth(0)
					button := ui.Button(c.Key("revert-close"), "").Label(tr("取消", "Cancel")).Width(28).Height(28).Padding(0).Radius(8).BorderWidth(0).Background(ui.Transparent).Disabled(m.Busy)
					if m.Busy {
						button.Opacity(.4)
					} else if button.Hovered() {
						button.Background(chrome.Color(colors.Hover))
					}
					button.Children(func() { projectIcon(c, "close", 14, secondary, false) })
					if button.Clicked() {
						close()
					}
				})
				ui.Column(c).FillWidth().MinWidth(0).Margin(20, 0, 0, 0).Padding(0, 24).Children(func() {
					// The original passes a bare p, not Modal.description: inherited
					// body size/normal line height, with the browser's 1em p margins.
					ui.Text(c, body).FillWidth().FontSize(16).LineHeight(0).Margin(16, 0).TextColor(chrome.Color(colors.FG))
					if revert.Error != "" {
						ui.Scroll(c).FillWidth().MinWidth(0).MaxHeight(160).Padding(8, 12).Radius(8).Background(chrome.Color(colors.Hover)).Children(func() {
							ui.Text(c, revert.Error).FillWidth().MinWidth(0).Font(theme.RenderPreferences().Code.Family).FontSize(12).FixedLineHeight(18).TextColor(secondary).Selectable()
						})
					}
					stale := ask.Revision != card.Revision || !revert.Known
					if stale {
						ui.Text(c, tr("要求已变化，请刷新后重新确认。", "The requirement changed; refresh and confirm again.")).FontSize(12).FixedLineHeight(18)
					}
				})
			})
			stale := ask.Revision != card.Revision || !revert.Known
			styleAction := func(button ui.Element, filled, disabled bool) ui.Element {
				button.Height(36).Padding(0, 14).Radius(12).FontSize(14).FixedLineHeight(22).Disabled(disabled)
				if disabled {
					button.Opacity(.4)
				}
				if filled {
					button.BorderWidth(0).Background(primary).TextColor(primaryText)
					if button.Hovered() && !disabled {
						button.Background(primaryHover)
					}
				} else {
					button.Border(.5, stroke).Background(ui.Transparent).TextColor(chrome.Color(colors.FG))
					if button.Hovered() && !disabled {
						button.Background(chrome.Color(colors.Hover))
					}
				}
				return button
			}
			ui.Row(c).FillWidth().MinWidth(0).Padding(0, 24).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Grow(1)
				ui.Row(c).MinWidth(0).Wrap().Gap(8).AlignItems(ui.Center).Children(func() {
					if styleAction(ui.Button(c, tr("取消", "Cancel")), false, m.Busy).Clicked() {
						close()
					}
					if conflict {
						if styleAction(ui.PrimaryButton(c, tr("交给协调者处理", "Ask the coordinator")), true, m.Busy || stale || !revert.Allowed || revert.Ask == nil).Clicked() {
							ask.Open = false
							revert.Ask()
						}
					} else if styleAction(ui.PrimaryButton(c, tr("撤销改动", "Revert changes")), true, m.Busy || stale || !revert.Allowed || revert.Confirm == nil).Clicked() {
						revert.Confirm()
					}
				})
			})
		})
	})
	if !open {
		close()
	}
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
