package interaction

import (
	"strings"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/interaction"
	"whaleisle/internal/ui/planpreview"
)

// planReviewCard deliberately replaces generic Continue/Deny. Discussion only
// dismisses the panel; approval is sent exclusively by the explicit decision.
func planReviewCard(c *ui.Context, pending interaction.Pending, card interaction.QuestionCardView, store *interaction.Store, fg, muted, accent ui.Color, onOpen func(planpreview.Document)) {
	review := card.PlanReview
	if review == nil {
		return
	}
	ui.Column(c.Key("plan-review:" + pending.AgentID + ":" + pending.EventID)).FillWidth().Padding(8).Gap(6).Background(accent.Alpha(0.08)).Radius(12).Children(func() {
		ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, planpreview.Text("计划审批", "Plan review")).Bold().FontSize(fonttheme.InterfaceSize(13)).TextColor(fg).Grow(1)
			if card.PlanReviewHidden {
				if ui.Button(c, planpreview.Text("返回审批", "Return to review")).Clicked() {
					store.ReopenPlanReview(pending.EventID)
				}
			}
		})
		if card.PlanReviewHidden {
			return
		}
		if review.Question != "" {
			ui.Text(c, review.Question).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
		}
		if pending.Continued {
			ui.Text(c, questionStatusText("continued")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		title := strings.SplitN(strings.TrimSpace(review.Plan), "\n", 2)[0]
		title = strings.TrimSpace(strings.TrimPrefix(title, "#"))
		if title == "" {
			title = planpreview.Text("提交的计划", "Submitted plan")
		}
		if onOpen != nil && ui.Button(c, planpreview.Text("在右侧打开计划", "Open plan in sidebar")).FontSize(fonttheme.InterfaceSize(11)).Clicked() {
			address := planpreview.ReviewAddress(pending.AgentID, pending.EventID)
			if review.CallID != "" {
				address = planpreview.PlanAddress(pending.AgentID, review.CallID)
			}
			onOpen(planpreview.Document{Address: address, Title: title, Markdown: review.Plan})
		}
		planpreview.View(c, "review-document:"+pending.AgentID+":"+pending.EventID, title, review.Plan, true, planpreview.Style{FG: fg, Muted: muted, Accent: accent, Field: accent.Alpha(0.03), Border: accent.Alpha(0.18)})
		if pending.Error != "" {
			ui.Text(c, pending.Error).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		}
		if review.Approve.Description != "" {
			ui.Text(c, review.Approve.Description).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		ui.Row(c).Gap(8).Children(func() {
			if ui.Button(c, planpreview.Text("讨论修改", "Discuss changes")).Disabled(pending.Sending).Clicked() {
				store.DismissPlanReview(pending.EventID)
			}
			label := planpreview.Text("批准计划", "Approve plan")
			if pending.Sending {
				label = planpreview.Text("正在提交…", "Submitting…")
			}
			if ui.Button(c, label).Tooltip(review.Approve.Label).Disabled(pending.Sending).Clicked() {
				store.ApprovePlanReview(pending.EventID)
			}
			if pending.Error != "" && ui.Button(c, planpreview.Text("重试上次操作", "Retry last action")).Disabled(pending.Sending).Clicked() {
				store.Retry(pending.EventID)
			}
		})
	})
}
