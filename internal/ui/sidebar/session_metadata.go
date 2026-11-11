package sidebar

import (
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
)

func sessionTrailingLabel(status string, updatedAt float64) string {
	switch status {
	case "approval":
		return i18n.T(i18n.KeyWSStatusCompactApproval)
	case "plan-review":
		return i18n.T(i18n.KeyWSStatusCompactPlanReview)
	case "question":
		return i18n.T(i18n.KeyWSStatusCompactAnswer)
	}
	return sessionRelativeTime(updatedAt, float64(time.Now().UnixMilli()))
}

// Original ui-primitives/relative-time.ts uses epoch milliseconds and these
// exact minute/day/month boundaries; row copy stays owned by the locale.
func sessionRelativeTime(at, now float64) string {
	diff := max(float64(0), now-at)
	for _, bucket := range []struct {
		end, unit float64
		key       string
	}{
		{60_000, 0, i18n.KeyWSTimeNow},
		{3_600_000, 60_000, i18n.KeyWSTimeMinutes},
		{86_400_000, 3_600_000, i18n.KeyWSTimeHours},
		{30 * 86_400_000, 86_400_000, i18n.KeyWSTimeDays},
		{365 * 86_400_000, 30 * 86_400_000, i18n.KeyWSTimeMonths},
		{0, 365 * 86_400_000, i18n.KeyWSTimeYears},
	} {
		if bucket.end == 0 || diff < bucket.end {
			if bucket.unit == 0 {
				return i18n.T(bucket.key)
			}
			return i18n.WSf(bucket.key, "n", strconv.FormatInt(int64(diff/bucket.unit), 10))
		}
	}
	return ""
}

func sessionPinIndicator(c *ui.Context, id string, ink ui.Color) {
	ui.Box(c.Key("session-pin")).Label("Session pinned "+id).Tooltip(i18n.T(i18n.KeyWSRowPinned)).
		Width(16).Height(20).Margin(0, 0, 0, 6).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
		ui.Icon(c, sessionPinArtwork).Width(14).Height(14).TextColor(ink)
	})
}

// Exact IconPinFillRegular paths from the original ui-primitives/icons/index.tsx.
var sessionPinArtwork = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" stroke-width="1">
<path d="M9.96976 1.70572L13.1554 3.93629L10.9019 8.12317L11.5158 11.605L10.7192 12.7427L2.52767 7.00693L3.3243 5.86922L6.80612 5.25528L9.96976 1.70572Z" fill="currentColor" stroke="currentColor" stroke-linejoin="round"/>
<path d="M6.05285 9.47511C6.27284 9.16094 6.70586 9.08458 7.02003 9.30457C7.3342 9.52455 7.41055 9.95757 7.19057 10.2717L3.98587 14.4708L3.21223 13.9291L6.05285 9.47511Z" fill="currentColor"/>
</svg>`))
