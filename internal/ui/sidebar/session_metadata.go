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
		Width(16).Height(20).Margin(0, 0, 0, 6).Draw(func(p *ui.Painter, r ui.Rect) {
		path := &ui.Path{}
		path.MoveTo(r.X+6, r.Y+3)
		path.LineTo(r.X+12, r.Y+3)
		path.LineTo(r.X+11, r.Y+8)
		path.LineTo(r.X+13, r.Y+11)
		path.LineTo(r.X+5, r.Y+11)
		path.LineTo(r.X+7, r.Y+8)
		path.Close()
		p.FillPath(path, ink)
		needle := &ui.Path{}
		needle.MoveTo(r.X+9, r.Y+11)
		needle.LineTo(r.X+9, r.Y+17)
		p.StrokePath(needle, 1.2, ink)
	})
}
