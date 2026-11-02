package session

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"math"
	"sort"
	"strings"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// HistoryTurn anchors the original Host turnOutline entry at turn/start, never
// at a synthetic chat row or a stream attempt's temporary identity.
type HistoryTurn struct {
	Turn             float64
	Seq              float64
	Prompt, Response string
	Loaded           bool
}

type HistoryNavigationView struct {
	Turns                  []HistoryTurn
	TargetSeq              float64
	HasTarget, Busy, Ready bool
	Request                uint64
	Err                    string
}

type historyNavigationState struct {
	outline            []HistoryTurn
	outlineSeq         float64
	outlineKnown       bool
	baseSeq            float64
	baseKnown          bool
	view               HistoryNavigationView
	cancel             context.CancelFunc
	cachedTurns        []HistoryTurn
	cachedSelectedID   string
	cachedOutlineSeq   float64
	cachedOutlineKnown bool
	cachedBaseSeq      float64
	cachedBaseKnown    bool
	cachedValid        bool
}

func historySafeNumber(v float64) bool { return !math.Signbit(v) && assistantSafeInt(v) && v >= 0 }

func (s *Store) resetHistoryNavigationLocked() {
	s.invalidateViewLocked()
	s.cancelHistoryTurnJumpLocked()
	n := s.historyNavigation
	s.historyNavigation = historyNavigationState{view: HistoryNavigationView{Request: n.view.Request}, outline: n.outline, outlineSeq: n.outlineSeq, outlineKnown: n.outlineKnown}
}

func (s *Store) clearHistoryOutlineLocked() {
	s.invalidateViewLocked()
	s.historyNavigation.outline = nil
	s.historyNavigation.outlineKnown = false
	s.historyNavigation.outlineSeq = 0
}

func (s *Store) cancelHistoryTurnJumpLocked() {
	s.invalidateViewLocked()
	n := &s.historyNavigation
	if n.cancel != nil {
		n.cancel()
		n.cancel = nil
	}
	if n.view.Busy {
		s.historyPageRequest++
		s.historyPageBusy = false
	}
	n.view.Request++
	n.view.HasTarget = false
	n.view.Busy = false
	n.view.Ready = false
	n.view.Err = ""
}

func (s *Store) noteHistoryBaseLocked(seq float64) {
	s.invalidateViewLocked()
	if !historySafeNumber(seq) {
		return
	}
	n := &s.historyNavigation
	if !n.baseKnown || seq < n.baseSeq {
		n.baseSeq = seq
		n.baseKnown = true
	}
}

func (s *Store) historyBeforeSeqLocked() float64 {
	if s.historyNavigation.baseKnown {
		return s.historyNavigation.baseSeq
	}
	return oldestPersistenceSeq(s.messages)
}

func (s *Store) applyHistoryOutlineValuesLocked(values json.RawMessage, seq float64) {
	var all map[string]json.RawMessage
	if json.Unmarshal(values, &all) != nil {
		return
	}
	s.applyHistoryOutlineLocked(all["turnOutline"], seq)
}

func (s *Store) applyHistoryOutlineLocked(raw json.RawMessage, seq float64) {
	s.invalidateViewLocked()
	n := &s.historyNavigation
	if s.selectedID == "" || !historySafeNumber(seq) || (n.outlineKnown && seq < n.outlineSeq) {
		return
	}
	var entries []json.RawMessage
	if len(raw) > 0 && string(raw) != "null" && json.Unmarshal(raw, &entries) != nil {
		return
	}
	byTurn := map[float64]HistoryTurn{}
	for _, entry := range entries {
		var value struct {
			Turn, Seq        *float64
			Prompt, Response json.RawMessage
		}
		if json.Unmarshal(entry, &value) != nil || value.Turn == nil || value.Seq == nil || !historySafeNumber(*value.Turn) || !historySafeNumber(*value.Seq) {
			continue
		}
		row := HistoryTurn{Turn: *value.Turn, Seq: *value.Seq}
		_ = json.Unmarshal(value.Prompt, &row.Prompt)
		_ = json.Unmarshal(value.Response, &row.Response)
		row.Prompt = historyPreview(row.Prompt, 50)
		row.Response = historyPreview(row.Response, 120)
		byTurn[row.Turn] = row
	}
	n.outline = nil
	for _, row := range byTurn {
		n.outline = append(n.outline, row)
	}
	sort.Slice(n.outline, func(i, j int) bool { return n.outline[i].Turn < n.outline[j].Turn })
	n.outlineSeq = seq
	n.outlineKnown = true
	n.cachedValid = false
}

func (s *Store) applyHistoryOutlineControlLocked(raw json.RawMessage) {
	s.invalidateViewLocked()
	var f struct {
		Type, SessionID, Key string
		Seq                  float64
		Value                json.RawMessage
	}
	if json.Unmarshal(raw, &f) != nil || f.Type != "projection" || f.SessionID != s.selectedID || f.Key != "turnOutline" {
		return
	}
	s.applyHistoryOutlineLocked(f.Value, f.Seq)
}

func historyPreview(body string, limit int) string {
	// Loaded message bodies may be large; normalize only a bounded prefix.
	text := make([]rune, 0, limit*2)
	clipped := false
	for _, r := range body {
		if len(text) == limit*2 {
			clipped = true
			break
		}
		text = append(text, r)
	}
	text = []rune(strings.Join(strings.Fields(string(text)), " "))
	if len(text) > limit-1 {
		return string(text[:limit-1]) + "…"
	}
	if clipped {
		return string(text) + "…"
	}
	return string(text)
}

func historyMessageTurn(m Message) (float64, bool) {
	if len(m.Raw) == 0 {
		return 0, false
	}
	if turn, _, known, cached := LookupCachedCoordinates(m.Seq, m.Raw); cached {
		if !known || !historySafeNumber(turn) {
			return 0, false
		}
		return turn, true
	}
	if !bytes.Contains(m.Raw, []byte("turn")) && !bytes.Contains(m.Raw, []byte(`\u`)) {
		return 0, false
	}
	var e struct{ Data struct{ Turn *float64 } }
	if json.Unmarshal(m.Raw, &e) != nil || e.Data.Turn == nil || !historySafeNumber(*e.Data.Turn) {
		return 0, false
	}
	return *e.Data.Turn, true
}

func (s *Store) historyMessageTurnLocked(m Message) (float64, bool) {
	if m.Seq > 0 && s.trajTimingParsed != nil {
		if p, ok := s.trajTimingParsed[m.Seq]; ok && bytes.Equal(p.raw, m.Raw) {
			if !p.ok || p.event.Data.Turn == nil || !historySafeNumber(*p.event.Data.Turn) {
				return 0, false
			}
			return *p.event.Data.Turn, true
		}
	}
	if m.Seq > 0 && s.trajTimingCache != nil {
		if entry, ok := s.trajTimingCache[m.Seq]; ok && bytes.Equal([]byte(entry.raw), m.Raw) {
			if !entry.parsed.ok || entry.parsed.event.Data.Turn == nil || !historySafeNumber(*entry.parsed.event.Data.Turn) {
				return 0, false
			}
			return *entry.parsed.event.Data.Turn, true
		}
	}
	return historyMessageTurn(m)
}

func (s *Store) historyNavigationSnapshotLocked() HistoryNavigationView {
	v := s.historyNavigation.view
	v.Turns = nil
	if s.selectedID == "" {
		return v
	}
	for _, it := range s.items {
		if it.SessionID == s.selectedID && it.Origin == "subagent" {
			return v
		}
	}
	n := &s.historyNavigation
	if n.cachedValid && s.historyVerifiedLocked() && n.cachedSelectedID == s.selectedID &&
		n.cachedOutlineSeq == n.outlineSeq && n.cachedOutlineKnown == n.outlineKnown &&
		n.cachedBaseSeq == n.baseSeq && n.cachedBaseKnown == n.baseKnown {
		v.Turns = cloneSlice(n.cachedTurns)
		return v
	}
	byTurn := map[float64]HistoryTurn{}
	for _, row := range s.historyNavigation.outline {
		byTurn[row.Turn] = row
	}
	if s.historyNavigation.baseKnown {
		for _, m := range s.messages {
			if m.Streaming || m.Seq <= 0 {
				continue
			}
			turn, ok := s.historyMessageTurnLocked(m)
			if !ok {
				continue
			}
			row, exists := byTurn[turn]
			if !exists {
				if m.TurnSeq <= 0 {
					continue
				}
				row = HistoryTurn{Turn: turn, Seq: m.TurnSeq}
			}
			// A middle-of-turn window does not establish that its opening is loaded.
			row.Loaded = s.historyNavigation.baseSeq <= row.Seq
			if row.Prompt == "" && isHumanMessage(m) {
				row.Prompt = historyPreview(m.Body, 50)
			}
			if row.Response == "" && m.Role == "assistant" {
				row.Response = historyPreview(m.Body, 120)
			}
			byTurn[turn] = row
		}
	}
	for _, row := range byTurn {
		v.Turns = append(v.Turns, row)
	}
	sort.Slice(v.Turns, func(i, j int) bool { return v.Turns[i].Turn < v.Turns[j].Turn })
	n.cachedTurns = cloneSlice(v.Turns)
	n.cachedSelectedID = s.selectedID
	n.cachedOutlineSeq = n.outlineSeq
	n.cachedOutlineKnown = n.outlineKnown
	n.cachedBaseSeq = n.baseSeq
	n.cachedBaseKnown = n.baseKnown
	n.cachedValid = true
	return v
}

// JumpToHistoryTurn accepts only a current known outline/loaded turn. Paging is
// read-only and retains the captured durable cut while live output continues.
func (s *Store) JumpToHistoryTurn(ctx context.Context, sessionID string, expectedRequest uint64, seq float64) {
	if s == nil || ctx.Err() != nil || !historySafeNumber(seq) {
		return
	}
	s.mu.Lock()
	if s.selectedID != sessionID || sessionID == "" || s.client == nil || s.historyNavigation.view.Request != expectedRequest {
		s.mu.Unlock()
		return
	}
	n := &s.historyNavigation
	known := false
	for _, row := range s.historyNavigationSnapshotLocked().Turns {
		if row.Seq == seq {
			known = true
			break
		}
	}
	if !known || (s.historyPageBusy && !n.view.Busy) || (n.view.Busy && n.view.TargetSeq == seq) {
		s.mu.Unlock()
		return
	}
	s.cancelHistoryTurnJumpLocked()
	n.view.HasTarget = true
	n.view.TargetSeq = seq
	before, through := s.historyBeforeSeqLocked(), s.historyThroughSeq
	if n.baseKnown && before <= seq {
		n.view.Ready = true
		s.mu.Unlock()
		s.fire()
		return
	}
	if !s.historyHasMore || before <= 0 || through <= 0 {
		n.view.Err = "The requested turn is not available in the current history window."
		s.mu.Unlock()
		s.fire()
		return
	}
	n.view.Busy = true
	s.historyPageBusy = true
	s.historyPageErr = nil
	s.historyPageRequest++
	request, pageRequest := n.view.Request, s.historyPageRequest
	client, generation := s.client, s.followGeneration
	fctx, cancel := context.WithCancel(ctx)
	n.cancel = cancel
	s.mu.Unlock()
	s.fire()
	go s.runHistoryTurnJump(fctx, client, sessionID, generation, request, pageRequest, through, before, seq)
}

func (s *Store) historyJumpOwnedLocked(client *dsh.Client, id string, generation, request, pageRequest uint64) bool {
	return s.historyPageOwnedLocked(client, id, generation, pageRequest) && s.historyNavigation.view.Request == request && s.historyNavigation.view.Busy
}

func (s *Store) runHistoryTurnJump(ctx context.Context, client *dsh.Client, id string, generation, request, pageRequest uint64, through, before, target float64) {
	var pages [][]api.SessionPageRecordsItem
	hasMore := true
	var resultErr error
	for hasMore && before > target && ctx.Err() == nil {
		// Check ownership before each request, as well as before any publication.
		s.mu.Lock()
		owned := s.historyJumpOwnedLocked(client, id, generation, request, pageRequest)
		s.mu.Unlock()
		if !owned {
			return
		}
		maxMessages := float64(500)
		cursor := before
		req := api.SessionPageRequest{Address: sessionAddressJSON(id), ThroughSeq: through, BeforeSeq: &cursor, MaxMessages: &maxMessages, TurnWindow: &api.SessionFollowRequestTurnWindow{MinMessages: 200, MinTurns: 2}}
		var out api.SessionPage
		pageCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		resultErr = client.Call(pageCtx, "session/page", map[string]any{"request": req}, &out)
		cancel()
		if resultErr != nil {
			break
		}
		if resultErr = validateHistoryPage(out.Records, through, before); resultErr != nil {
			break
		}
		next := before
		var accepted []api.SessionPageRecordsItem
		for _, rec := range out.Records {
			if rec.Type != "event" || !historySafeNumber(rec.Event.Seq) || rec.Event.Seq >= before || rec.Event.Seq > through {
				continue
			}
			accepted = append(accepted, rec)
			if rec.Event.Seq < next {
				next = rec.Event.Seq
			}
		}
		if next >= before {
			resultErr = errors.New("History did not advance; retry to load this turn.")
			break
		}
		sort.SliceStable(accepted, func(i, j int) bool { return accepted[i].Event.Seq < accepted[j].Event.Seq })
		pages = append(pages, accepted)
		before = next
		hasMore = out.HasMore
	}
	s.mu.Lock()
	if !s.historyJumpOwnedLocked(client, id, generation, request, pageRequest) {
		s.mu.Unlock()
		return
	}
	n := &s.historyNavigation
	n.view.Busy = false
	s.historyPageBusy = false
	s.invalidateViewLocked()
	if ctx.Err() != nil {
		resultErr = ctx.Err()
	}
	if n.cancel != nil {
		n.cancel()
		n.cancel = nil
	}
	// Cancellation never publishes pages or resurrects a queued landing.
	if errors.Is(resultErr, context.Canceled) {
		n.view.HasTarget = false
		n.view.Ready = false
		s.mu.Unlock()
		s.fire()
		return
	}
	// Prepend in request order: each successive page is older. Only this final
	// commit changes the displayed window, preserving live attempts and drafts.
	for _, records := range pages {
		s.prependPageRecordsLocked(records)
	}
	if len(pages) > 0 {
		s.historyHasMore = hasMore
	}
	if resultErr == nil && before > target {
		resultErr = errors.New("The requested turn is no longer available in history.")
	}
	if resultErr != nil {
		n.view.Err = resultErr.Error()
	} else {
		n.view.Ready = true
	}
	s.mu.Unlock()
	s.fire()
}

// CancelHistoryTurnJump cancels only navigation-owned reads, never a model run.
func (s *Store) CancelHistoryTurnJump(id string, request uint64) bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	if s.selectedID != id || s.historyNavigation.view.Request != request {
		s.mu.Unlock()
		return false
	}
	s.cancelHistoryTurnJumpLocked()
	s.mu.Unlock()
	s.fire()
	return true
}

// ConsumeHistoryTurnLanding makes each explicit selection a one-shot UI scroll.
func (s *Store) ConsumeHistoryTurnLanding(id string, request uint64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := &s.historyNavigation
	if s.selectedID != id || n.view.Request != request || !n.view.Ready {
		return false
	}
	n.view.Ready = false
	s.invalidateViewLocked()
	return true
}

// HistoryTurnMessageIndex resolves against the final displayed projection. If
// an empty/filtered turn has no row, use the next durable turn as the original.
func HistoryTurnMessageIndex(messages []Message, turn HistoryTurn) int {
	fallback := -1
	for i, m := range messages {
		if m.Streaming || m.Seq <= 0 {
			continue
		}
		if m.TurnSeq == turn.Seq && m.TurnSeq > 0 {
			return i
		}
		if n, ok := historyMessageTurn(m); ok && n == turn.Turn && m.Seq >= turn.Seq {
			return i
		}
		if fallback < 0 && m.Seq >= turn.Seq {
			fallback = i
		}
	}
	return fallback
}
