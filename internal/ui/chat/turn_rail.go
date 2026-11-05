package chat

import (
	"fmt"
	"sort"
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
	Seq      float64
	Index    int
	Preview  string
	Response string
	Status   rowStatus
	Loaded   bool
}

// turnAnchors lists durable turn starts, retaining the user-row entry points of
// imported transcripts that have no turn coordinates.
func turnAnchors(messages []session.Message) []turnAnchor {
	// A steer/user row inside an existing turn is not a second Turn. Prefer
	// the durable turn/start coordinates carried by the current projection.
	bySeq := map[float64]*turnAnchor{}
	for _, msg := range messages {
		if msg.TurnSeq <= 0 {
			continue
		}
		anchor := bySeq[msg.TurnSeq]
		if anchor == nil {
			anchor = &turnAnchor{Seq: msg.TurnSeq, Loaded: true}
			bySeq[msg.TurnSeq] = anchor
		}
		if anchor.Preview == "" && (msg.Role == "user" || msg.ContextWaking) {
			anchor.Preview = messagePreviewText(msg.Body, 72)
		}
		if anchor.Response == "" && msg.Role == "assistant" && msg.Kind == session.MessageKindText {
			anchor.Response = messagePreviewText(msg.Body, 120)
		}
	}
	if len(bySeq) > 0 {
		out := make([]turnAnchor, 0, len(bySeq))
		for _, anchor := range bySeq {
			out = append(out, *anchor)
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Seq < out[j].Seq })
		for i := range out {
			out[i].Index = i + 1
			if out[i].Preview == "" {
				out[i].Preview = i18n.T(i18n.KeyChatTurnRailEmpty)
			}
		}
		return out
	}
	// Older imported transcripts without turn coordinates keep their existing
	// user-message navigation rather than losing the entry point.
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
	if len(snap.HistoryNavigation.Turns) == 0 {
		return turnAnchors(snap.Messages)
	}
	out := make([]turnAnchor, 0, len(snap.HistoryNavigation.Turns))
	for _, t := range snap.HistoryNavigation.Turns {
		preview := messagePreviewText(t.Prompt, 72)
		if preview == "" {
			preview = i18n.T(i18n.KeyChatTurnRailEmpty)
		}
		out = append(out, turnAnchor{
			Seq:      t.Seq,
			Index:    int(t.Turn),
			Preview:  preview,
			Response: messagePreviewText(t.Response, 120),
			Loaded:   t.Loaded,
		})
	}
	return out
}

// Read the same folded transcript coordinates as the visible list, not the
// journal's ordinal or the latest navigation request (the reader may scroll).
func activeTurnAnchor(m Model, anchors []turnAnchor, memo *transcriptViewMemo) int {
	if len(anchors) == 0 {
		return -1
	}
	if m.List == nil || m.List.AtEnd() {
		return len(anchors) - 1
	}
	first, last := m.List.Visible()
	if last < first {
		return -1
	}
	messages, _, _, _ := memo.resolve(m)
	index := max(0, first-transcriptHistoryRows(m))
	if index >= len(messages) {
		return -1
	}
	seq := messages[index].TurnSeq
	if seq <= 0 {
		seq = messages[index].Seq
	}
	active := -1
	for i, anchor := range anchors {
		if anchor.Seq > seq {
			break
		}
		active = i
	}
	return active
}

type turnRailState struct {
	list      ui.ListState
	activeSeq float64
	count     int
	placed    bool
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
	projection := ui.Local(c.Root(), "turn-rail-projection:"+m.Snapshot.PlanHistoryBinding+":"+m.Snapshot.SelectedID, func() transcriptViewMemo { return transcriptViewMemo{} })
	current := activeTurnAnchor(m, anchors, projection)
	if available < turnRailNarrowMin {
		turnRailMenu(c, anchors, m, current, muted, onJump)
		return
	}
	_, viewportHeight := c.Size()
	if m.List != nil {
		if box := m.List.Bounds(c); box.H > 0 {
			viewportHeight = box.H
		}
	}
	railHeight := min(float32(len(anchors))*turnRailSeatHeight+12, max(float32(12), viewportHeight))
	rail := ui.Column(c).Key("turn-rail:" + m.Snapshot.PlanHistoryBinding + ":" + m.Snapshot.SelectedID).Label(i18n.T(i18n.KeyChatTurnRailLabel)).Width(turnRailWidth).Height(railHeight).MinHeight(0)
	state := ui.Local(rail, "rail", func() turnRailState { return turnRailState{} })
	state.list.Key = func(index int) any { return anchors[index].Seq }
	if current >= 0 && !rail.Hovered() {
		if !state.placed {
			state.list.ScrollTo(current, ui.Center)
		} else if state.activeSeq != anchors[current].Seq || state.count != len(anchors) {
			state.list.ScrollIntoView(current)
		}
		state.activeSeq, state.count, state.placed = anchors[current].Seq, len(anchors), true
	}
	rail.Children(func() {
		ui.List(c, &state.list, len(anchors)).Label(i18n.T(i18n.KeyChatTurnRailLabel)).Width(turnRailWidth).FillHeight().MinHeight(0).Padding(6, 0).Rows(func(row ui.ListRow) {
			anchor := anchors[row.Index]
			active := row.Index == current
			// The seat is the hit target: a 10 DIP row holding the 2 DIP tick, so
			// a multi-turn transcript gives each tick a comfortable right-aligned target.
			seat := ui.Row(row.Context).Key("turn-rail-tick:"+itoaFast(anchor.Index)).
				Width(turnRailWidth).Height(turnRailSeatHeight).Padding(0, 4, 0, 4).Justify(ui.End).AlignItems(ui.Center)
			interactive := onJump != nil
			if interactive {
				jumpKey := i18n.KeyChatTurnRailJump
				if !anchor.Loaded {
					jumpKey = i18n.KeyChatTurnRailJumpLoad
				}
				seat.Label(fmt.Sprintf(i18n.T(jumpKey), anchor.Index)).Focusable().Role(ui.RoleButton)
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
			if !anchor.Loaded {
				barColor = barColor.Alpha(.5)
			}
			if m.Snapshot.HistoryNavigation.Busy && m.Snapshot.HistoryNavigation.TargetSeq == anchor.Seq {
				barColor = chrome.Color(colors.Accent)
			}
			seat.Children(func() {
				ui.Box(row.Context).Width(barWidth).Height(barHeight).Radius(999).Background(barColor)
			})
			preview := i18n.T(i18n.KeyChatTurnRailTurn) + " " + itoaFast(anchor.Index) + " · " + anchor.Preview
			if anchor.Response != "" {
				preview += "\n" + anchor.Response
			}
			seat.Tooltip(preview)
			if interactive && seat.Clicked() {
				onJump(anchor.Seq)
			}
		})
	})
}

// turnRailMenu is the narrow fallback: one button that opens the turn list.
func turnRailMenu(c *ui.Context, anchors []turnAnchor, m Model, current int, muted ui.Color, onJump func(seq float64)) {
	button := ui.Button(c, i18n.T(i18n.KeyChatTurnRailLabel)+" "+itoaFast(len(anchors))).
		FontSize(11).Padding(0, 8, 0, 8).Height(24).Radius(6).
		Background(ui.Transparent).Border(0.5, muted.Alpha(.4)).TextColor(muted)
	open := ui.Local(button, "turn-rail-menu", func() bool { return false })
	ui.Popover(c, button, open, func() {
		ui.Scroll(c).Width(260).MaxHeight(320).Children(func() {
			ui.Column(c).FillWidth().Gap(2).Padding(4).Children(func() {
				for index, anchor := range anchors {
					anchor := anchor
					jumpKey := i18n.KeyChatTurnRailJump
					if !anchor.Loaded {
						jumpKey = i18n.KeyChatTurnRailJumpLoad
					}
					item := chrome.MenuRowColors(c, c.Theme().SurfaceHover, c.Theme().SurfacePressed).
						Label(fmt.Sprintf(i18n.T(jumpKey), anchor.Index)).Selected(index == current).MinHeight(32)
					if index == current {
						item.Background(c.Theme().SurfaceHover)
					}
					item.Tooltip(anchor.Response)
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
