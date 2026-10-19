package sidebar

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
)

func sessionTrailingLabel(status string, updatedAt float64) string {
	switch status {
	case "approval":
		return workspaceText("待审批", "Approval")
	case "plan-review":
		return workspaceText("计划待审", "Plan review")
	case "question":
		return workspaceText("待回答", "Answer")
	}
	return sessionRelativeTime(updatedAt, float64(time.Now().UnixMilli()))
}

// Original ui-primitives/relative-time.ts uses epoch milliseconds and these
// exact minute/day/month boundaries; row copy stays owned by the locale.
func sessionRelativeTime(at, now float64) string {
	diff := max(float64(0), now-at)
	for _, bucket := range []struct {
		end, unit float64
		zh, en    string
	}{
		{60_000, 0, "刚刚", "now"},
		{3_600_000, 60_000, "%d分钟", "%dmin"},
		{86_400_000, 3_600_000, "%d小时", "%dh"},
		{30 * 86_400_000, 86_400_000, "%d天", "%dd"},
		{365 * 86_400_000, 30 * 86_400_000, "%d个月", "%dmo"},
		{0, 365 * 86_400_000, "%d年", "%dy"},
	} {
		if bucket.end == 0 || diff < bucket.end {
			label := workspaceText(bucket.zh, bucket.en)
			if bucket.unit == 0 {
				return label
			}
			return fmt.Sprintf(label, int64(diff/bucket.unit))
		}
	}
	return ""
}

func sessionPinIndicator(c *ui.Context, id string, ink ui.Color) {
	ui.Box(c.Key("session-pin")).Label("Session pinned "+id).Tooltip(workspaceText("已置顶", "Pinned")).
		Width(16).Height(20).Margin(0, 0, 0, 6).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
		ui.Icon(c, sessionPinArtwork).Width(14).Height(14).TextColor(ink)
	})
}

// Exact IconPinFillRegular paths from the original ui-primitives/icons/index.tsx.
var sessionPinArtwork = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" stroke-width="1">
<path d="M9.96976 1.70572L13.1554 3.93629L10.9019 8.12317L11.5158 11.605L10.7192 12.7427L2.52767 7.00693L3.3243 5.86922L6.80612 5.25528L9.96976 1.70572Z" fill="currentColor" stroke="currentColor" stroke-linejoin="round"/>
<path d="M6.05285 9.47511C6.27284 9.16094 6.70586 9.08458 7.02003 9.30457C7.3342 9.52455 7.41055 9.95757 7.19057 10.2717L3.98587 14.4708L3.21223 13.9291L6.05285 9.47511Z" fill="currentColor"/>
</svg>`))
