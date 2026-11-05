package chat

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// QueueDockView paints the compact queue dock that belongs above the composer:
// one row per queued prompt with its preview and icon-only actions, plus the
// inline queue editor. The expanded plan board renders the same rows from
// paintQueueRows, so the two surfaces never drift.
//
// The frame owns the seat; it only needs to call this when the dock has rows:
//
//	if snap.PlanBoard.HasQueue() {
//		chat.QueueDockView(c, chat.PlanBoardModel{...})
//	}
func QueueDockView(c *ui.Context, m PlanBoardModel) {
	if !queueDockHasRows(m.Board) && m.QueueEditor.ItemID == "" {
		return
	}
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	border := chrome.Color(colors.Border)
	field := chrome.Color(colors.Field)
	accent := chrome.Color(colors.Accent)
	root := ui.Column(c).Key("queue-dock").FillWidth().MinWidth(0).Gap(6).
		Background(field).Border(0.5, border).Radius(12).Padding(8, 10, 8, 10)
	collapsed := ui.Local(root, "collapsed", func() bool { return true })
	root.Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyChatQueueTitle)).FontSize(11).TextColor(muted).Grow(1)
			if n := len(m.Board.Queue); n > 0 {
				ui.Text(c, queueCountLabel(n)).FontSize(11).TextColor(muted)
			}
			if len(m.Board.Queue)+len(m.Board.Pending) > 1 {
				label := "▴"
				if *collapsed {
					label = "▾"
				}
				if ui.Button(c, label).Label(trajectoryText("展开或收起队列", "Expand or collapse queue")).Disabled(m.QueueEditor.ItemID != "").Clicked() {
					*collapsed = !*collapsed
				}
			}
		})
		if len(m.Board.Queue)+len(m.Board.Pending) > 1 && *collapsed && m.QueueEditor.ItemID == "" {
			return
		}
		paintQueueEditor(c, m, fg, muted)
		paintQueueRows(c, m, fg, muted, accent, border, true)
	})
}

// queueDockHasRows reports whether the dock has anything worth mounting.
func queueDockHasRows(board session.PlanBoard) bool {
	return len(board.Queue) > 0 || len(board.Pending) > 0
}

func queueCountLabel(n int) string {
	if i18n.Locale() == i18n.LocaleZH {
		return queueCountNumber(n) + " 条排队消息"
	}
	if n == 1 {
		return "1 queued"
	}
	return queueCountNumber(n) + " items"
}

func queueCountNumber(n int) string {
	if n == 0 {
		return "0"
	}
	digits := make([]byte, 0, 6)
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// paintQueueRows is the shared row list: pending submissions first, then queued
// prompts. Compact rows keep the preview on one line and put the actions in
// icon seats; the expanded board keeps the same order with more room.
func paintQueueRows(c *ui.Context, m PlanBoardModel, fg, muted, accent, border ui.Color, compact bool) {
	for _, pending := range m.Board.Pending {
		paintPendingSubmission(c, pending, fg, muted)
	}
	for _, q := range m.Board.Queue {
		item := q
		row := ui.Row(c).Key("queue-row:"+item.ID).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).
			Padding(4, 6, 4, 6).Radius(8)
		row.Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).Gap(3).Children(func() {
				paintQueueAttachments(c, m, item, fg, muted)
				line := item.Preview
				if item.Source != "" && !compact {
					line = item.Source + ": " + line
				}
				if strings.TrimSpace(line) == "" {
					line = i18n.T(i18n.KeyChatQueueEmptyPreview)
				}
				ui.Text(c, maskedUserText(line)).FontSize(theme.InterfaceSize(13)).FixedLineHeight(20).TextColor(fg).SingleLine().MinWidth(0)
			})
			action := func(icon *ui.SVG, label string, enabled bool, tone ui.Color, run func()) {
				if run == nil {
					return
				}
				button := ui.Button(c, "").Label(label).Tooltip(label).Width(24).Height(24).Radius(6).
					Padding(0, 0, 0, 0).Background(ui.Transparent).BorderWidth(0)
				button.Children(func() { ui.Icon(c, icon).Width(13).Height(13).TextColor(tone) })
				if !enabled {
					button.Disabled(true).Opacity(.35)
					return
				}
				if button.Hovered() {
					button.Background(tone.Alpha(.12))
				}
				if button.Clicked() {
					run()
				}
			}
			can := m.Board.QueueActions
			if m.OnQueueEdit != nil && item.ID != "" {
				action(messageIconEditOutline, i18n.T(i18n.KeyPlanBoardQueueEdit), can.Can(session.QueueActionEdit, item), muted, func() {
					m.OnQueueEdit(item.ID, item.Preview)
				})
			}
			if m.OnQueueSteer != nil && item.ID != "" {
				action(queueIconSteer, i18n.T(i18n.KeyPlanBoardQueueSteer), can.Can(session.QueueActionSteer, item), accent, func() {
					m.OnQueueSteer(item.ID)
				})
			}
			if m.OnQueueRemove != nil && item.ID != "" {
				action(messageIconCloseOutline, i18n.T(i18n.KeyPlanBoardQueueRemove), can.Can(session.QueueActionRemove, item), muted, func() {
					m.OnQueueRemove(item.ID)
				})
			}
		})
	}
}
