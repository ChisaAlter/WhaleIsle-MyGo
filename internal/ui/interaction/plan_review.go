package interaction

import (
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/interaction"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/planpreview"
	"whaleisle/internal/ui/widgets"
)

// planDiscussIcon is the original's IconEditOutlineRegular at size 14
// (ui-primitives icons/artwork.ts 'edit-outline'), the leading glyph the
// original seats inside the Request changes button.
// planReviewChevron is IconChevronRightOutlineRegular, the trailing glyph on
// the original's .reviewLink (ui-plan PlanPreview.module.css).
var planReviewChevron = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke-width="1"><path d="M6 12L9.29289 8.70711C9.68342 8.31658 9.68342 7.68342 9.29289 7.29289L6 4" stroke="currentColor"/></svg>`))

var planDiscussIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none"><path d="M8.85596 2.69971H4.19971C3.37141 2.69971 2.69992 3.37146 2.69971 4.19971V11.8003C2.69992 12.6285 3.37141 13.3003 4.19971 13.3003H11.8003C12.6283 13.2999 13.3001 12.6283 13.3003 11.8003V7.89893H14.3003V11.8003C14.3001 13.1806 13.1806 14.2999 11.8003 14.3003H4.19971C2.81913 14.3003 1.69992 13.1808 1.69971 11.8003V4.19971C1.69992 2.81918 2.81913 1.69971 4.19971 1.69971H8.85596V2.69971Z" fill="currentColor"/><path d="M7.7849 8.23878L13.888 2.13574" stroke="currentColor" stroke-width="1"/></svg>`))

// planReviewCard deliberately replaces generic Continue/Deny. Discussion only
// dismisses the panel; approval is sent exclusively by the explicit decision.
func planReviewCard(c *ui.Context, pending interaction.Pending, card interaction.QuestionCardView, store CardStore, colors fonttheme.Colors, fg, muted, accent ui.Color, onOpen func(planpreview.Document)) {
	review := card.PlanReview
	if review == nil {
		return
	}
	seat := chrome.FrostSurface(ui.Column(c.Key("plan-review:"+pending.AgentID+":"+pending.EventID)).Label(review.Question).FillWidth().MinWidth(0).Gap(0).Clip(), colors, c.Theme().Dark, c.Theme().Surface, 20)
	summary := ui.Local(seat, "plan-summary", func() planReviewSummary { return planReviewSummary{} })
	summary.resolve(review.Plan)
	openPlan := func() {
		address := planpreview.ReviewAddress(pending.AgentID, pending.EventID)
		if review.CallID != "" {
			address = planpreview.PlanAddress(pending.AgentID, review.CallID)
		}
		onOpen(planpreview.Document{Address: address, Title: summary.title, Markdown: review.Plan})
	}
	if !card.PlanReviewHidden && onOpen != nil && store.ConsumePlanReviewOpen(pending.AgentID, pending.EventID) {
		openPlan()
	}
	seat.Children(func() {
		// PlanReviewPanel.module.css .strip: warn-tertiary band, 12px/16px
		// padding, 14px/20 warn-primary label — the same tinted header the
		// approval takeover uses.
		ui.Row(c).FillWidth().Shrink(0).Padding(12, 16).Background(c.Theme().Warning.Alpha(.12)).Gap(8).AlignItems(ui.Center).Label("Plan review strip").Children(func() {
			state := widgets.StateWarning
			if pending.Sending {
				state = widgets.StateRunning
			}
			widgets.StateDot(c, state)
			ui.Text(c, planpreview.Text("计划待审", "Plan review")).FontSize(fonttheme.InterfaceSize(14)).FixedLineHeight(fonttheme.InterfaceSize(20)).TextColor(c.Theme().Warning).Grow(1)
			if card.PlanReviewHidden {
				if widgets.Button(c, colors, planpreview.Text("返回审批", "Return to review"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Clicked() {
					store.ReopenPlanReview(pending.EventID)
				}
			} else if onOpen != nil {
				// PlanPreview.module.css .reviewLink: a borderless inline
				// link at the strip's 14/20, label-secondary until hover
				// lifts it to label-primary; the chevron is
				// IconChevronRightOutlineRegular at 14px with a 4px gap.
				// The original's aria-label/title carry preview.open.
				link := coss.Button(c, "", coss.ButtonOptions{
					Variant: coss.ButtonGhost,
					Size:    coss.ButtonXS,
					Label:   planpreview.Text("在侧边栏打开计划", "Open plan in sidebar"),
					Content: func() {
						ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
							ui.Text(c, planpreview.Text("查看全文", "View full plan")).FontSize(fonttheme.InterfaceSize(14)).FixedLineHeight(fonttheme.InterfaceSize(20))
							ui.Icon(c, planReviewChevron).Size(14, 14).Role(ui.RoleNone)
						})
					},
				}).Tooltip(planpreview.Text("在侧边栏打开计划", "Open plan in sidebar")).Padding(0).TextColor(muted)
				if link.Hovered() || link.Pressed() {
					link.TextColor(fg)
				}
				if link.Clicked() {
					openPlan()
				}
			}
		})
		if card.PlanReviewHidden {
			return
		}
		// .summary { padding: 14px 16px 12px }: 15/22 single-line title, then a
		// 14/24 secondary two-line description with an 8px gap.
		ui.Column(c).FillWidth().MinWidth(0).Padding(14, 16, 12, 16).Gap(8).Label("Plan review summary").Children(func() {
			ui.Text(c, summary.title).FillWidth().MinWidth(0).FontSize(fonttheme.InterfaceSize(15)).FixedLineHeight(fonttheme.InterfaceSize(22)).FontWeight(500).SingleLine().TextColor(fg)
			if summary.description != "" {
				ui.Text(c, summary.description).FillWidth().MinWidth(0).FontSize(fonttheme.InterfaceSize(14)).FixedLineHeight(fonttheme.InterfaceSize(24)).MaxHeight(fonttheme.InterfaceSize(48)).Clip().TextColor(muted)
			}
		})
		if pending.Error != nil {
			ui.Text(c, questionErrorText(pending.Error)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		}
		// .footer { padding: 8px 16px 12px }: the original seats the same
		// medium outline/primary pair as the approval card.
		ui.Row(c).FillWidth().Shrink(0).Padding(8, 16, 12, 16).Gap(8).Justify(ui.End).Wrap().Label("Plan review actions").Children(func() {
			if widgets.Button(c, colors, planpreview.Text("要求修改", "Request changes"), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonMedium, Disabled: pending.Sending, Leading: func(cc *ui.Context) {
				ui.Icon(cc, planDiscussIcon).Width(14).Height(14).TextColor(fg)
			}}).Clicked() {
				store.DismissPlanReview(pending.EventID)
			}
			label := planpreview.Text("同意执行", "Approve")
			if pending.Sending {
				label = planpreview.Text("正在提交…", "Submitting…")
			}
			if widgets.Button(c, colors, label, widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonMedium, Disabled: pending.Sending}).Tooltip(review.Approve.Description).Clicked() {
				store.ApprovePlanReview(pending.EventID)
			}
			if pending.Error != nil && widgets.Button(c, colors, planpreview.Text("重试上次操作", "Retry last action"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonMedium, Disabled: pending.Sending}).Clicked() {
				store.Retry(pending.EventID)
			}
		})
	})
}
