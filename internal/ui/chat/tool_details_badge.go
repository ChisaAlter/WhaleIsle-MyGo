package chat

import (
	"strconv"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
)

// toolDetailsModelStatusKeys is the installed STATUS_KEYS vocabulary. Only an
// own, known key is localized; anything else keeps its recorded spelling.
var toolDetailsModelStatusKeys = map[string]string{
	"running":      i18n.KeyToolDetailStatusRunning,
	"idle":         i18n.KeyToolDetailStatusIdle,
	"ready":        i18n.KeyToolDetailStatusReady,
	"inactive":     i18n.KeyToolDetailStatusInactive,
	"provisioning": i18n.KeyToolDetailStatusProvisioning,
	"failed":       i18n.KeyToolDetailStatusFailed,
	"error":        i18n.KeyToolDetailStatusFailed,
	"completed":    i18n.KeyToolDetailStatusCompleted,
	"complete":     i18n.KeyToolDetailStatusCompleted,
	"done":         i18n.KeyToolDetailStatusCompleted,
	"pending":      i18n.KeyToolDetailStatusPending,
	"in_progress":  i18n.KeyToolDetailStatusProgress,
	"deleted":      i18n.KeyToolDetailStatusDeleted,
	"killed":       i18n.KeyToolDetailStatusKilled,
	"blocked":      i18n.KeyToolDetailStatusBlocked,
	"accepted":     i18n.KeyToolDetailStatusAccepted,
	"queued":       i18n.KeyToolDetailStatusQueued,
}

// toolDetailsModelBadge mirrors the installed detailBadge: a localized known
// status and the same static semantic tone, with unknown statuses verbatim.
func toolDetailsModelBadge(status string) toolDetailBadge {
	label := status
	if key, known := toolDetailsModelStatusKeys[status]; known {
		label = i18n.T(key)
	}
	tone := "neutral"
	switch status {
	case "completed", "complete", "done", "accepted":
		tone = "success"
	case "failed", "error":
		tone = "error"
	case "blocked", "killed", "pending", "queued", "inactive":
		tone = "warning"
	case "running", "in_progress", "provisioning":
		tone = "info"
	}
	return toolDetailBadge{Label: label, Tone: tone}
}

// toolDetailsBadgeKey is the stable key of one item's status badge row.
func toolDetailsBadgeKey(index int) string {
	return "tool-details-badge:" + strconv.Itoa(index)
}

// toolDetailsBadgeInk maps the static tone to the card palette. The installed
// warn token is approximated by the tool-card warning ink already used for
// status summaries; business/info uses the card accent.
func toolDetailsBadgeInk(tone string, st ToolCardStyle, caption ui.Color) ui.Color {
	switch tone {
	case "success":
		return st.OK
	case "error":
		return st.Err
	case "warning":
		return ui.Hex("#dd8629")
	case "info":
		return st.Accent
	default:
		return caption
	}
}

// toolDetailsBadgeView is the installed inline status badge: a 5px round dot, a
// 5px gap and the 12px label, sized and aligned with the heading's first line.
func toolDetailsBadgeView(c *ui.Context, badge toolDetailBadge, st ToolCardStyle, caption ui.Color, font, key string) {
	ink := toolDetailsBadgeInk(badge.Tone, st, caption)
	ui.Row(c.Key(key)).Shrink(0).Gap(5).AlignItems(ui.Center).Label(badge.Label).Children(func() {
		ui.Box(c.Key("dot")).Size(5, 5).Radius(999).Background(ink).Shrink(0)
		ui.Text(c, badge.Label).Font(font).FontSize(12).FixedLineHeight(20).TextColor(ink).MinWidth(0)
	})
}
