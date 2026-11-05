package chat

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// List coordinates include the optional history control; message projections,
// actions and inspector ownership continue to use message coordinates.
func transcriptHistoryRows(m Model) int {
	if ShowLoadEarlier(m.Snapshot) && m.OnLoadEarlier != nil {
		return 1
	}
	return 0
}

func transcriptMessageRow(m Model, index int) int { return index + transcriptHistoryRows(m) }

type transcriptMessageKey struct {
	Scope, Session, ID, Kind, Event string
	Seq                             float64
	Occurrence                      int
}

type transcriptHistoryKey struct{ Scope, Session string }

// Repeated IDs can be separate journal occurrences. Count identical coordinates
// from the tail, so prepending an older duplicate does not rename loaded rows.
func transcriptMessageKeys(m Model, messages []session.Message) []transcriptMessageKey {
	keys := make([]transcriptMessageKey, len(messages))
	counts := make(map[transcriptMessageKey]int, len(messages))
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		key := transcriptMessageKey{Scope: m.Snapshot.PlanHistoryBinding, Session: m.Snapshot.SelectedID, ID: msg.ID, Kind: string(msg.Kind), Event: msg.EventType, Seq: msg.Seq}
		counts[key]++
		key.Occurrence = counts[key]
		keys[i] = key
	}
	return keys
}

func prepareTranscriptListWithKeys(m Model, keys []transcriptMessageKey, list *ui.ListState) []transcriptMessageKey {
	prefix := transcriptHistoryRows(m)
	list.Key = func(row int) any {
		if prefix > 0 && row == 0 {
			return transcriptHistoryKey{m.Snapshot.PlanHistoryBinding, m.Snapshot.SelectedID}
		}
		index := row - prefix
		if index >= 0 && index < len(keys) {
			return keys[index]
		}
		return row
	}
	return keys
}

func prepareTranscriptList(m Model, messages []session.Message, list *ui.ListState) []transcriptMessageKey {
	return prepareTranscriptListWithKeys(m, transcriptMessageKeys(m, messages), list)
}

// ScrollToMessageSeq resolves the current visible/folded projection before
// translating to List coordinates. Callers must pass the same presentation
// preferences and history callback as View, rather than a raw journal index.
// A folded source lands on its visible group. False means the sequence has no
// visible occurrence; no scroll request is queued in that case.
func ScrollToMessageSeq(m Model, seq float64, align ui.Align) bool {
	if m.List == nil {
		return false
	}
	visible := visibleProjectedMessages(m.Snapshot)
	messages, controls := projectTranscript(visible, m)
	for index, msg := range messages {
		if msg.Seq == seq {
			m.List.ScrollTo(transcriptMessageRow(m, index), align)
			return true
		}
	}
	// The rail addresses durable turn/start, which may own no visible row.
	// Resolve that semantic anchor with the same policy as a paged landing.
	for _, turn := range m.Snapshot.HistoryNavigation.Turns {
		if turn.Seq == seq {
			if index := session.HistoryTurnMessageIndex(messages, turn); index >= 0 {
				m.List.ScrollTo(transcriptMessageRow(m, index), align)
				return true
			}
			return false
		}
	}
	for index, msg := range messages {
		if msg.TurnSeq > 0 && msg.TurnSeq == seq {
			m.List.ScrollTo(transcriptMessageRow(m, index), align)
			return true
		}
	}
	// A folded process can own the requested sequence without painting its
	// source row. Land on that group's visible disclosure, never a raw ordinal.
	for _, source := range visible {
		if source.Seq != seq {
			continue
		}
		for index, msg := range messages {
			control, ok := controls[msg.ID]
			if !ok {
				continue
			}
			for _, id := range control.members {
				if id == source.ID {
					m.List.ScrollTo(transcriptMessageRow(m, index), align)
					return true
				}
			}
		}
	}
	return false
}

type transcriptHistoryAnchor struct {
	key   transcriptMessageKey
	index int
	top   float32
	valid bool
}

type transcriptHistoryState struct {
	scope, session string
	list           *ui.ListState
	viewport       ui.Rect
	visible        transcriptHistoryAnchor
	anchor         transcriptHistoryAnchor
	requested      bool
	sawBusy        bool
	activated      bool
}

func (h *transcriptHistoryState) prepare(m Model, messages []session.Message, list *ui.ListState) {
	if h.scope != m.Snapshot.PlanHistoryBinding || h.session != m.Snapshot.SelectedID || h.list != list {
		*h = transcriptHistoryState{scope: m.Snapshot.PlanHistoryBinding, session: m.Snapshot.SelectedID, list: list}
	}
	if !h.requested {
		return
	}
	h.sawBusy = h.sawBusy || m.Snapshot.HistoryPageBusy
	index := -1
	if h.anchor.valid {
		for i, key := range transcriptMessageKeys(m, messages) {
			if key == h.anchor.key {
				index = i
				break
			}
		}
	}
	prepended := index > h.anchor.index
	// A fast page can begin and finish between UI snapshots, including one
	// that advances the journal cursor without adding a visible message. Busy
	// is authoritative on the next build; the input latch only deduplicates
	// the activation's build passes, rather than waiting forever for busy=true.
	settled := !m.Snapshot.HistoryPageBusy && (h.sawBusy || prepended || transcriptHistoryRows(m) == 0 || m.Snapshot.HistoryPageErr != "" || !h.activated)
	if settled {
		if index >= 0 && (prepended || transcriptHistoryRows(m) == 0) {
			list.ScrollToPosition(transcriptMessageRow(m, index), h.anchor.top)
		}
		h.requested, h.sawBusy = false, false
		h.anchor.valid = false
	}
}

func (h *transcriptHistoryState) observeInput(c *ui.Context, list *ui.ListState) {
	focused := list.FocusWithin(c)
	c.ObserveInput(func(ev ui.InputEvent) {
		// A new input is a new activation, even if no busy snapshot was painted.
		h.activated = false
		if !h.requested {
			return
		}
		// Explicit reader input takes over the anchor. Resize and virtual height
		// estimates can also change ScrollOffset, and must not look like input.
		inViewport := h.viewport.Contains(ev.X, ev.Y)
		pointer := inViewport && (ev.Kind == ui.InputScroll || ev.Kind == ui.InputPointerDown && ev.X >= h.viewport.X+h.viewport.W-16)
		key := ev.Kind == ui.InputKeyDown && focused && (ev.Key == ui.KeyUp || ev.Key == ui.KeyDown || ev.Key == ui.KeyHome || ev.Key == ui.KeyEnd || ev.Key == ui.KeyPageUp || ev.Key == ui.KeyPageDown)
		if pointer || key {
			h.anchor.valid = false
		}
	})
}

func (h *transcriptHistoryState) observeViewport(box ui.Rect) {
	h.viewport = box
	h.visible.valid = false
	h.activated = false
}

func (h *transcriptHistoryState) observeMessage(key transcriptMessageKey, index int, box ui.Rect) {
	if !h.visible.valid && box.H > 0 && box.Y+box.H > h.viewport.Y && box.Y < h.viewport.Y+h.viewport.H {
		h.visible = transcriptHistoryAnchor{key: key, index: index, top: box.Y - h.viewport.Y, valid: true}
	}
}

func (h *transcriptHistoryState) load(m Model) {
	if h.requested || h.activated || m.Snapshot.HistoryPageBusy || m.OnLoadEarlier == nil {
		return
	}
	h.requested, h.sawBusy, h.activated = true, false, true
	h.anchor = h.visible
	m.OnLoadEarlier()
}

func transcriptLoadEarlierView(c *ui.Context, m Model, h *transcriptHistoryState, colors theme.Colors, muted ui.Color) {
	label := i18n.T(i18n.KeyChatLoadEarlier)
	if m.Snapshot.HistoryPageBusy {
		label = i18n.T(i18n.KeyChatLoadEarlierBusy)
	}
	ui.Column(c).FillWidth().Padding(8, 12, 0, 12).Gap(4).Children(func() {
		ui.Row(c).FillWidth().Children(func() {
			if stdButton(c, colors, label, buttonGhost, m.Snapshot.HistoryPageBusy || h.requested).Clicked() {
				h.load(m)
			}
		})
		if m.Snapshot.HistoryPageErr != "" {
			ui.Text(c, m.Snapshot.HistoryPageErr).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		}
	})
}

func transcriptRowContent(c *ui.Context, m Model, paint func()) {
	if !m.FullColumnViewport {
		paint()
		return
	}
	axis := m.ContentWidth
	if axis <= 0 {
		axis = 748
	}
	// The same content axis serves history and messages; the native scrollbar
	// stays at the edge of the full conversation column.
	ui.Row(c).FillWidth().Padding(0, 37, 0, 32).Children(func() {
		ui.Box(c).Grow(1).MinWidth(0)
		ui.Column(c).FillWidth().MaxWidth(axis).MinWidth(0).Children(paint)
		ui.Box(c).Grow(1).MinWidth(0)
	})
}
