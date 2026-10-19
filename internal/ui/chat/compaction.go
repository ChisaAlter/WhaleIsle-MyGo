package chat

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/widgets/markdown"
)

// CompactionView keeps a landed automatic checkpoint compact until the user
// opens its cited summary. Missing history evidence cannot open a disclosure.
func CompactionView(c *ui.Context, msg session.Message, fg, muted, field ui.Color) {
	root := ui.Column(c).Gap(6)
	open := ui.Local(root, "summary-open", func() bool { return false })
	compact := msg.AutomaticCompaction
	if compact == nil {
		root.Background(field).Radius(6).Padding(8, 10).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyChatCompaction)).FontSize(11).Bold().TextColor(muted)
			ui.Text(c, msg.Body).FontSize(12).TextColor(fg)
		})
		return
	}
	expandable := compact.Summary != ""
	label := i18n.T(i18n.KeyChatCompactionSummaryMissing)
	if compact.ShadowedItemCount != nil && compact.ShadowedTokenCount != nil {
		label = fmt.Sprintf(i18n.T(i18n.KeyChatCompactionCounts), *compact.ShadowedItemCount, *compact.ShadowedTokenCount)
	} else if expandable {
		label = i18n.T(i18n.KeyPlanBoardExpand)
	}
	root.Children(func() {
		button := ui.Button(c, "").Label(i18n.T(i18n.KeyChatCompaction)).Disabled(!expandable).
			Background(ui.Transparent).BorderWidth(0).Padding(4, 0).Gap(6).AlignItems(ui.Center)
		button.Children(func() {
			chevron := "▸"
			if expandable && *open {
				chevron = "▾"
			}
			ui.Text(c, chevron).FontSize(12).TextColor(muted)
			ui.Text(c, i18n.T(i18n.KeyChatCompaction)).FontSize(12).TextColor(muted)
			ui.Text(c, "· "+label).FontSize(12).TextColor(muted)
		})
		if button.Clicked() {
			*open = !*open
		}
		if expandable && *open {
			markdown.View(c, compact.Summary, markdown.Style{FG: fg, Muted: muted, CodeBG: field, Accent: c.Theme().Accent})
		}
	})
}
