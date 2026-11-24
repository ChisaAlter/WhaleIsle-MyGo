package chat

import (
	"strconv"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/design"
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

func toolDetailsBadgeInkFor(c *ui.Context, tone string, caption ui.Color) ui.Color {
	t := design.For(c)
	switch tone {
	case "success":
		return t.SuccessForeground
	case "error":
		return t.DestructiveForeground
	case "warning":
		return t.WarningForeground
	case "info":
		return t.InfoForeground
	default:
		if caption.A != 0 {
			return caption
		}
		return t.MutedForeground
	}
}

// toolDetailsBadgeView is the installed inline status badge: a 5px round dot, a
// 5px gap and the 12px label, sized and aligned with the heading's first line.
func toolDetailsBadgeView(c *ui.Context, badge toolDetailBadge, _ ToolCardStyle, caption ui.Color, font, key string) {
	ink := toolDetailsBadgeInkFor(c, badge.Tone, caption)
	ui.Row(c.Key(key)).Shrink(0).Gap(design.Sp(1.25)).AlignItems(ui.Center).Label(badge.Label).Children(func() {
		ui.Box(c.Key("dot")).Size(5, 5).Radius(design.RadiusFull).Background(ink).Shrink(0)
		ui.Text(c, badge.Label).Font(font).FontSize(design.TextXS.Size).FixedLineHeight(20).TextColor(ink).MinWidth(0)
	})
}
