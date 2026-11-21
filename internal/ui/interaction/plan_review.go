package interaction

import (
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/interaction"
	"whaleisle/internal/ui/planpreview"
	"whaleisle/internal/ui/widgets"
)

// planReviewCard deliberately replaces generic Continue/Deny. Discussion only
// dismisses the panel; approval is sent exclusively by the explicit decision.
func planReviewCard(c *ui.Context, pending interaction.Pending, card interaction.QuestionCardView, store CardStore, colors fonttheme.Colors, fg, muted, accent ui.Color, onOpen func(planpreview.Document)) {
	review := card.PlanReview
	if review == nil {
		return
	}
	seat := ui.Column(c.Key("plan-review:" + pending.AgentID + ":" + pending.EventID)).Label(review.Question).FillWidth().MinWidth(0).Gap(0).Background(c.Theme().Surface).Radius(20).Clip()
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
			ui.Text(c, planpreview.Text("计划待审", "Plan review")).FontSize(14).FixedLineHeight(20).TextColor(c.Theme().Warning).Grow(1)
			if card.PlanReviewHidden {
				if widgets.Button(c, colors, planpreview.Text("返回审批", "Return to review"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Clicked() {
					store.ReopenPlanReview(pending.EventID)
				}
			} else if onOpen != nil {
				if widgets.Button(c, colors, planpreview.Text("预览计划", "Preview plan"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Clicked() {
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
		if pending.Error != "" {
			ui.Text(c, questionErrorText(pending.Error)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		}
		// .footer { padding: 8px 16px 12px }: the original seats the same
		// medium outline/primary pair as the approval card.
		ui.Row(c).FillWidth().Shrink(0).Padding(8, 16, 12, 16).Gap(8).Justify(ui.End).Wrap().Label("Plan review actions").Children(func() {
			if widgets.Button(c, colors, planpreview.Text("要求修改", "Request changes"), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonMedium, Disabled: pending.Sending}).Clicked() {
				store.DismissPlanReview(pending.EventID)
			}
			label := planpreview.Text("同意执行", "Approve")
			if pending.Sending {
				label = planpreview.Text("正在提交…", "Submitting…")
			}
			if widgets.Button(c, colors, label, widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonMedium, Disabled: pending.Sending}).Tooltip(review.Approve.Description).Clicked() {
				store.ApprovePlanReview(pending.EventID)
			}
			if pending.Error != "" && widgets.Button(c, colors, planpreview.Text("重试上次操作", "Retry last action"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonMedium, Disabled: pending.Sending}).Clicked() {
				store.Retry(pending.EventID)
			}
		})
	})
}
