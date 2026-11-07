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
func planReviewCard(c *ui.Context, pending interaction.Pending, card interaction.QuestionCardView, store CardStore, colors fonttheme.Colors, fg, muted, accent ui.Color, onOpen func(planpreview.Document), renderers ...SlotRenderer) {
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
		ui.Row(c).FillWidth().Shrink(0).Padding(12, 16).Background(c.Theme().Warning.Alpha(.12)).Gap(8).AlignItems(ui.Center).Children(func() {
			state := widgets.StateWarning
			if pending.Sending {
				state = widgets.StateRunning
			}
			widgets.StateDot(c, state)
			ui.Text(c, planpreview.Text("计划审批", "Plan review")).Bold().FontSize(fonttheme.InterfaceSize(13)).TextColor(fg).Grow(1)
			if card.PlanReviewHidden {
				if widgets.Button(c, colors, planpreview.Text("返回审批", "Return to review"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Clicked() {
					store.ReopenPlanReview(pending.EventID)
				}
			} else {
				renderInteractionSlot(c, pending, "conversation.plan-review.actions", func() {
					if onOpen != nil {
						if widgets.Button(c, colors, planpreview.Text("预览计划", "Preview plan"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Clicked() {
							openPlan()
						}
					}
				}, renderers...)
			}
		})
		if card.PlanReviewHidden {
			return
		}
		ui.Column(c).FillWidth().MinWidth(0).Padding(14, 16, 12, 16).Gap(8).Children(func() {
			ui.Text(c, summary.title).FillWidth().MinWidth(0).FontSize(fonttheme.InterfaceSize(15)).FixedLineHeight(fonttheme.InterfaceSize(22)).FontWeight(500).SingleLine().TextColor(fg)
			if summary.description != "" {
				ui.Text(c, summary.description).FillWidth().MinWidth(0).FontSize(fonttheme.InterfaceSize(14)).FixedLineHeight(fonttheme.InterfaceSize(24)).MaxHeight(fonttheme.InterfaceSize(48)).Clip().TextColor(muted)
			}
		})
		if pending.Error != "" {
			ui.Text(c, pending.Error).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		}
		ui.Row(c).FillWidth().Shrink(0).Padding(8, 16, 12, 16).Gap(8).Justify(ui.End).Wrap().Children(func() {
			if widgets.Button(c, colors, planpreview.Text("要求修改", "Request changes"), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall, Disabled: pending.Sending}).Clicked() {
				store.DismissPlanReview(pending.EventID)
			}
			label := planpreview.Text("同意执行", "Approve and run")
			if pending.Sending {
				label = planpreview.Text("正在提交…", "Submitting…")
			}
			if widgets.Button(c, colors, label, widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonSmall, Disabled: pending.Sending}).Tooltip(review.Approve.Description).Clicked() {
				store.ApprovePlanReview(pending.EventID)
			}
			if pending.Error != "" && widgets.Button(c, colors, planpreview.Text("重试上次操作", "Retry last action"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall, Disabled: pending.Sending}).Clicked() {
				store.Retry(pending.EventID)
			}
		})
	})
}
