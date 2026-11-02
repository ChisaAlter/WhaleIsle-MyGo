package chat

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// The Original keeps a 28 DIP turn rail floating on the transcript's trailing
// edge: one tick per turn, the current one filled, a preview on hover and a
// jump on click. Below that width the rail folds into a menu button.
const (
	turnRailWidth      = float32(28)
	turnRailNarrowMin  = float32(720)
	turnRailTickWidth  = float32(14)
	turnRailSeatHeight = float32(10)
)

// turnAnchor is one jump target on the rail.
type turnAnchor struct {
	Seq     float64
	Index   int
	Preview string
	Status  rowStatus
	Loaded  bool
}

// turnAnchors lists the durable user prompts of a transcript: each one opens a
// turn, which is what the rail counts and what a jump scrolls to.
func turnAnchors(messages []session.Message) []turnAnchor {
	var out []turnAnchor
	for _, msg := range messages {
		if msg.Role != "user" {
			continue
		}
		if msg.Kind != session.MessageKindText && msg.Kind != session.MessageKindSubmission {
			continue
		}
		preview := messagePreviewText(msg.Body, 72)
		if preview == "" && msg.Submission != nil {
			preview = messagePreviewText(msg.Submission.Text, 72)
		}
		if preview == "" {
			preview = i18n.T(i18n.KeyChatTurnRailEmpty)
		}
		out = append(out, turnAnchor{Seq: msg.Seq, Index: len(out) + 1, Preview: preview, Loaded: true})
	}
	return out
}

func turnAnchorsFromSnapshot(snap session.ViewState) []turnAnchor {
	loaded := turnAnchors(snap.Messages)
	if len(snap.HistoryNavigation.Turns) <= len(loaded) {
		return loaded
	}
	out := make([]turnAnchor, 0, len(snap.HistoryNavigation.Turns))
	for i, t := range snap.HistoryNavigation.Turns {
		preview := messagePreviewText(t.Prompt, 72)
		if preview == "" {
			preview = i18n.T(i18n.KeyChatTurnRailEmpty)
		}
		out = append(out, turnAnchor{
			Seq:     t.Seq,
			Index:   i + 1,
			Preview: preview,
			Loaded:  t.Loaded,
		})
	}
	return out
}

// messagePreviewText flattens a body for a one-line preview: image placeholders
// drop out, whitespace collapses and the tail is elided.
func messagePreviewText(body string, limit int) string {
	refs, text := userMessageParts(strings.TrimSpace(body))
	_ = refs
	flat := strings.Join(strings.Fields(text), " ")
	runes := []rune(flat)
	if len(runes) <= limit {
		return flat
	}
	return string(runes[:limit]) + "…"
}

// TurnRailView paints the turn rail when the transcript is wide enough and a
// compact turn menu below that. available is the transcript's width; onJump is
// told the target message sequence and may be nil for a read-only surface.
func TurnRailView(c *ui.Context, m Model, available float32, onJump func(seq float64)) {
	anchors := turnAnchorsFromSnapshot(m.Snapshot)
	if len(anchors) < 2 {
		return
	}
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	accent := chrome.Color(colors.Accent)
	if available < turnRailNarrowMin {
		turnRailMenu(c, anchors, m, muted, accent, onJump)
		return
	}
	current := len(anchors) - 1
	rail := ui.Column(c).Key("turn-rail").Width(turnRailWidth).Gap(0).AlignItems(ui.End).Padding(6, 0, 6, 0)
	rail.Children(func() {
		for index, anchor := range anchors {
			anchor := anchor
			active := index == current
			// The seat is the hit target: a 12 DIP row holding the 2 DIP tick, so
			// a multi-turn transcript gives each tick a comfortable right-aligned target.
			seat := ui.Row(c).Key("turn-rail-tick:"+itoaFast(anchor.Index)).
				Width(turnRailWidth).Height(turnRailSeatHeight).Padding(0, 4, 0, 4).Justify(ui.End).AlignItems(ui.Center)
			interactive := onJump != nil
			if interactive {
				seat.Label(i18n.T(i18n.KeyChatTurnRailJump) + " " + itoaFast(anchor.Index)).Focusable().Role(ui.RoleButton)
			}
			hovered := interactive && seat.Hovered()
			barWidth := float32(12)
			barHeight := float32(2)
			barColor := muted.Alpha(.38)
			if hovered {
				barWidth = 16
				barColor = fg.Alpha(.72)
			}
			if active {
				barWidth = 20
				barColor = fg
			}
			seat.Children(func() {
				ui.Box(c).Width(barWidth).Height(barHeight).Radius(999).Background(barColor)
			})
			seat.Tooltip(i18n.T(i18n.KeyChatTurnRailTurn) + " " + itoaFast(anchor.Index) + " · " + anchor.Preview)
			if interactive && seat.Clicked() {
				onJump(anchor.Seq)
			}
		}
	})
}

// turnRailMenu is the narrow fallback: one button that opens the turn list.
func turnRailMenu(c *ui.Context, anchors []turnAnchor, m Model, muted, accent ui.Color, onJump func(seq float64)) {
	button := ui.Button(c, i18n.T(i18n.KeyChatTurnRailLabel)+" "+itoaFast(len(anchors))).
		FontSize(11).Padding(0, 8, 0, 8).Height(24).Radius(6).
		Background(ui.Transparent).Border(0.5, muted.Alpha(.4)).TextColor(muted)
	open := ui.Local(button, "turn-rail-menu", func() bool { return false })
	ui.Popover(c, button, open, func() {
		ui.Column(c).Width(260).MaxHeight(320).Gap(2).Padding(4, 4, 4, 4).Children(func() {
			for _, anchor := range anchors {
				anchor := anchor
				item := chrome.MenuRowColors(c, c.Theme().SurfaceHover, c.Theme().SurfacePressed).
					Label(i18n.T(i18n.KeyChatTurnRailJump) + " " + itoaFast(anchor.Index)).MinHeight(32)
				item.Children(func() {
					ui.Text(c, itoaFast(anchor.Index)).Width(20).FontSize(11).TextColor(muted)
					ui.Text(c, anchor.Preview).FontSize(12).Grow(1).MinWidth(0).SingleLine()
				})
				if onJump != nil && item.Clicked() {
					onJump(anchor.Seq)
					*open = false
				}
			}
		})
	})
}

// itoaFast avoids importing strconv just to number rail ticks.
func itoaFast(n int) string {
	if n <= 0 {
		return "0"
	}
	var buf [6]byte
	pos := len(buf)
	for n > 0 && pos > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}
