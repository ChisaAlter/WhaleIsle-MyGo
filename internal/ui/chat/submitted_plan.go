package chat

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/planpreview"
)

// SubmittedPlanCardView remains readable after approval, dismissal or tool
// failure. The document is the original invocation, not the current plan mode.
func SubmittedPlanCardView(c *ui.Context, message session.Message, sessionID string, style planpreview.Style) {
	plan, ok := session.SubmittedPlanFromEvent(message.Raw)
	if !ok {
		return
	}
	planpreview.View(c, "plan:"+sessionID+":"+plan.CallID, plan.Title, plan.Markdown, false, style)
}
