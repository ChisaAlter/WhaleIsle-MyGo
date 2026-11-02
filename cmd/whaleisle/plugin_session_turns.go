package main

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

var errPluginSessionTurnUnavailable = errors.New("plugin UI: completed turn is no longer available")

// Message IDs are process-local for turn/end. Use only the durable raw event,
// cross-checked with its actual transcript row, never a cached coordinate or a
// display ordinal. In particular TurnSeq is the opening, not the closing seq.
func pluginSessionTurnEnd(msg session.Message) (pluginui.SessionTurnTail, bool) {
	var raw struct {
		Type string   `json:"type"`
		Seq  *float64 `json:"seq"`
		Data struct {
			Turn *float64 `json:"turn"`
		} `json:"data"`
	}
	valid := func(n float64) bool {
		return !math.Signbit(n) && !math.IsNaN(n) && !math.IsInf(n, 0) && n <= 1<<53-1 && n == math.Trunc(n)
	}
	if msg.Streaming || msg.EventType != "turn/end" || json.Unmarshal(msg.Raw, &raw) != nil || raw.Type != "turn/end" || raw.Seq == nil || raw.Data.Turn == nil || !valid(*raw.Seq) || !valid(*raw.Data.Turn) || msg.Seq != *raw.Seq || uint64(*raw.Data.Turn) > uint64(^uint(0)>>1) {
		return pluginui.SessionTurnTail{}, false
	}
	return pluginui.SessionTurnTail{Turn: int(*raw.Data.Turn), Seq: uint64(*raw.Seq)}, true
}

// Duplicate turn boundaries or row seqs are ambiguous even if one occurrence
// happens to carry usable raw data. Do not attach a browser action to either.
func pluginSessionTurnBoundaries(messages []session.Message) map[pluginui.SessionTurnTail]bool {
	turns := map[int]int{}
	seqs := map[float64]int{}
	var tails []pluginui.SessionTurnTail
	for _, msg := range messages {
		seqs[msg.Seq]++
		if tail, ok := pluginSessionTurnEnd(msg); ok {
			turns[tail.Turn]++
			tails = append(tails, tail)
		}
	}
	out := make(map[pluginui.SessionTurnTail]bool, len(tails))
	for _, tail := range tails {
		if turns[tail.Turn] == 1 && seqs[float64(tail.Seq)] == 1 {
			out[tail] = true
		}
	}
	return out
}

func pluginSessionTurnRows(state pluginui.Snapshot, snapshot session.ViewState) map[pluginui.SessionTurnTail][]sessionPluginOutletItem {
	subject := pluginSessionSubject(snapshot)
	if !state.Connected || !state.Observation || state.Page == 0 || state.Revision == 0 || state.Unavailable != "" || state.Subject == nil || *state.Subject != subject || subject.SessionID == "" || subject.Epoch == "" {
		return nil
	}
	boundaries := pluginSessionTurnBoundaries(snapshot.Messages)
	out := map[pluginui.SessionTurnTail][]sessionPluginOutletItem{}
	for _, item := range sessionPluginOutletItems(state.Outlets) {
		row := item.Outlet
		tail := pluginui.SessionTurnTail{Turn: row.Turn, Seq: row.TurnTailSeq}
		if row.Available && row.Handle != "" && row.Slot == "conversation.chat.turnTail" && row.HasTurn && row.HasTurnTail && boundaries[tail] {
			out[tail] = append(out[tail], item)
		}
	}
	return out
}

type sessionPluginTurnRequest struct {
	Page, Revision uint64
	Subject        pluginui.Subject
	Tail           pluginui.SessionTurnTail
	Handle         string
	Nonce          string
}

// invoke keeps the row's captured revision and opaque handle. A newer snapshot
// may refuse this gesture, but can never silently redirect it to another entry.
func (request sessionPluginTurnRequest) invoke(current func() bool, snapshot func() session.ViewState, show func(sessionPluginTurnRequest) error, opened func()) error {
	if !current() {
		return errPluginSessionTurnUnavailable
	}
	now := snapshot()
	if pluginSessionSubject(now) != request.Subject || !pluginSessionTurnBoundaries(now.Messages)[request.Tail] || !current() {
		return errPluginSessionTurnUnavailable
	}
	if err := show(request); err != nil {
		return err
	}
	if !current() {
		return errPluginSessionTurnUnavailable
	}
	opened()
	return nil
}

type sessionPluginTurnPending struct {
	Request sessionPluginTurnRequest
	Until   time.Time
}

func (pending sessionPluginTurnPending) settled(state pluginui.Snapshot, now time.Time) (ready, expired bool) {
	if sessionPluginTurnSelectionReady(state, pending.Request) {
		return true, false
	}
	return false, !state.Connected || state.Subject == nil || *state.Subject != pending.Request.Subject || state.Page != pending.Request.Page || state.Unavailable != "" || !sessionPluginTurnOccurrenceExists(state, pending.Request) || !now.Before(pending.Until)
}

func sessionPluginTurnSelectionReady(state pluginui.Snapshot, request sessionPluginTurnRequest) bool {
	if !state.Connected || state.Subject == nil || *state.Subject != request.Subject || state.Page != request.Page || state.Revision <= request.Revision || state.SessionSurfaceHandle != request.Handle || state.SessionSurfaceRequest != request.Nonce || request.Nonce == "" || state.Surface != "conversation.chat.turnTail" || state.Unavailable != "" {
		return false
	}
	return sessionPluginTurnOccurrenceExists(state, request)
}

func sessionPluginTurnOccurrenceExists(state pluginui.Snapshot, request sessionPluginTurnRequest) bool {
	for _, row := range state.Outlets {
		if row.Slot == "conversation.chat.turnTail" && row.Available && row.Handle == request.Handle && row.HasTurn && row.HasTurnTail && row.Turn == request.Tail.Turn && row.TurnTailSeq == request.Tail.Seq {
			return true
		}
	}
	return false
}

// Surface selection must succeed before a collapsed/expired seat is unmasked.
func selectSessionPluginSurface(selectSurface func() error, unmask func()) error {
	if err := selectSurface(); err != nil {
		return err
	}
	unmask()
	return nil
}

func renderSessionPluginTurnOutlets(c *ui.Context, items []sessionPluginOutletItem, show func(pluginui.SessionOutlet)) {
	if len(items) == 0 || show == nil {
		return
	}
	ui.Column(c).FillWidth().MinWidth(0).Gap(4).Children(func() {
		for _, item := range items {
			label := fmt.Sprintf(i18n.T(i18n.KeySessionPluginsOpenTurnContent), item.Title)
			button := ui.Button(c.Key("session-plugin-turn:"+item.Outlet.Handle), "").Label(label+" · "+item.Attribution).Description(item.Attribution).Tooltip(item.Accessible).MaxWidth(520).FillWidth().MinWidth(0).Padding(6, 10)
			button.Children(func() {
				ui.Column(c).FillWidth().MinWidth(0).Gap(2).Children(func() {
					ui.Text(c, label).FillWidth().MinWidth(0).NoWrap().Ellipsis("…")
					ui.Text(c, item.Attribution).FillWidth().MinWidth(0).FontSize(12)
				})
			})
			if button.Clicked() {
				show(item.Outlet)
			}
		}
	})
}

func (s *mainState) showPluginSessionTurn(p *nativePluginSessionState, owner *pluginui.Owner, request sessionPluginTurnRequest) error {
	s.mu.Lock()
	sessions := p.sessions
	s.mu.Unlock()
	if sessions == nil {
		return errPluginSessionTurnUnavailable
	}
	current := func() bool {
		if !s.pluginSessionCurrent(p) {
			return false
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		return p.owner == owner && p.sessions == sessions && p.subject == request.Subject
	}
	request.Nonce = rand.Text()
	err := request.invoke(current, sessions.Snapshot, func(captured sessionPluginTurnRequest) error {
		if owner.Inline() {
			return owner.RequestMirrorTail(captured.Page, captured.Revision, captured.Subject, captured.Tail, captured.Handle)
		}
		return owner.ShowSessionTurnTail(captured.Page, captured.Revision, captured.Subject, captured.Handle, captured.Tail, captured.Nonce)
	}, func() {
		// Mask(false) does not show a window. The dialog waits for the browser's
		// acknowledgement before Present, so its prior outlet cannot flash here.
		owner.Hide()
		s.mu.Lock()
		p.open, p.error, p.pendingTurn, p.turnExpired = true, "", &sessionPluginTurnPending{Request: request, Until: time.Now().Add(3 * time.Second)}, false
		if owner.Inline() {
			p.open, p.pendingTurn = false, nil
		}
		s.mu.Unlock()
		owner.Mask(false)
	})
	if err != nil && current() {
		owner.Mask(true)
		owner.Show("")
		s.mu.Lock()
		p.open, p.error, p.pendingTurn, p.turnExpired = true, "", nil, true
		s.mu.Unlock()
	}
	return err
}
