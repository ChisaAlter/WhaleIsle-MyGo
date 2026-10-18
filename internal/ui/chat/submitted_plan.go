package chat

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/planpreview"
)

// SubmittedPlanCardView remains readable after approval, dismissal or tool
// failure. The document is the original invocation, not the current plan mode.
func SubmittedPlanCardView(c *ui.Context, message session.Message, sessionID string, style planpreview.Style, onOpen ...func(planpreview.Document)) {
	plan, ok := session.SubmittedPlanFromEvent(message.Raw)
	if !ok {
		return
	}
	if len(onOpen) > 0 && onOpen[0] != nil && ui.Button(c, planpreview.Text("在右侧打开计划", "Open plan in sidebar")).FontSize(11).Clicked() {
		onOpen[0](planpreview.Document{Address: planpreview.PlanAddress(sessionID, plan.CallID), Title: plan.Title, Markdown: plan.Markdown})
	}
	planpreview.View(c, "plan:"+sessionID+":"+plan.CallID, plan.Title, plan.Markdown, false, style)
}
